package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const chatSchema = `
CREATE TABLE IF NOT EXISTS chat_sessions (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  vault_id   TEXT NOT NULL,
  started_at TEXT NOT NULL DEFAULT (datetime('now')),
  ended_at   TEXT DEFAULT NULL,
  dry_run    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS chat_messages (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
  role       TEXT NOT NULL CHECK(role IN ('user','assistant','system')),
  content    TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS action_proposals (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id   INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
  message_id   INTEGER DEFAULT NULL REFERENCES chat_messages(id) ON DELETE SET NULL,
  action_type  TEXT NOT NULL CHECK(action_type IN ('create','update','delete')),
  item_type    TEXT NOT NULL CHECK(item_type IN ('todo','note')),
  vault_id     TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  diff_json    TEXT DEFAULT NULL,
  status       TEXT NOT NULL DEFAULT 'pending'
               CHECK(status IN ('pending','approved','denied','executed','failed','dry_run')),
  created_at   TEXT NOT NULL DEFAULT (datetime('now')),
  resolved_at  TEXT DEFAULT NULL,
  error_msg    TEXT DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS action_audit (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  proposal_id INTEGER DEFAULT NULL REFERENCES action_proposals(id) ON DELETE SET NULL,
  action_type TEXT NOT NULL,
  item_type   TEXT NOT NULL,
  vault_id    TEXT NOT NULL,
  item_id     INTEGER DEFAULT NULL,
  outcome     TEXT NOT NULL
              CHECK(outcome IN ('proposed','approved','denied','executed','failed','dry_run')),
  payload_json TEXT NOT NULL,
  actor       TEXT NOT NULL DEFAULT 'user',
  timestamp   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS chat_tool_calls (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id  INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
  message_id  INTEGER NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
  tool_name   TEXT NOT NULL,
  input_json  TEXT NOT NULL,
  result_json TEXT NOT NULL,
  cache_hit   INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_tool_calls_session ON chat_tool_calls(session_id);

CREATE TABLE IF NOT EXISTS templates (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  slug          TEXT NOT NULL UNIQUE,
  name          TEXT NOT NULL,
  description   TEXT NOT NULL DEFAULT '',
  type          TEXT NOT NULL DEFAULT 'generation',
  prompt        TEXT NOT NULL,
  parameters    TEXT NOT NULL DEFAULT '[]',
  output_format TEXT NOT NULL DEFAULT 'markdown',
  created_at    TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
`

// ChatStore manages the chat.db SQLite database.
type ChatStore struct {
	db *sql.DB
}

// Open opens (or creates) chat.db at the given config directory and initialises the schema.
func Open(ctx context.Context, configDir string) (*ChatStore, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("chat: create config dir: %w", err)
	}
	dbPath := filepath.Join(configDir, "chat.db?_fk=1")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("chat: open db: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("chat: set WAL: %w", err)
	}
	if _, err := db.ExecContext(ctx, chatSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("chat: init schema: %w", err)
	}
	cs := &ChatStore{db: db}
	cs.seedBuiltinTemplates(ctx)
	return cs, nil
}

