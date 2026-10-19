// Package capture owns Markdown source and local delivery custody, independently
// of legacy vault documents and of any desktop transport.
package capture

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	_ "modernc.org/sqlite" // Register the SQLite driver for the independent capture database.
)

const SchemaVersion = 1
const RendererVersion = "escaped-source-v1"
const MaxMarkdownBytes = 1 << 20

var ErrConflict = errors.New("draft changed; reload before saving")

// Draft preserves the exact UTF-8 source. HTML is a disposable, safely escaped
// source preview; neither the renderer nor a rich editor may replace Markdown.
type Draft struct {
	CaptureID       string      `json:"captureID"`
	SchemaVersion   int         `json:"schemaVersion"`
	Revision        int         `json:"revision"`
	Markdown        string      `json:"markdown"`
	ContentHash     string      `json:"contentHash"`
	HTML            string      `json:"html"`
	RendererVersion string      `json:"rendererVersion"`
	CreatedAt       string      `json:"createdAt"`
	UpdatedAt       string      `json:"updatedAt"`
	QuickAdd        string      `json:"quickAdd"`
	Fields          QuickFields `json:"fields"`
	References      []Reference `json:"references"`
}

type Store struct{ db *sql.DB }

func Open(ctx context.Context, dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "capture.db")
	// The path is native configuration, never a renderer-supplied filename.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600) // #nosec G304 -- fixed filename under native config directory
	if err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.ExecContext(ctx, `PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL;
CREATE TABLE IF NOT EXISTS capture_drafts (
 capture_id TEXT PRIMARY KEY, schema_version INTEGER NOT NULL, revision INTEGER NOT NULL,
 markdown TEXT NOT NULL, content_hash TEXT NOT NULL, html TEXT NOT NULL,
 renderer_version TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
 );
 CREATE TABLE IF NOT EXISTS capture_inputs (capture_id TEXT PRIMARY KEY, schema_version INTEGER NOT NULL DEFAULT 1, quick_add TEXT NOT NULL);`)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func hash(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

func project(d *Draft) {
	d.ContentHash = hash(d.Markdown)
	d.HTML = "<pre>" + html.EscapeString(d.Markdown) + "</pre>"
	d.RendererVersion = RendererVersion
	d.Fields = ParseQuickAdd(d.QuickAdd)
	d.References = References(d.Markdown)
}

func (s *Store) Create(ctx context.Context) (Draft, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	d := Draft{CaptureID: uuid.NewString(), SchemaVersion: SchemaVersion, Revision: 1, CreatedAt: now, UpdatedAt: now}
	project(&d)
	_, err := s.db.ExecContext(ctx, `INSERT INTO capture_drafts VALUES (?,?,?,?,?,?,?,?,?)`, d.CaptureID, d.SchemaVersion, d.Revision, d.Markdown, d.ContentHash, d.HTML, d.RendererVersion, d.CreatedAt, d.UpdatedAt)
	return d, err
}

const draftSelect = `SELECT d.*, COALESCE(i.quick_add, ''), COALESCE(i.schema_version,1) FROM capture_drafts d LEFT JOIN capture_inputs i ON i.capture_id=d.capture_id`

func scan(row interface{ Scan(...any) error }) (Draft, error) {
	var d Draft
	var inputVersion int
	err := row.Scan(&d.CaptureID, &d.SchemaVersion, &d.Revision, &d.Markdown, &d.ContentHash, &d.HTML, &d.RendererVersion, &d.CreatedAt, &d.UpdatedAt, &d.QuickAdd, &inputVersion)
	if err != nil {
		return d, err
	}
	if inputVersion != 1 {
		return d, fmt.Errorf("unsupported capture input schema %d", inputVersion)
	}
	if d.SchemaVersion != SchemaVersion {
		return d, fmt.Errorf("unsupported capture schema %d", d.SchemaVersion)
	}
	// Cache provenance is verified on every read. Rebuild without rewriting source.
	project(&d)
	return d, nil
}

func (s *Store) Get(ctx context.Context, id string) (Draft, error) {
	return scan(s.db.QueryRowContext(ctx, draftSelect+` WHERE d.capture_id=?`, id))
}

func (s *Store) List(ctx context.Context) ([]Draft, error) {
	rows, err := s.db.QueryContext(ctx, draftSelect+` ORDER BY d.updated_at DESC, d.capture_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	drafts := []Draft{}
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			return nil, e
		}
		drafts = append(drafts, d)
	}
	return drafts, rows.Err()
}

func (s *Store) Save(ctx context.Context, id string, revision int, markdown string) (Draft, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return Draft{}, err
	}
	return s.SaveContent(ctx, id, revision, markdown, existing.QuickAdd)
}

// SaveContent atomically persists exact source and the explicitly separate quick
// input. Its CAS covers both; Markdown is never scanned for Quick Add tokens.
func (s *Store) SaveContent(ctx context.Context, id string, revision int, markdown, quickAdd string) (Draft, error) {
	if !utf8.ValidString(markdown) || len(markdown) > MaxMarkdownBytes || !utf8.ValidString(quickAdd) || len(quickAdd) > 4096 {
		return Draft{}, errors.New("draft must be valid UTF-8, Markdown at most 1 MiB and quick input at most 4096 bytes")
	}
	d := Draft{Markdown: markdown}
	project(&d)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Draft{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE capture_drafts SET markdown=?,content_hash=?,html=?,renderer_version=?,updated_at=?,revision=revision+1 WHERE capture_id=? AND revision=? AND schema_version=?`, markdown, d.ContentHash, d.HTML, d.RendererVersion, time.Now().UTC().Format(time.RFC3339Nano), id, revision, SchemaVersion)
	if err != nil {
		return Draft{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Draft{}, err
	}
	if n != 1 {
		return Draft{}, ErrConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO capture_inputs(capture_id,quick_add) VALUES(?,?) ON CONFLICT(capture_id) DO UPDATE SET quick_add=excluded.quick_add`, id, quickAdd)
	if err != nil {
		return Draft{}, err
	}
	got, err := scan(tx.QueryRowContext(ctx, draftSelect+` WHERE d.capture_id=?`, id))
	if err != nil {
		return Draft{}, err
	}
	if err = tx.Commit(); err != nil {
		return Draft{}, err
	}
	return got, nil
}
