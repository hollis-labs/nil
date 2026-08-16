package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	feotel "github.com/hollis-labs/go-otel"
	"github.com/hollis-labs/nil/ingest"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// dbtx is satisfied by both *sql.DB and *sql.Tx. Internal item-mutation
// helpers (createItemTx, updateItemTx, setLinksTx, etc.) take a dbtx instead
// of assuming *sql.DB directly, so the same logic can run either against the
// store's pooled connection (the normal single-item path) or inside an
// explicit transaction (Store.CreateItemsBatch's all-or-nothing batch
// create) without duplicating the create/update logic for each case.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Store struct {
	DB *sql.DB
}

func (s *Store) Close() error {
	if s.DB != nil {
		return s.DB.Close()
	}
	return nil
}

func Open(ctx context.Context, dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	// busy_timeout is a per-connection setting (unlike journal_mode, it is
	// NOT persisted in the database file), and Go's database/sql pool opens
	// additional physical connections on demand as concurrent callers show
	// up — a one-time `PRAGMA busy_timeout` exec after Open only reaches
	// whichever single connection runs it, leaving every other pooled
	// connection at the default (0, i.e. fail fast on contention instead of
	// waiting). Passing it via the DSN's _pragma param instead makes the
	// modernc.org/sqlite driver apply it to every connection it opens, which
	// is what actually matters once the GUI, a standalone `nil serve-api`
	// process, and CLI invocations can all have live connections to the same
	// vault at once. See modernc.org/sqlite's Driver.Open doc comment for
	// the supported _pragma DSN syntax.
	//
	// Foreign key enforcement (needed for refs' ON DELETE CASCADE, see
	// schema.sql) is also per-connection in SQLite and must go through
	// _pragma for the same reason — but the DSN param actually recognized by
	// modernc.org/sqlite's applyQueryParams is "foreign_keys(1)", NOT "_fk=1"
	// (that was never a real driver param; it was silently ignored, meaning
	// foreign key enforcement was never actually turned on). _pragma may be
	// repeated — applyQueryParams collects every "_pragma" value from the
	// query string and execs each as its own "PRAGMA ..." statement (sorted
	// so busy_timeout always runs first), so both pragmas below are applied
	// on every connection the pool opens, not just the last one.
	dbPath := filepath.Join(dataDir, "todo.db?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better cloud sync compatibility and true
	// multi-process concurrency: the GUI, a standalone `nil serve-api`
	// process, and CLI invocations can all open the same vault at once.
	// Unlike busy_timeout, journal_mode=WAL is persisted in the database
	// file itself, so setting it once here (on whichever connection runs
	// it first) is sufficient — it doesn't need to be a per-connection DSN
	// param.
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return nil, err
	}

	// Run migrations
	if err := runMigrations(ctx, db); err != nil {
		return nil, err
	}

	// Ensure the external_ref partial unique index exists. This must run
	// AFTER migrations (not as part of the unconditional schemaSQL exec
	// above): schemaSQL runs on every Open() call regardless of the
	// database's existing schema_version, and for a pre-v12 database (no
	// external_ref column yet) an index on that column would fail outright
	// before migration v12 ever got a chance to add it via ALTER TABLE. By
	// this point the column is guaranteed to exist on every path — a fresh
	// install created it via schemaSQL's CREATE TABLE, an upgrading install
	// created it via migrateV12's ALTER TABLE — so this is safe. Idempotent
	// (IF NOT EXISTS), so running it on every Open() call is harmless.
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS todos_external_ref_idx
		ON todos(external_ref)
		WHERE external_ref IS NOT NULL AND external_ref != ''`); err != nil {
		return nil, fmt.Errorf("ensure external_ref index: %w", err)
	}

	return &Store{DB: db}, nil
}

// CreateItem inserts an item and writes its FTS5 entry. The caller supplies
// notes_doc (PM JSON, source of truth) and optionally notes_html (write-time
// render cache). FTS5 indexable text is derived from notes_doc.
//
// Idempotent-write dedup: if t.ExternalRef is non-empty and a row with the
// same external_ref already exists in this vault, CreateItem overwrites that
// row's create-payload fields (see overwriteByExternalRefTx) instead of
// inserting a duplicate. See CW-20260816-0045.
func (s *Store) CreateItem(ctx context.Context, t *Item) (*Item, error) {
	ctx, span := feotel.StartSpan(ctx, "nil.item.create")
	defer span.End()
	return createItemTx(ctx, s.DB, t)
}

// CreateItemsBatch creates every item in items inside a single transaction:
// all-or-nothing. If any item fails (invalid kind, a DB constraint, etc.)
// the entire batch is rolled back and no item is created — nothing is
// half-applied for the caller to reconcile. This is a deliberate design
// choice over best-effort/partial-success: for a local, single-vault SQLite
// store, a caller that gets an error back can simply fix the offending item
// (the error names its index) and retry the whole batch, rather than having
// to diff a per-item results list against what it originally sent to figure
// out what still needs pushing. See CW-20260816-0045 for the full tradeoff.
//
// Each item goes through the same create-or-match-by-external_ref logic as
// a single CreateItem call, so a batch that re-pushes previously-seen
// external_ref values updates those rows in place rather than duplicating
// them — the batch and single-item paths share identical semantics.
//
// Returns the created/updated items in the same order as the input. An
// empty input returns an empty (non-nil) slice and no error, without
// opening a transaction.
func (s *Store) CreateItemsBatch(ctx context.Context, items []Item) ([]Item, error) {
	ctx, span := feotel.StartSpan(ctx, "nil.item.create_batch")
	defer span.End()

	if len(items) == 0 {
		return []Item{}, nil
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin batch create tx: %w", err)
	}
	commit := false
	defer func() {
		if !commit {
			_ = tx.Rollback()
		}
	}()

	out := make([]Item, 0, len(items))
	for i := range items {
		item := items[i] // local copy: createItemTx mutates fields (ID, Kind, Inbox, ...)
		created, err := createItemTx(ctx, tx, &item)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		out = append(out, *created)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit batch create: %w", err)
	}
	commit = true
	return out, nil
}

// createItemTx is the shared implementation behind Store.CreateItem (dbx =
// s.DB) and Store.CreateItemsBatch (dbx = the batch's *sql.Tx), so both
// paths get identical validation, external_ref dedup, taxonomy linking, FTS
// indexing, and ref-syncing behavior.
func createItemTx(ctx context.Context, dbx dbtx, t *Item) (*Item, error) {
	if t.Section == "" {
		t.Section = "anytime"
	}
	kind, err := validateKindOrDefaultTx(ctx, dbx, t.Kind)
	if err != nil {
		return nil, err
	}
	t.Kind = kind
	if strings.TrimSpace(t.Title) == "" {
		t.Inbox = true
	}

	if strings.TrimSpace(t.ExternalRef) != "" {
		var existingID int64
		err = dbx.QueryRowContext(ctx, "SELECT id FROM todos WHERE external_ref = ?", t.ExternalRef).Scan(&existingID)
		switch {
		case err == nil:
			return overwriteByExternalRefTx(ctx, dbx, existingID, t)
		case errors.Is(err, sql.ErrNoRows):
			// No existing row for this external_ref — fall through to a
			// normal insert below.
		default:
			return nil, fmt.Errorf("checking external_ref: %w", err)
		}
	}

	res, err := dbx.ExecContext(ctx, `
INSERT INTO todos(title, priority, completed, archived, due_at, threshold_at, recurrence_rule, source_line, notes_doc, notes_html, notes_html_version, section, pinned, kind, inbox, api_source, external_ref)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.Title, t.Priority, t.Completed, t.Archived, t.DueAt, t.Threshold, t.Recur, t.Source,
		t.NotesDoc, t.NotesHTML, t.NotesHTMLVersion,
		t.Section, t.Pinned, t.Kind, t.Inbox, t.APISource, nullIfEmpty(t.ExternalRef),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	t.ID = id

	// set links
	if err := setLinksTx(ctx, dbx, "todo_projects", "project_id", id, t.Projects); err != nil {
		return nil, err
	}
	if err := setLinksTx(ctx, dbx, "todo_contexts", "context_id", id, t.Contexts); err != nil {
		return nil, err
	}
	if err := setLinksTx(ctx, dbx, "todo_tags", "tag_id", id, t.Tags); err != nil {
		return nil, err
	}

	if err := updateFTSTx(ctx, dbx, id, t.Title, derivePlainText(t.NotesDoc)); err != nil {
		return nil, err
	}
	// Sync refs for any wikilinks present in notes_doc.
	if refIDs := ingest.ExtractRefIDs(t.NotesDoc); len(refIDs) > 0 {
		if err := updateRefsTx(ctx, dbx, id, refIDs); err != nil {
			return nil, err
		}
	}

	_ = hydrateTx(ctx, dbx, t)
	return t, nil
}