// Close closes the underlying database connection.
func (s *ChatStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// --- Session ---

// CreateSession opens a new chat session for the given vault.
func (s *ChatStore) CreateSession(ctx context.Context, vaultID string, dryRun bool) (*ChatSession, error) {
	dryRunInt := 0
	if dryRun {
		dryRunInt = 1
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_sessions (vault_id, dry_run) VALUES (?, ?)`,
		vaultID, dryRunInt,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetSession(ctx, id)
}

// EndSession sets ended_at on the session.
func (s *ChatStore) EndSession(ctx context.Context, sessionID int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE chat_sessions SET ended_at = datetime('now') WHERE id = ?`,
		sessionID,
	)
	return err
}

// GetSession returns a single session by ID.
func (s *ChatStore) GetSession(ctx context.Context, sessionID int64) (*ChatSession, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, vault_id, started_at, COALESCE(ended_at,''), dry_run
		 FROM chat_sessions WHERE id = ?`,
		sessionID,
	)
	var sess ChatSession
	var dryRunInt int
	if err := row.Scan(&sess.ID, &sess.VaultID, &sess.StartedAt, &sess.EndedAt, &dryRunInt); err != nil {
		return nil, err
	}
	sess.DryRun = dryRunInt == 1
	return &sess, nil
}

// --- Messages ---

// AddMessage persists a chat message and returns it with its assigned ID.
func (s *ChatStore) AddMessage(ctx context.Context, msg *ChatMessage) (*ChatMessage, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_messages (session_id, role, content) VALUES (?, ?, ?)`,
		msg.SessionID, msg.Role, msg.Content,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.getMessage(ctx, id)
}

func (s *ChatStore) getMessage(ctx context.Context, id int64) (*ChatMessage, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, role, content, created_at FROM chat_messages WHERE id = ?`, id)
	var m ChatMessage
	if err := row.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMessages returns all messages for a session, ordered oldest-first.
func (s *ChatStore) GetMessages(ctx context.Context, sessionID int64) ([]ChatMessage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, role, content, created_at
		 FROM chat_messages WHERE session_id = ? ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// --- Proposals ---

// CreateProposal persists a new ActionProposal (status=pending) and returns its ID.
func (s *ChatStore) CreateProposal(ctx context.Context, p *ActionProposal) (int64, error) {
	payloadJSON, err := json.Marshal(p.Payload)
	if err != nil {
		return 0, err
	}
	var diffJSON []byte
	if p.Diff != nil {
		diffJSON, err = json.Marshal(p.Diff)
		if err != nil {
			return 0, err
		}
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO action_proposals
		 (session_id, message_id, action_type, item_type, vault_id, payload_json, diff_json, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'pending')`,
		p.SessionID, p.MessageID, p.ActionType, p.ItemType, p.VaultID,
		string(payloadJSON), nullableStr(diffJSON),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateProposalStatus sets the status (and optional error message) on a proposal.
func (s *ChatStore) UpdateProposalStatus(ctx context.Context, proposalID int64, status, errorMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE action_proposals
		 SET status = ?, error_msg = ?, resolved_at = datetime('now')
		 WHERE id = ?`,
		status, nullStr(errorMsg), proposalID,
	)
	return err
}

// GetProposal returns a single ActionProposal by ID.
func (s *ChatStore) GetProposal(ctx context.Context, proposalID int64) (*ActionProposal, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, session_id, message_id, action_type, item_type, vault_id,
		        payload_json, COALESCE(diff_json,''), status,
		        created_at, COALESCE(resolved_at,''), COALESCE(error_msg,'')
		 FROM action_proposals WHERE id = ?`,
		proposalID,
	)
	var p ActionProposal
	var payloadJSON, diffJSON string
	var messageID sql.NullInt64
	if err := row.Scan(
		&p.ID, &p.SessionID, &messageID, &p.ActionType, &p.ItemType, &p.VaultID,
		&payloadJSON, &diffJSON, &p.Status,
		&p.CreatedAt, &p.ResolvedAt, &p.ErrorMsg,
	); err != nil {
		return nil, err
	}
	if messageID.Valid {
		p.MessageID = &messageID.Int64
	}
	if err := json.Unmarshal([]byte(payloadJSON), &p.Payload); err != nil {
		return nil, fmt.Errorf("chat: unmarshal proposal payload: %w", err)
	}
	if diffJSON != "" {
		if err := json.Unmarshal([]byte(diffJSON), &p.Diff); err != nil {
			return nil, fmt.Errorf("chat: unmarshal proposal diff: %w", err)
		}
	}
	return &p, nil
}

// --- Audit ---

// AppendAudit inserts an immutable audit record.
func (s *ChatStore) AppendAudit(ctx context.Context, entry *AuditEntry) error {
	var proposalID interface{}
	if entry.ProposalID != nil {
		proposalID = *entry.ProposalID
	}
	var itemID interface{}
	if entry.ItemID != nil {
		itemID = *entry.ItemID
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO action_audit
		 (proposal_id, action_type, item_type, vault_id, item_id, outcome, payload_json, actor)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		proposalID, entry.ActionType, entry.ItemType, entry.VaultID,
		itemID, entry.Outcome, entry.Payload, entry.Actor,
	)
	return err
}

// GetAudit returns the most recent audit entries (up to limit), newest-first.
func (s *ChatStore) GetAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, proposal_id, action_type, item_type, vault_id, item_id,
		        outcome, payload_json, actor, timestamp
		 FROM action_audit ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var proposalID, itemID sql.NullInt64
		if err := rows.Scan(
			&e.ID, &proposalID, &e.ActionType, &e.ItemType, &e.VaultID, &itemID,
			&e.Outcome, &e.Payload, &e.Actor, &e.Timestamp,
		); err != nil {
			return nil, err
		}
		if proposalID.Valid {
			e.ProposalID = &proposalID.Int64
		}
		if itemID.Valid {
			e.ItemID = &itemID.Int64
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// --- Tool calls ---

// PersistToolCall records one tool invocation tied to a session and assistant message.
func (s *ChatStore) PersistToolCall(ctx context.Context, sessionID, messageID int64, tc ToolCallRecord) error {
	cacheHit := 0
	if tc.CacheHit {
		cacheHit = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_tool_calls (session_id, message_id, tool_name, input_json, result_json, cache_hit)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		sessionID, messageID, tc.ToolName, tc.InputJSON, tc.ResultJSON, cacheHit,
	)
	return err
}

// GetRecentToolCalls returns up to limit tool calls for a session, newest-first.
func (s *ChatStore) GetRecentToolCalls(ctx context.Context, sessionID int64, limit int) ([]ToolCallRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT tool_name, input_json, result_json, cache_hit
		 FROM chat_tool_calls WHERE session_id = ? ORDER BY id DESC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var calls []ToolCallRecord
	for rows.Next() {
		var tc ToolCallRecord
		var cacheHit int
		if err := rows.Scan(&tc.ToolName, &tc.InputJSON, &tc.ResultJSON, &cacheHit); err != nil {
			return nil, err
		}
		tc.CacheHit = cacheHit == 1
		calls = append(calls, tc)
	}
	return calls, rows.Err()
}

// --- Templates ---

// CreateTemplate inserts a new template and returns it with the DB-assigned ID and timestamps.
func (s *ChatStore) CreateTemplate(ctx context.Context, t *Template) (*Template, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO templates (slug, name, description, type, prompt, parameters, output_format)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.Slug, t.Name, t.Description, t.Type, t.Prompt, t.Parameters, t.OutputFormat,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.getTemplateByID(ctx, id)
}

// GetTemplate returns a template by slug.
func (s *ChatStore) GetTemplate(ctx context.Context, slug string) (*Template, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, description, type, prompt, parameters, output_format, created_at, updated_at
		 FROM templates WHERE slug = ?`, slug)
	return scanTemplate(row)
}

// ListTemplates returns all templates ordered by name.
func (s *ChatStore) ListTemplates(ctx context.Context) ([]Template, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, slug, name, description, type, prompt, parameters, output_format, created_at, updated_at
		 FROM templates ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		t, err := scanTemplateRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// UpdateTemplate updates an existing template by slug.
func (s *ChatStore) UpdateTemplate(ctx context.Context, t *Template) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE templates SET name=?, description=?, type=?, prompt=?, parameters=?, output_format=?,
		                      updated_at=datetime('now')
		 WHERE slug=?`,
		t.Name, t.Description, t.Type, t.Prompt, t.Parameters, t.OutputFormat, t.Slug,
	)
	return err
}

// DeleteTemplate removes a template by slug.
func (s *ChatStore) DeleteTemplate(ctx context.Context, slug string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM templates WHERE slug = ?`, slug)
	return err
}

func (s *ChatStore) getTemplateByID(ctx context.Context, id int64) (*Template, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, description, type, prompt, parameters, output_format, created_at, updated_at
		 FROM templates WHERE id = ?`, id)
	return scanTemplate(row)
}

type templateScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(row templateScanner) (*Template, error) {
	var t Template
	if err := row.Scan(&t.ID, &t.Slug, &t.Name, &t.Description, &t.Type,
		&t.Prompt, &t.Parameters, &t.OutputFormat, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func scanTemplateRow(rows *sql.Rows) (*Template, error) {
	var t Template
	if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Description, &t.Type,
		&t.Prompt, &t.Parameters, &t.OutputFormat, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

// seedBuiltinTemplates inserts the built-in templates on first run (idempotent via INSERT OR IGNORE).
func (s *ChatStore) seedBuiltinTemplates(ctx context.Context) {
	builtins := []Template{
		{
			Slug:         "weekly-review",
			Name:         "Weekly Review",
			Description:  "Generate a structured weekly review: completions, overdue items, and next-week focus.",
			Type:         "analysis",
			OutputFormat: "markdown",
			Parameters:   `["vault_name","week_start","week_end"]`,
			Prompt: `Generate a weekly review for {{vault_name}} for the week of {{week_start}} to {{week_end}}.

Steps:
1. Call get_vault_stats for current counts.
2. Call search_vault with a query to find recently completed items.
3. Call search_vault to identify overdue items.
4. Write the review with these sections:
   - What was completed this week
   - What was not completed / carried over
   - Overdue items that need attention
   - Suggested focus for next week
   - Any patterns or opportunities worth noting`,
		},
		{
			Slug:         "vault-audit",
			Name:         "Vault Audit",
			Description:  "Audit vault health: stale items, taxonomy hygiene, Now section review, and recommended actions.",
			Type:         "analysis",
			OutputFormat: "markdown",
			Parameters:   `["vault_name"]`,
			Prompt: `Perform a vault audit for {{vault_name}}.

Steps:
1. Call get_vault_stats for an overview of counts and status.
2. Call list_taxonomy to review all projects, contexts, and tags.
3. Call search_vault to identify overdue items.
4. Call search_vault with section:now to review the Now section.
5. Write the audit report with these sections:
   - Vault health summary (counts, completion rate)
   - Now section review (is it focused? over-loaded?)
   - Stale and overdue items
   - Taxonomy hygiene (unused tags, overlapping contexts)
   - Recommended actions (prioritised)`,
		},
	}

	for _, tmpl := range builtins {
		_, _ = s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO templates (slug, name, description, type, prompt, parameters, output_format)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tmpl.Slug, tmpl.Name, tmpl.Description, tmpl.Type, tmpl.Prompt, tmpl.Parameters, tmpl.OutputFormat,
		)
	}
}

// --- helpers ---

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableStr(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}
