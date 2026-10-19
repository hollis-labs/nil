package capture

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Preset stores a raw editable Quick Add default line and its typed projection.
// It is copied once at draft initiation; no live preset lookup occurs on save.
type Preset struct {
	ID       string      `json:"id"`
	Revision int         `json:"revision"`
	Name     string      `json:"name"`
	QuickAdd string      `json:"quickAdd"`
	Fields   QuickFields `json:"fields"`
}

type PresetSelection struct {
	Presets    []Preset `json:"presets"`
	SelectedID string   `json:"selectedID"`
}

// ContextSnapshot records explicit lack of context until native 0080 sampling.
// No current activity session is inferred from localStorage or reconnection.
type ContextSnapshot struct {
	State      string `json:"state"`
	ObservedAt string `json:"observedAt"`
	BindingID  string `json:"bindingID,omitempty"`
	DraftAt    string `json:"draftAt,omitempty"`
	DeviceID   string `json:"deviceID,omitempty"`
	Origin     string `json:"origin,omitempty"`
}

type DraftSnapshot struct {
	SchemaVersion    int             `json:"schemaVersion"`
	Preset           *Preset         `json:"preset"`
	AppliedBy        string          `json:"appliedBy"`
	ResolvedDefaults QuickFields     `json:"resolvedDefaults"`
	Context          ContextSnapshot `json:"context"`
}

const presetSchema = `
 CREATE TABLE IF NOT EXISTS capture_presets(id TEXT PRIMARY KEY,revision INTEGER NOT NULL,name TEXT NOT NULL,quick_add TEXT NOT NULL,deleted INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE IF NOT EXISTS capture_preferences(key TEXT PRIMARY KEY,value TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS capture_snapshots(capture_id TEXT PRIMARY KEY,snapshot_json TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS capture_local_queue(capture_id TEXT PRIMARY KEY,envelope_json TEXT NOT NULL,envelope_hash TEXT NOT NULL);
 CREATE TRIGGER IF NOT EXISTS immutable_capture_envelope BEFORE UPDATE ON capture_local_queue BEGIN SELECT RAISE(ABORT,'queued envelopes are immutable'); END;
`

func (s *Store) SavePreset(ctx context.Context, id string, revision int, name, quickAdd string) (Preset, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 || !utf8.ValidString(name) || len(quickAdd) > 4096 || !utf8.ValidString(quickAdd) {
		return Preset{}, errors.New("preset needs a name up to 100 bytes and UTF-8 defaults up to 4096 bytes")
	}
	if id == "" {
		p := Preset{ID: uuid.NewString(), Revision: 1, Name: name, QuickAdd: quickAdd, Fields: ParseQuickAdd(quickAdd)}
		_, err := s.db.ExecContext(ctx, `INSERT INTO capture_presets(id,revision,name,quick_add) VALUES(?,?,?,?)`, p.ID, p.Revision, p.Name, p.QuickAdd)
		return p, err
	}
	var p Preset
	err := s.db.QueryRowContext(ctx, `UPDATE capture_presets SET revision=revision+1,name=?,quick_add=? WHERE id=? AND revision=? AND deleted=0 RETURNING id,revision,name,quick_add`, name, quickAdd, id, revision).Scan(&p.ID, &p.Revision, &p.Name, &p.QuickAdd)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrConflict
	}
	p.Fields = ParseQuickAdd(p.QuickAdd)
	return p, err
}

func (s *Store) Presets(ctx context.Context) (PresetSelection, error) {
	out := PresetSelection{Presets: []Preset{}}
	rows, err := s.db.QueryContext(ctx, `SELECT id,revision,name,quick_add FROM capture_presets WHERE deleted=0 ORDER BY name,id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var p Preset
		if err = rows.Scan(&p.ID, &p.Revision, &p.Name, &p.QuickAdd); err != nil {
			_ = rows.Close()
			return out, err
		}
		p.Fields = ParseQuickAdd(p.QuickAdd)
		out.Presets = append(out.Presets, p)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return out, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT value FROM capture_preferences WHERE key='selected_preset'`).Scan(&out.SelectedID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return out, err
}

func (s *Store) SelectPreset(ctx context.Context, id string) error {
	// One statement validates a live selection. This affects future drafts only.
	result, err := s.db.ExecContext(ctx, `INSERT INTO capture_preferences(key,value) SELECT 'selected_preset',? WHERE ?='' OR EXISTS(SELECT 1 FROM capture_presets WHERE id=? AND deleted=0) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, id, id, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("preset no longer available")
	}
	return nil
}

func initialSnapshot(ctx context.Context, tx *sql.Tx, at string) (DraftSnapshot, error) {
	snap := DraftSnapshot{SchemaVersion: 1, AppliedBy: "none", ResolvedDefaults: ParseQuickAdd(""), Context: ContextSnapshot{State: "unavailable", ObservedAt: at}}
	var p Preset
	err := tx.QueryRowContext(ctx, `SELECT p.id,p.revision,p.name,p.quick_add FROM capture_presets p JOIN capture_preferences pref ON pref.value=p.id AND pref.key='selected_preset' WHERE p.deleted=0`).Scan(&p.ID, &p.Revision, &p.Name, &p.QuickAdd)
	if errors.Is(err, sql.ErrNoRows) {
		return snap, nil
	}
	if err != nil {
		return snap, err
	}
	p.Fields = ParseQuickAdd(p.QuickAdd)
	snap.Preset = &p
	snap.AppliedBy = "manual"
	snap.ResolvedDefaults = p.Fields
	return snap, nil
}

// Envelope is a versioned local custody record, not a Basalisk wire request.
// 0080 must persist the finalized authorized request before any network send.
type Envelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Draft         Draft  `json:"draft"`
	Content       string `json:"content"`
	SubmittedAt   string `json:"submittedAt"`
}

func (s *Store) Queue(ctx context.Context, id string, revision int) (Envelope, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Envelope{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var raw, storedHash string
	err = tx.QueryRowContext(ctx, `SELECT envelope_json,envelope_hash FROM capture_local_queue WHERE capture_id=?`, id).Scan(&raw, &storedHash)
	if err == nil {
		if hash(raw) != storedHash {
			return Envelope{}, errors.New("queued envelope integrity check failed")
		}
		var env Envelope
		err = json.Unmarshal([]byte(raw), &env)
		if err == nil && (env.SchemaVersion != 1 || env.Draft.CaptureID != id) {
			err = errors.New("unsupported or mismatched queued envelope")
		}
		if err == nil && env.Draft.Revision != revision {
			err = ErrConflict
		}
		return env, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Envelope{}, err
	}
	d, err := scan(tx.QueryRowContext(ctx, draftSelect+` WHERE d.capture_id=?`, id))
	if err != nil {
		return Envelope{}, err
	}
	if d.Revision != revision {
		return Envelope{}, ErrConflict
	}
	content := d.Markdown
	if strings.TrimSpace(content) == "" {
		content = d.Fields.Title
	}
	if strings.TrimSpace(content) == "" {
		return Envelope{}, errors.New("write a title or Markdown before queuing")
	}
	env := Envelope{SchemaVersion: 1, Draft: d, Content: content, SubmittedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	data, err := json.Marshal(env)
	if err != nil {
		return env, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO capture_local_queue(capture_id,envelope_json,envelope_hash) VALUES(?,?,?)`, id, string(data), hash(string(data)))
	if err != nil {
		return env, err
	}
	return env, tx.Commit()
}