// overwriteByExternalRefTx implements the "re-push updates the existing
// row" half of external_ref idempotency. It performs a full overwrite of
// every field a create payload can carry (title, priority, due/threshold/
// recurrence, notes, section, pinned, kind, inbox, taxonomy) — matching the
// "re-pushing the same logical item" acceptance criteria — but deliberately
// does NOT touch completed, archived, api_source, external_ref, or
// created_at:
//
//   - completed/archived have no representation in items.CreateInput (the
//     type every create surface — HTTP, CLI, MCP — funnels through), so
//     there is no "create payload value" for them to overwrite with. If this
//     path blasted them to the create-call's zero value, every re-push would
//     silently undo a user's local completion/archival of the item. They are
//     left exactly as they were on the existing row.
//   - api_source records who originally created the row; external_ref is the
//     matching key itself (already correct — it's how we found this row).
//   - created_at is preserved by simply never being part of this UPDATE;
//     updated_at still advances via the todos_update_ts trigger, same as any
//     other update.
func overwriteByExternalRefTx(ctx context.Context, dbx dbtx, id int64, t *Item) (*Item, error) {
	_, err := dbx.ExecContext(ctx, `
UPDATE todos SET title=?, priority=?, due_at=?, threshold_at=?, recurrence_rule=?, notes_doc=?, notes_html=?, notes_html_version=?, section=?, pinned=?, kind=?, inbox=? WHERE id=?`,
		t.Title, t.Priority, t.DueAt, t.Threshold, t.Recur,
		t.NotesDoc, t.NotesHTML, t.NotesHTMLVersion,
		t.Section, t.Pinned, t.Kind, t.Inbox, id,
	)
	if err != nil {
		return nil, fmt.Errorf("overwrite by external_ref: %w", err)
	}
	t.ID = id

	if err := setLinksTx(ctx, dbx, "todo_projects", "project_id", id, t.Projects); err != nil {
		return nil, err
	}
	if err := setLinksTx(ctx, dbx, "todo_contexts", "context_id", id, t.Contexts); err != nil {
		return nil, err
	}
	if err := setLinksTx(ctx, dbx, "todo_tags", "tag_id", id, t.Tags); err != nil {
		return nil, err
	}
	if err := updateFTSTx(ctx, dbx, id, t.Title, derivePlainText(t.NotesDoc)); err != nil {
		return nil, err
	}
	refIDs := ingest.ExtractRefIDs(t.NotesDoc)
	if err := updateRefsTx(ctx, dbx, id, refIDs); err != nil {
		return nil, err
	}

	// Re-fetch rather than trust the caller's t: completed/archived,
	// created_at, api_source, and external_ref were deliberately not
	// touched above, so the caller's in-memory copy (built fresh from a
	// create payload) doesn't reflect their real, preserved values.
	return getItemTx(ctx, dbx, id)
}

// nullIfEmpty maps the Go zero value ("") to a real SQL NULL for external_ref
// inserts. Without this, every plain create (no external_ref supplied) would
// store ” instead of NULL — harmless for the partial unique index (which
// already excludes ” as well as NULL), but NULL is the more honest
// "not set" representation for a column whose DDL default is NULL, and it's
// what every pre-migration row already has.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Store) UpdateItem(ctx context.Context, t *Item) error {
	ctx, span := feotel.StartSpan(ctx, "nil.item.update")
	defer span.End()
	return updateItemTx(ctx, s.DB, t)
}

func updateItemTx(ctx context.Context, dbx dbtx, t *Item) error {
	if t.Section == "" {
		t.Section = "anytime"
	}
	kind, err := validateKindOrDefaultTx(ctx, dbx, t.Kind)
	if err != nil {
		return err
	}
	t.Kind = kind
	_, err = dbx.ExecContext(ctx, `
UPDATE todos SET title=?, priority=?, completed=?, archived=?, due_at=?, threshold_at=?, recurrence_rule=?, notes_doc=?, notes_html=?, notes_html_version=?, section=?, pinned=?, kind=?, inbox=?, external_ref=? WHERE id=?`,
		t.Title, t.Priority, t.Completed, t.Archived, t.DueAt, t.Threshold, t.Recur,
		t.NotesDoc, t.NotesHTML, t.NotesHTMLVersion,
		t.Section, t.Pinned, t.Kind, t.Inbox, nullIfEmpty(t.ExternalRef), t.ID,
	)
	if err != nil {
		return err
	}
	if err := setLinksTx(ctx, dbx, "todo_projects", "project_id", t.ID, t.Projects); err != nil {
		return err
	}
	if err := setLinksTx(ctx, dbx, "todo_contexts", "context_id", t.ID, t.Contexts); err != nil {
		return err
	}
	if err := setLinksTx(ctx, dbx, "todo_tags", "tag_id", t.ID, t.Tags); err != nil {
		return err
	}
	if err := updateFTSTx(ctx, dbx, t.ID, t.Title, derivePlainText(t.NotesDoc)); err != nil {
		return err
	}
	refIDs := ingest.ExtractRefIDs(t.NotesDoc)
	if err := updateRefsTx(ctx, dbx, t.ID, refIDs); err != nil {
		return err
	}
	return nil
}

// GetItem returns a single item by ID.
func (s *Store) GetItem(ctx context.Context, id int64) (*Item, error) {
	ctx, span := feotel.StartSpan(ctx, "nil.item.get")
	defer span.End()
	return getItemTx(ctx, s.DB, id)
}

func getItemTx(ctx context.Context, dbx dbtx, id int64) (*Item, error) {
	var t Item
	var pri *string
	var source sql.NullString
	var notesDoc, notesHTML sql.NullString
	var notesHTMLVersion sql.NullInt64
	var apiSource, externalRef sql.NullString
	err := dbx.QueryRowContext(ctx, `
SELECT id, title, priority, completed, archived, created_at, updated_at, due_at, threshold_at, recurrence_rule, source_line, notes_doc, notes_html, notes_html_version, section, pinned, kind, inbox, api_source, external_ref
FROM todos WHERE id=?`, id).Scan(
		&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt,
		&t.DueAt, &t.Threshold, &t.Recur, &source,
		&notesDoc, &notesHTML, &notesHTMLVersion,
		&t.Section, &t.Pinned, &t.Kind, &t.Inbox, &apiSource, &externalRef,
	)
	if err != nil {
		return nil, err
	}
	t.Priority = pri
	t.Source = source.String
	t.NotesDoc = notesDoc.String
	t.NotesHTML = notesHTML.String
	t.NotesHTMLVersion = int(notesHTMLVersion.Int64)
	t.APISource = apiSource.String
	t.ExternalRef = externalRef.String
	if t.Section == "" {
		t.Section = "anytime"
	}
	if err := hydrateTx(ctx, dbx, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateRefs replaces all outgoing refs from sourceID with targetIDs.
func (s *Store) UpdateRefs(ctx context.Context, sourceID int64, targetIDs []int64) error {
	return updateRefsTx(ctx, s.DB, sourceID, targetIDs)
}

func updateRefsTx(ctx context.Context, dbx dbtx, sourceID int64, targetIDs []int64) error {
	if _, err := dbx.ExecContext(ctx, "DELETE FROM refs WHERE source_id = ?", sourceID); err != nil {
		return err
	}
	for _, targetID := range targetIDs {
		if targetID == sourceID {
			continue // skip self-references
		}
		if _, err := dbx.ExecContext(ctx, "INSERT OR IGNORE INTO refs (source_id, target_id) VALUES (?, ?)", sourceID, targetID); err != nil {
			return err
		}
	}
	return nil
}

// GetBackrefs returns all items that reference targetID.
func (s *Store) GetBackrefs(ctx context.Context, targetID int64) ([]Item, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT t.id, t.title, t.priority, t.completed, t.archived, t.created_at, t.updated_at,
       t.due_at, t.threshold_at, t.recurrence_rule, t.source_line, t.notes_doc, t.notes_html, t.notes_html_version, t.section, t.pinned, t.kind, t.inbox, t.api_source, t.external_ref
FROM todos t JOIN refs r ON r.source_id = t.id
WHERE r.target_id = ?
ORDER BY t.updated_at DESC`, targetID)
	if err != nil {
		return []Item{}, err
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var t Item
		var pri *string
		var source sql.NullString
		var notesDoc, notesHTML sql.NullString
		var notesHTMLVersion sql.NullInt64
		var apiSource, externalRef sql.NullString
		err := rows.Scan(&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt,
			&t.DueAt, &t.Threshold, &t.Recur, &source,
			&notesDoc, &notesHTML, &notesHTMLVersion,
			&t.Section, &t.Pinned, &t.Kind, &t.Inbox, &apiSource, &externalRef)
		if err != nil {
			return []Item{}, err
		}
		t.Priority = pri
		t.Source = source.String
		t.NotesDoc = notesDoc.String
		t.NotesHTML = notesHTML.String
		t.NotesHTMLVersion = int(notesHTMLVersion.Int64)
		t.APISource = apiSource.String
		t.ExternalRef = externalRef.String
		if t.Section == "" {
			t.Section = "anytime"
		}
		if err := s.hydrate(ctx, &t); err != nil {
			return []Item{}, err
		}
		out = append(out, t)
	}
	if out == nil {
		return []Item{}, nil
	}
	return out, nil
}

func (s *Store) ToggleComplete(ctx context.Context, id int64, completed bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE todos SET completed=? WHERE id=?`, completed, id)
	return err
}
func (s *Store) Archive(ctx context.Context, id int64, archived bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE todos SET archived=? WHERE id=?`, archived, id)
	return err
}

func (s *Store) DeleteItem(ctx context.Context, id int64) error {
	ctx, span := feotel.StartSpan(ctx, "nil.item.delete")
	defer span.End()

	// Wrap the row delete + FTS delete in a transaction so the two stay in
	// sync — if either fails the whole operation rolls back and search won't
	// reference a now-missing row (or vice versa).
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM todos WHERE id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM todos_fts WHERE rowid = ?", id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// GetStats returns aggregate counts for the vault.
func (s *Store) GetStats(ctx context.Context) (*VaultStats, error) {
	var st VaultStats
	err := s.DB.QueryRowContext(ctx, `
SELECT
    COUNT(*) AS total,
    SUM(CASE WHEN completed=0 AND archived=0 AND inbox=0 THEN 1 ELSE 0 END) AS open,
    SUM(CASE WHEN completed=1 THEN 1 ELSE 0 END) AS completed,
    SUM(CASE WHEN archived=1 THEN 1 ELSE 0 END) AS archived,
    SUM(CASE WHEN completed=0 AND archived=0 AND due_at IS NOT NULL AND due_at < date('now') THEN 1 ELSE 0 END) AS overdue,
    SUM(CASE WHEN inbox=1 AND archived=0 THEN 1 ELSE 0 END) AS inbox,
    SUM(CASE WHEN kind='note' THEN 1 ELSE 0 END) AS notes,
    SUM(CASE WHEN kind='todo' THEN 1 ELSE 0 END) AS todos,
    SUM(CASE WHEN kind='scratch' THEN 1 ELSE 0 END) AS scratch,
    SUM(CASE WHEN section='now'     AND inbox=0 AND archived=0 THEN 1 ELSE 0 END) AS by_now,
    SUM(CASE WHEN section='soon'    AND inbox=0 AND archived=0 THEN 1 ELSE 0 END) AS by_soon,
    SUM(CASE WHEN section='anytime' AND inbox=0 AND archived=0 THEN 1 ELSE 0 END) AS by_anytime
FROM todos
`).Scan(&st.Total, &st.Open, &st.Completed, &st.Archived,
		&st.Overdue, &st.Inbox, &st.Notes, &st.Todos, &st.Scratch,
		&st.BySection.Now, &st.BySection.Soon, &st.BySection.Anytime)
	if err != nil {
		return nil, err
	}
	return &st, nil
}
