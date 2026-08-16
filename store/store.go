package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	feotel "github.com/hollis-labs/go-otel"
	"github.com/hollis-labs/nil/ingest"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// reHTMLBR matches <br> / <br/> variants.
var reHTMLBR = regexp.MustCompile(`(?i)<br\s*/?>`)

// reHTMLBlock matches closing block-level tags whose content ends a "paragraph".
var reHTMLBlock = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|blockquote)>`)

// reHTMLTag strips any remaining HTML tag.
var reHTMLTag = regexp.MustCompile(`<[^>]+>`)

// reMultiNL collapses 3+ consecutive newlines to 2.
var reMultiNL = regexp.MustCompile(`\n{3,}`)

// stripHTML converts TipTap HTML to plain text for FTS5 indexing and AI context.
// It preserves paragraph breaks and handles common HTML entities. No CGo required.
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	s = reHTMLBR.ReplaceAllString(s, "\n")
	s = reHTMLBlock.ReplaceAllString(s, "\n\n")
	s = reHTMLTag.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = reMultiNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

const currentSchemaVersion = 11

type migration struct {
	version int
	sql     string
}

// Migrations v7-v10 use empty sql and are dispatched as inline blocks in
// runMigrations (multi-statement, idempotency-aware).
var migrations = []migration{
	{
		version: 1,
		sql:     "ALTER TABLE todos ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0",
	},
	{
		version: 2,
		sql:     "ALTER TABLE todos ADD COLUMN type TEXT NOT NULL DEFAULT 'todo'",
	},
	{
		version: 3,
		sql: `CREATE TABLE IF NOT EXISTS refs (
			source_id INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
			target_id INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
			PRIMARY KEY (source_id, target_id)
		)`,
	},
	{
		version: 4,
		sql:     "ALTER TABLE todos ADD COLUMN inbox INTEGER NOT NULL DEFAULT 0",
	},
	{
		version: 5,
		sql:     "ALTER TABLE todos ADD COLUMN api_source TEXT DEFAULT NULL",
	},
	{
		version: 6,
		sql:     "ALTER TABLE todos ADD COLUMN notes_text TEXT DEFAULT ''",
	},
	{version: 7, sql: ""},
	{version: 8, sql: ""},
	{version: 9, sql: ""},
	{version: 10, sql: ""},
	{version: 11, sql: ""},
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
	dbPath := filepath.Join(dataDir, "todo.db?_fk=1&_pragma=busy_timeout(5000)")
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

	return &Store{DB: db}, nil
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	// Create schema_version table if it doesn't exist
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return err
	}

	// Get current version
	var currentVersion int
	err = db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return err
	}

	// Fresh-install fast path: if schema_version is empty AND the latest
	// columns/tables are all already present (schema.sql produced the final
	// shape on the first run), skip every historical migration and just
	// stamp the current version. This avoids the wasteful churn of v6/v8
	// dropping and recreating todos_fts twice on a brand-new DB. Fresh
	// installs also can't have any pre-existing notes_md content, so the
	// v11 backfill is a no-op for them.
	if currentVersion == 0 {
		var hasNotesDoc, hasKind, hasKinds int
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='notes_doc'").Scan(&hasNotesDoc)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='kind'").Scan(&hasKind)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='kinds'").Scan(&hasKinds)
		if hasNotesDoc > 0 && hasKind > 0 && hasKinds > 0 {
			if _, err := db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", currentSchemaVersion); err != nil {
				return err
			}
			return nil
		}
	}

	// Run pending migrations
	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}

		// Check if column already exists (idempotency for ALTER TABLE migrations)
		if m.version == 1 {
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='pinned'").Scan(&count)
			if err == nil && count > 0 {
				_, err = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				if err != nil {
					return err
				}
				continue
			}
		}
		if m.version == 2 {
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='type'").Scan(&count)
			if err == nil && count > 0 {
				_, err = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				if err != nil {
					return err
				}
				continue
			}
		}
		if m.version == 3 {
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='refs'").Scan(&count)
			if err == nil && count > 0 {
				_, err = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				if err != nil {
					return err
				}
				continue
			}
		}
		if m.version == 4 {
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='inbox'").Scan(&count)
			if err == nil && count > 0 {
				_, err = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				if err != nil {
					return err
				}
				continue
			}
		}
		if m.version == 5 {
			var count int
			err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='api_source'").Scan(&count)
			if err == nil && count > 0 {
				_, _ = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				continue
			}
		}
		if m.version == 6 {
			// Idempotency: skip if notes_text already exists.
			var count int
			if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='notes_text'").Scan(&count); err == nil && count > 0 {
				_, _ = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				continue
			}
			// 1. Add the column.
			if _, err := db.ExecContext(ctx, m.sql); err != nil {
				return fmt.Errorf("migration v6 add column: %w", err)
			}
			// 2. Backfill notes_text from notes_md (Go-side stripHTML).
			bRows, err := db.QueryContext(ctx, "SELECT id, notes_md FROM todos WHERE notes_md != '' AND (notes_text = '' OR notes_text IS NULL)")
			if err != nil {
				return fmt.Errorf("migration v6 backfill query: %w", err)
			}
			type bf struct {
				id    int64
				notes string
			}
			var bItems []bf
			for bRows.Next() {
				var b bf
				if scanErr := bRows.Scan(&b.id, &b.notes); scanErr == nil {
					bItems = append(bItems, b)
				}
			}
			bRows.Close()
			for _, b := range bItems {
				if _, err := db.ExecContext(ctx, "UPDATE todos SET notes_text = ? WHERE id = ?", stripHTML(b.notes), b.id); err != nil {
					return fmt.Errorf("migration v6 backfill update id=%d: %w", b.id, err)
				}
			}
			// 3. Drop old FTS triggers (they reference notes_md).
			for _, stmt := range []string{
				"DROP TRIGGER IF EXISTS todos_ai",
				"DROP TRIGGER IF EXISTS todos_ad",
				"DROP TRIGGER IF EXISTS todos_au",
			} {
				if _, err := db.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("migration v6 drop trigger: %w", err)
				}
			}
			// 4. Drop and recreate FTS5 virtual table with notes_text.
			if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS todos_fts"); err != nil {
				return fmt.Errorf("migration v6 drop fts: %w", err)
			}
			if _, err := db.ExecContext(ctx, "CREATE VIRTUAL TABLE todos_fts USING fts5(title, notes_text, content='todos', content_rowid='id')"); err != nil {
				return fmt.Errorf("migration v6 create fts: %w", err)
			}
			// 5. Recreate triggers referencing notes_text.
			triggers := []string{
				`CREATE TRIGGER todos_ai AFTER INSERT ON todos BEGIN INSERT INTO todos_fts(rowid, title, notes_text) VALUES (new.id, new.title, new.notes_text); END`,
				`CREATE TRIGGER todos_ad AFTER DELETE ON todos BEGIN INSERT INTO todos_fts(todos_fts, rowid, title, notes_text) VALUES('delete', old.id, old.title, old.notes_text); END`,
				`CREATE TRIGGER todos_au AFTER UPDATE ON todos BEGIN INSERT INTO todos_fts(todos_fts, rowid, title, notes_text) VALUES('delete', old.id, old.title, old.notes_text); INSERT INTO todos_fts(rowid, title, notes_text) VALUES (new.id, new.title, new.notes_text); END`,
			}
			for _, stmt := range triggers {
				if _, err := db.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("migration v6 create trigger: %w", err)
				}
			}
			// 6. Rebuild FTS5 index from content table.
			if _, err := db.ExecContext(ctx, "INSERT INTO todos_fts(todos_fts) VALUES('rebuild')"); err != nil {
				return fmt.Errorf("migration v6 fts rebuild: %w", err)
			}
			// Record migration.
			if _, err := db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
				return err
			}
			continue
		}
		if m.version == 7 {
			if err := migrateV7(ctx, db); err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
				return err
			}
			continue
		}
		if m.version == 8 {
			if err := migrateV8(ctx, db); err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
				return err
			}
			continue
		}
		if m.version == 9 {
			if err := migrateV9(ctx, db); err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
				return err
			}
			continue
		}
		if m.version == 10 {
			if err := migrateV10(ctx, db); err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
				return err
			}
			continue
		}
		if m.version == 11 {
			if err := migrateV11(ctx, db); err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
				return err
			}
			continue
		}

		// Run migration
		_, err = db.ExecContext(ctx, m.sql)
		if err != nil {
			return err
		}

		// Record migration
		_, err = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
		if err != nil {
			return err
		}
	}

	return nil
}

// migrateV7 adds notes_doc, notes_html, notes_html_version columns to todos.
// Idempotent: skips if notes_doc already exists (fresh installs ran schema.sql first).
func migrateV7(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='notes_doc'").Scan(&count); err == nil && count > 0 {
		return nil
	}
	stmts := []string{
		"ALTER TABLE todos ADD COLUMN notes_doc TEXT DEFAULT ''",
		"ALTER TABLE todos ADD COLUMN notes_html TEXT DEFAULT ''",
		"ALTER TABLE todos ADD COLUMN notes_html_version INTEGER NOT NULL DEFAULT 0",
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration v7: %w", err)
		}
	}
	return nil
}

// migrateV8 switches FTS5 from external-content (todos.notes_text) to
// standalone (FTS5 stores its own copy). Drops the v6 triggers and the
// todos.notes_text column. The application layer (CreateItem / UpdateItem
// / DeleteItem) is responsible for keeping todos_fts in sync via
// updateFTS / deleteFTS.
// Idempotent: skips if notes_text column already gone.
func migrateV8(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='notes_text'").Scan(&count); err == nil && count == 0 {
		return nil
	}
	// 1. Drop existing FTS triggers (they reference notes_text).
	for _, stmt := range []string{
		"DROP TRIGGER IF EXISTS todos_ai",
		"DROP TRIGGER IF EXISTS todos_ad",
		"DROP TRIGGER IF EXISTS todos_au",
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration v8 drop trigger: %w", err)
		}
	}
	// 2. Drop the external-content FTS5 table.
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS todos_fts"); err != nil {
		return fmt.Errorf("migration v8 drop fts: %w", err)
	}
	// 3. Create new standalone FTS5 table.
	if _, err := db.ExecContext(ctx, "CREATE VIRTUAL TABLE todos_fts USING fts5(title, notes_text)"); err != nil {
		return fmt.Errorf("migration v8 create fts: %w", err)
	}
	// 4. Backfill FTS5 from current todos. notes_doc is empty at this stage
	//    (the frontend backfill hasn't run yet); use stripHTML(notes_md) as
	//    the plain-text source.
	rows, err := db.QueryContext(ctx, "SELECT id, title, notes_md FROM todos")
	if err != nil {
		return fmt.Errorf("migration v8 backfill query: %w", err)
	}
	type ftsRow struct {
		id      int64
		title   string
		notesMD string
	}
	var items []ftsRow
	scanErrs := 0
	for rows.Next() {
		var r ftsRow
		if err := rows.Scan(&r.id, &r.title, &r.notesMD); err != nil {
			scanErrs++
			fmt.Fprintf(os.Stderr, "migration v8: scan error on a todos row, skipping: %v\n", err)
			continue
		}
		items = append(items, r)
	}
	rows.Close()
	if scanErrs > 0 {
		fmt.Fprintf(os.Stderr, "migration v8: %d row(s) skipped due to scan errors; those rows will not be in the search index until they are next edited\n", scanErrs)
	}
	for _, r := range items {
		text := stripHTML(r.notesMD)
		if _, err := db.ExecContext(ctx, "INSERT INTO todos_fts(rowid, title, notes_text) VALUES (?,?,?)", r.id, r.title, text); err != nil {
			return fmt.Errorf("migration v8 backfill insert id=%d: %w", r.id, err)
		}
	}
	// 5. Drop notes_text column from todos. SQLite 3.35+ supports DROP COLUMN.
	if _, err := db.ExecContext(ctx, "ALTER TABLE todos DROP COLUMN notes_text"); err != nil {
		return fmt.Errorf("migration v8 drop notes_text: %w", err)
	}
	return nil
}

// migrateV9 renames the todos.type column to todos.kind.
// Idempotent: skips if kind already exists.
func migrateV9(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='kind'").Scan(&count); err == nil && count > 0 {
		return nil
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE todos RENAME COLUMN type TO kind"); err != nil {
		return fmt.Errorf("migration v9 rename: %w", err)
	}
	return nil
}

// migrateV11 backfills notes_doc + notes_html for any row that still has
// notes_md content but no notes_doc. Replaces the v1.3.0 frontend-driven
// backfill, which failed silently for at least one user (the localStorage
// "done" flag got set without rows actually converting). Running this on
// the Go side, inside the migration sequence, means the work happens
// deterministically as part of app startup and can't be skipped.
//
// Idempotent: only touches rows where notes_doc is empty/null. Safe to
// re-run after partial completion.
func migrateV11(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `
SELECT id, title, notes_md FROM todos
WHERE notes_md IS NOT NULL AND notes_md != ''
  AND (notes_doc IS NULL OR notes_doc = '')`)
	if err != nil {
		return fmt.Errorf("migration v11 query: %w", err)
	}
	type cand struct {
		id      int64
		title   string
		notesMD string
	}
	var candidates []cand
	scanErrs := 0
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.title, &c.notesMD); err != nil {
			scanErrs++
			fmt.Fprintf(os.Stderr, "migration v11: scan error on a todos row, skipping: %v\n", err)
			continue
		}
		candidates = append(candidates, c)
	}
	rows.Close()
	if scanErrs > 0 {
		fmt.Fprintf(os.Stderr, "migration v11: %d row(s) skipped due to scan errors; their notes_md content will not be converted\n", scanErrs)
	}
	if len(candidates) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migration v11 begin: %w", err)
	}
	commit := false
	defer func() {
		if !commit {
			_ = tx.Rollback()
		}
	}()
	convertErrs := 0
	ftsErrs := 0
	converted := 0
	for _, c := range candidates {
		// notes_md is TipTap HTML (v1.3.0 column-naming legacy). Convert to
		// PM JSON via the ingest package; keep the original HTML as the cache.
		doc, ierr := ingest.HTMLToDoc(c.notesMD)
		if ierr != nil {
			// Skip the row but record the failure. Failing the whole migration
			// over one bad row would block startup; the row stays in its
			// pre-backfill state and can be recovered via the nil-recover tool.
			convertErrs++
			fmt.Fprintf(os.Stderr, "migration v11: id=%d HTML→doc conversion failed (skipping): %v\n", c.id, ierr)
			continue
		}
		if _, uerr := tx.ExecContext(ctx, `
UPDATE todos SET notes_doc = ?, notes_html = ?, notes_html_version = 1
WHERE id = ?`, doc, c.notesMD, c.id); uerr != nil {
			return fmt.Errorf("migration v11 update id=%d: %w", c.id, uerr)
		}
		// Refresh FTS5 to reflect the new plain-text projection. Failures
		// here mean search will be stale for this row until the user next
		// edits it — log but continue so a broken FTS index can't block the
		// whole backfill.
		plain, _ := ingest.DocToPlainText(doc)
		if _, ferr := tx.ExecContext(ctx, "DELETE FROM todos_fts WHERE rowid = ?", c.id); ferr != nil {
			ftsErrs++
			fmt.Fprintf(os.Stderr, "migration v11: id=%d FTS delete failed: %v\n", c.id, ferr)
		}
		if _, ferr := tx.ExecContext(ctx, "INSERT INTO todos_fts(rowid, title, notes_text) VALUES (?,?,?)", c.id, c.title, plain); ferr != nil {
			ftsErrs++
			fmt.Fprintf(os.Stderr, "migration v11: id=%d FTS insert failed: %v\n", c.id, ferr)
		}
		converted++
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration v11 commit: %w", err)
	}
	commit = true
	if convertErrs > 0 || ftsErrs > 0 {
		fmt.Fprintf(os.Stderr, "migration v11 summary: %d converted, %d conversion errors, %d FTS errors\n",
			converted, convertErrs, ftsErrs)
	}
	return nil
}

// migrateV10 creates the kinds registry table and seeds the three core kinds.
// Idempotent: ensures the seed rows exist whether or not the table already did.
func migrateV10(ctx context.Context, db *sql.DB) error {
	createSQL := `CREATE TABLE IF NOT EXISTS kinds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL,
		icon TEXT NOT NULL DEFAULT '',
		default_view TEXT NOT NULL DEFAULT '',
		plugin_id TEXT DEFAULT NULL,
		is_core INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`
	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("migration v10 create kinds: %w", err)
	}
	seedSQL := `INSERT OR IGNORE INTO kinds (name, display_name, icon, is_core) VALUES
		('todo','Todo','check-square',1),
		('note','Note','file-text',1),
		('scratch','Scratch','edit',1)`
	if _, err := db.ExecContext(ctx, seedSQL); err != nil {
		return fmt.Errorf("migration v10 seed kinds: %w", err)
	}
	return nil
}

// -- Helpers to upsert taxonomy and links

func (s *Store) upsertName(ctx context.Context, table string, name string) (int64, error) {
	var id int64
	err := s.DB.QueryRowContext(ctx, "SELECT id FROM "+table+" WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := s.DB.ExecContext(ctx, "INSERT INTO "+table+"(name) VALUES(?)", name)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	return id, err
}

func (s *Store) setLinks(ctx context.Context, table string, linkCol string, todoID int64, names []string) error {
	// delete existing
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM "+table+" WHERE todo_id = ?", todoID); err != nil {
		return err
	}
	// insert new
	for _, n := range names {
		var id int64
		var err error
		switch table {
		case "todo_projects":
			id, err = s.upsertName(ctx, "projects", n)
		case "todo_contexts":
			id, err = s.upsertName(ctx, "contexts", n)
		case "todo_tags":
			id, err = s.upsertName(ctx, "tags", n)
		}
		if err != nil {
			return err
		}
		if _, err := s.DB.ExecContext(ctx, "INSERT OR IGNORE INTO "+table+"(todo_id, "+linkCol+") VALUES(?,?)", todoID, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) hydrate(ctx context.Context, t *Item) error {
	// Initialize empty slices to avoid null in JSON
	t.Projects = []string{}
	t.Contexts = []string{}
	t.Tags = []string{}

	// projects
	rows, _ := s.DB.QueryContext(ctx, `SELECT p.name FROM projects p JOIN todo_projects tp ON tp.project_id=p.id WHERE tp.todo_id=?`, t.ID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			t.Projects = append(t.Projects, name)
		}
	}
	// contexts
	rows2, _ := s.DB.QueryContext(ctx, `SELECT c.name FROM contexts c JOIN todo_contexts tc ON tc.context_id=c.id WHERE tc.todo_id=?`, t.ID)
	defer func() {
		if rows2 != nil {
			rows2.Close()
		}
	}()
	for rows2.Next() {
		var name string
		if err := rows2.Scan(&name); err == nil {
			t.Contexts = append(t.Contexts, name)
		}
	}
	// tags
	rows3, _ := s.DB.QueryContext(ctx, `SELECT t2.name FROM tags t2 JOIN todo_tags tt ON tt.tag_id=t2.id WHERE tt.todo_id=?`, t.ID)
	defer func() {
		if rows3 != nil {
			rows3.Close()
		}
	}()
	for rows3.Next() {
		var name string
		if err := rows3.Scan(&name); err == nil {
			t.Tags = append(t.Tags, name)
		}
	}
	return nil
}

// CreateItem inserts an item and writes its FTS5 entry. The caller supplies
// notes_doc (PM JSON, source of truth) and optionally notes_html (write-time
// render cache). FTS5 indexable text is derived from notes_doc.
func (s *Store) CreateItem(ctx context.Context, t *Item) (*Item, error) {
	ctx, span := feotel.StartSpan(ctx, "nil.item.create")
	defer span.End()

	if t.Section == "" {
		t.Section = "anytime"
	}
	kind, err := s.validateKindOrDefault(ctx, t.Kind)
	if err != nil {
		return nil, err
	}
	t.Kind = kind
	if strings.TrimSpace(t.Title) == "" {
		t.Inbox = true
	}
	res, err := s.DB.ExecContext(ctx, `
INSERT INTO todos(title, priority, completed, archived, due_at, threshold_at, recurrence_rule, source_line, notes_doc, notes_html, notes_html_version, section, pinned, kind, inbox, api_source)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.Title, t.Priority, t.Completed, t.Archived, t.DueAt, t.Threshold, t.Recur, t.Source,
		t.NotesDoc, t.NotesHTML, t.NotesHTMLVersion,
		t.Section, t.Pinned, t.Kind, t.Inbox, t.APISource,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	t.ID = id

	// set links
	if err := s.setLinks(ctx, "todo_projects", "project_id", id, t.Projects); err != nil {
		return nil, err
	}
	if err := s.setLinks(ctx, "todo_contexts", "context_id", id, t.Contexts); err != nil {
		return nil, err
	}
	if err := s.setLinks(ctx, "todo_tags", "tag_id", id, t.Tags); err != nil {
		return nil, err
	}

	if err := s.updateFTS(ctx, id, t.Title, derivePlainText(t.NotesDoc)); err != nil {
		return nil, err
	}
	// Sync refs for any wikilinks present in notes_doc.
	if refIDs := ingest.ExtractRefIDs(t.NotesDoc); len(refIDs) > 0 {
		if err := s.UpdateRefs(ctx, id, refIDs); err != nil {
			return nil, err
		}
	}

	_ = s.hydrate(ctx, t)
	return t, nil
}

func (s *Store) UpdateItem(ctx context.Context, t *Item) error {
	ctx, span := feotel.StartSpan(ctx, "nil.item.update")
	defer span.End()

	if t.Section == "" {
		t.Section = "anytime"
	}
	kind, err := s.validateKindOrDefault(ctx, t.Kind)
	if err != nil {
		return err
	}
	t.Kind = kind
	_, err = s.DB.ExecContext(ctx, `
UPDATE todos SET title=?, priority=?, completed=?, archived=?, due_at=?, threshold_at=?, recurrence_rule=?, notes_doc=?, notes_html=?, notes_html_version=?, section=?, pinned=?, kind=?, inbox=? WHERE id=?`,
		t.Title, t.Priority, t.Completed, t.Archived, t.DueAt, t.Threshold, t.Recur,
		t.NotesDoc, t.NotesHTML, t.NotesHTMLVersion,
		t.Section, t.Pinned, t.Kind, t.Inbox, t.ID,
	)
	if err != nil {
		return err
	}
	if err := s.setLinks(ctx, "todo_projects", "project_id", t.ID, t.Projects); err != nil {
		return err
	}
	if err := s.setLinks(ctx, "todo_contexts", "context_id", t.ID, t.Contexts); err != nil {
		return err
	}
	if err := s.setLinks(ctx, "todo_tags", "tag_id", t.ID, t.Tags); err != nil {
		return err
	}
	if err := s.updateFTS(ctx, t.ID, t.Title, derivePlainText(t.NotesDoc)); err != nil {
		return err
	}
	refIDs := ingest.ExtractRefIDs(t.NotesDoc)
	if err := s.UpdateRefs(ctx, t.ID, refIDs); err != nil {
		return err
	}
	return nil
}

// GetItem returns a single item by ID.
func (s *Store) GetItem(ctx context.Context, id int64) (*Item, error) {
	ctx, span := feotel.StartSpan(ctx, "nil.item.get")
	defer span.End()

	var t Item
	var pri *string
	var source sql.NullString
	var notesDoc, notesHTML sql.NullString
	var notesHTMLVersion sql.NullInt64
	var apiSource sql.NullString
	err := s.DB.QueryRowContext(ctx, `
SELECT id, title, priority, completed, archived, created_at, updated_at, due_at, threshold_at, recurrence_rule, source_line, notes_doc, notes_html, notes_html_version, section, pinned, kind, inbox, api_source
FROM todos WHERE id=?`, id).Scan(
		&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt,
		&t.DueAt, &t.Threshold, &t.Recur, &source,
		&notesDoc, &notesHTML, &notesHTMLVersion,
		&t.Section, &t.Pinned, &t.Kind, &t.Inbox, &apiSource,
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
	if t.Section == "" {
		t.Section = "anytime"
	}
	if err := s.hydrate(ctx, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateRefs replaces all outgoing refs from sourceID with targetIDs.
func (s *Store) UpdateRefs(ctx context.Context, sourceID int64, targetIDs []int64) error {
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM refs WHERE source_id = ?", sourceID); err != nil {
		return err
	}
	for _, targetID := range targetIDs {
		if targetID == sourceID {
			continue // skip self-references
		}
		if _, err := s.DB.ExecContext(ctx, "INSERT OR IGNORE INTO refs (source_id, target_id) VALUES (?, ?)", sourceID, targetID); err != nil {
			return err
		}
	}
	return nil
}

// GetBackrefs returns all items that reference targetID.
func (s *Store) GetBackrefs(ctx context.Context, targetID int64) ([]Item, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT t.id, t.title, t.priority, t.completed, t.archived, t.created_at, t.updated_at,
       t.due_at, t.threshold_at, t.recurrence_rule, t.source_line, t.notes_doc, t.notes_html, t.notes_html_version, t.section, t.pinned, t.kind, t.inbox, t.api_source
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
		var apiSource sql.NullString
		err := rows.Scan(&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt,
			&t.DueAt, &t.Threshold, &t.Recur, &source,
			&notesDoc, &notesHTML, &notesHTMLVersion,
			&t.Section, &t.Pinned, &t.Kind, &t.Inbox, &apiSource)
		if err != nil {
			return []Item{}, err
		}
		t.Priority = pri
		t.Source = source.String
		t.NotesDoc = notesDoc.String
		t.NotesHTML = notesHTML.String
		t.NotesHTMLVersion = int(notesHTMLVersion.Int64)
		t.APISource = apiSource.String
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

type qparts struct {
	where []string
	args  []any
	joins []string
	order string
	limit string
}

func (s *Store) Search(ctx context.Context, req SearchRequest) ([]Item, error) {
	ctx, span := feotel.StartSpan(ctx, "nil.item.search")
	defer span.End()

	q := qparts{}
	q.where = append(q.where, "(1=1)")

	// Exclude inbox items from normal search results unless explicitly requested
	if !req.IncludeInbox {
		q.where = append(q.where, "t.inbox = 0")
	}

	// kind filter (default to 'todo' for backward compatibility; "all" skips filter)
	kindFilter := req.Kind
	if kindFilter == "" {
		kindFilter = "todo"
	}
	if kindFilter != "all" {
		q.where = append(q.where, "t.kind = ?")
		q.args = append(q.args, kindFilter)
	}

	// updated_since: bulk/incremental-sync filter. Empty means no filter
	// (existing default behavior is unchanged). Input is RFC3339; stored
	// updated_at is naive UTC "YYYY-MM-DD HH:MM:SS" (see todos_update_ts
	// trigger in schema.sql), so we normalize before the string comparison —
	// SQLite's TEXT datetime format sorts/compares correctly lexicographically
	// once both sides share the same shape.
	if strings.TrimSpace(req.UpdatedSince) != "" {
		since, err := time.Parse(time.RFC3339, strings.TrimSpace(req.UpdatedSince))
		if err != nil {
			return []Item{}, fmt.Errorf("%w: %q: %w", ErrInvalidUpdatedSince, req.UpdatedSince, err)
		}
		q.where = append(q.where, "t.updated_at >= ?")
		q.args = append(q.args, since.UTC().Format("2006-01-02 15:04:05"))
	}

	// statuses
	for _, st := range req.Statuses {
		switch strings.ToLower(st) {
		case "open":
			q.where = append(q.where, "(completed=0 AND archived=0)")
		case "completed":
			q.where = append(q.where, "completed=1")
		case "archived":
			q.where = append(q.where, "archived=1")
		case "overdue":
			q.where = append(q.where, "(completed=0 AND archived=0 AND due_at < date('now'))")
		case "today":
			q.where = append(q.where, "date(due_at)=date('now')")
		}
	}

	// priorities
	if len(req.Priorities) > 0 {
		place := strings.Repeat("?,", len(req.Priorities))
		place = place[:len(place)-1]
		q.where = append(q.where, "priority IN ("+place+")")
		for _, p := range req.Priorities {
			q.args = append(q.args, p)
		}
	}

	// keywords via FTS
	if strings.TrimSpace(req.Query) != "" {
		q.joins = append(q.joins, "JOIN todos_fts ON todos_fts.rowid=t.id")
		q.where = append(q.where, "todos_fts MATCH ?")

		// If query is quoted, use exact match. Otherwise, add wildcard for prefix matching
		query := strings.TrimSpace(req.Query)
		if strings.HasPrefix(query, `"`) && strings.HasSuffix(query, `"`) {
			// Quoted search - exact match
			q.args = append(q.args, query)
		} else {
			// Add wildcard to each word for prefix matching
			words := strings.Fields(query)
			for i, word := range words {
				if !strings.HasSuffix(word, "*") {
					words[i] = word + "*"
				}
			}
			q.args = append(q.args, strings.Join(words, " "))
		}
	}

	// taxonomy filters
	if len(req.Projects) > 0 {
		place := strings.Repeat("?,", len(req.Projects))
		place = place[:len(place)-1]
		q.where = append(q.where, `EXISTS(SELECT 1 FROM todo_projects tp JOIN projects p ON p.id=tp.project_id WHERE tp.todo_id=t.id AND p.name IN (`+place+`))`)
		for _, v := range req.Projects {
			q.args = append(q.args, v)
		}
	}
	if len(req.Contexts) > 0 {
		place := strings.Repeat("?,", len(req.Contexts))
		place = place[:len(place)-1]
		q.where = append(q.where, `EXISTS(SELECT 1 FROM todo_contexts tc JOIN contexts c ON c.id=tc.context_id WHERE tc.todo_id=t.id AND c.name IN (`+place+`))`)
		for _, v := range req.Contexts {
			q.args = append(q.args, v)
		}
	}
	if len(req.Tags) > 0 {
		place := strings.Repeat("?,", len(req.Tags))
		place = place[:len(place)-1]
		q.where = append(q.where, `EXISTS(SELECT 1 FROM todo_tags tt JOIN tags tg ON tg.id=tt.tag_id WHERE tt.todo_id=t.id AND tg.name IN (`+place+`))`)
		for _, v := range req.Tags {
			q.args = append(q.args, v)
		}
	}

	// sorting / paging
	if req.SortBy == "" {
		req.SortBy = "created_at"
	}
	if strings.ToLower(req.SortDir) != "asc" {
		req.SortDir = "desc"
	}
	q.order = " ORDER BY t." + req.SortBy + " " + req.SortDir
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	offset := 0
	if req.Page > 0 {
		offset = req.Page * req.PageSize
	}
	q.limit = " LIMIT ? OFFSET ?"
	q.args = append(q.args, req.PageSize, offset)

	sqlStr := "SELECT t.id, t.title, t.priority, t.completed, t.archived, t.created_at, t.updated_at, t.due_at, t.threshold_at, t.recurrence_rule, t.source_line, t.notes_doc, t.notes_html, t.notes_html_version, t.section, t.pinned, t.kind, t.inbox, t.api_source FROM todos t "
	if len(q.joins) > 0 {
		sqlStr += strings.Join(q.joins, " ") + " "
	}
	sqlStr += " WHERE " + strings.Join(q.where, " AND ") + q.order + q.limit

	rows, err := s.DB.QueryContext(ctx, sqlStr, q.args...)
	if err != nil {
		return []Item{}, err
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var t Item
		var pri *string
		var section string
		var source sql.NullString
		var notesDoc, notesHTML sql.NullString
		var notesHTMLVersion sql.NullInt64
		var apiSource sql.NullString
		err := rows.Scan(&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt, &t.DueAt, &t.Threshold, &t.Recur, &source,
			&notesDoc, &notesHTML, &notesHTMLVersion,
			&section, &t.Pinned, &t.Kind, &t.Inbox, &apiSource)
		if err != nil {
			return []Item{}, err
		}
		t.Priority = pri
		t.Section = section
		t.Source = source.String
		t.NotesDoc = notesDoc.String
		t.NotesHTML = notesHTML.String
		t.NotesHTMLVersion = int(notesHTMLVersion.Int64)
		t.APISource = apiSource.String
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

// GetInboxCount returns the count of non-archived inbox items.
func (s *Store) GetInboxCount(ctx context.Context) (int, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos WHERE inbox = 1 AND archived = 0").Scan(&count)
	return count, err
}

// GetInboxItems returns inbox items, optionally filtered by a keyword query.
func (s *Store) GetInboxItems(ctx context.Context, req SearchRequest) ([]Item, error) {
	q := qparts{}
	q.where = append(q.where, "t.inbox = 1")
	q.where = append(q.where, "t.archived = 0")

	// keywords via FTS
	if strings.TrimSpace(req.Query) != "" {
		q.joins = append(q.joins, "JOIN todos_fts ON todos_fts.rowid=t.id")
		q.where = append(q.where, "todos_fts MATCH ?")
		words := strings.Fields(strings.TrimSpace(req.Query))
		for i, word := range words {
			if !strings.HasSuffix(word, "*") {
				words[i] = word + "*"
			}
		}
		q.args = append(q.args, strings.Join(words, " "))
	}

	// paging
	if req.PageSize <= 0 {
		req.PageSize = 200
	}
	offset := 0
	if req.Page > 0 {
		offset = req.Page * req.PageSize
	}
	q.order = " ORDER BY t.created_at DESC"
	q.limit = " LIMIT ? OFFSET ?"
	q.args = append(q.args, req.PageSize, offset)

	sqlStr := "SELECT t.id, t.title, t.priority, t.completed, t.archived, t.created_at, t.updated_at, t.due_at, t.threshold_at, t.recurrence_rule, t.source_line, t.notes_doc, t.notes_html, t.notes_html_version, t.section, t.pinned, t.kind, t.inbox, t.api_source FROM todos t "
	if len(q.joins) > 0 {
		sqlStr += strings.Join(q.joins, " ") + " "
	}
	sqlStr += " WHERE " + strings.Join(q.where, " AND ") + q.order + q.limit

	rows, err := s.DB.QueryContext(ctx, sqlStr, q.args...)
	if err != nil {
		return []Item{}, err
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var t Item
		var pri *string
		var section string
		var source sql.NullString
		var notesDoc, notesHTML sql.NullString
		var notesHTMLVersion sql.NullInt64
		var apiSource sql.NullString
		err := rows.Scan(&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt, &t.DueAt, &t.Threshold, &t.Recur, &source,
			&notesDoc, &notesHTML, &notesHTMLVersion,
			&section, &t.Pinned, &t.Kind, &t.Inbox, &apiSource)
		if err != nil {
			return []Item{}, err
		}
		t.Priority = pri
		t.Section = section
		t.Source = source.String
		t.NotesDoc = notesDoc.String
		t.NotesHTML = notesHTML.String
		t.NotesHTMLVersion = int(notesHTMLVersion.Int64)
		t.APISource = apiSource.String
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

// ProcessInboxItem clears the inbox flag for the given item.
func (s *Store) ProcessInboxItem(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE todos SET inbox = 0 WHERE id = ?", id)
	return err
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

// GetTaxonomyWithCounts returns all projects, contexts, and tags with the number
// of items that reference each one.
func (s *Store) GetTaxonomyWithCounts(ctx context.Context) (projects, contexts, tags []TaxonomyItem, err error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT p.name, COUNT(tp.todo_id) FROM projects p
LEFT JOIN todo_projects tp ON tp.project_id = p.id
GROUP BY p.id, p.name ORDER BY COUNT(tp.todo_id) DESC, p.name`)
	if err != nil {
		return
	}
	for rows.Next() {
		var ti TaxonomyItem
		rows.Scan(&ti.Name, &ti.Count)
		projects = append(projects, ti)
	}
	rows.Close()

	rows, err = s.DB.QueryContext(ctx, `
SELECT c.name, COUNT(tc.todo_id) FROM contexts c
LEFT JOIN todo_contexts tc ON tc.context_id = c.id
GROUP BY c.id, c.name ORDER BY COUNT(tc.todo_id) DESC, c.name`)
	if err != nil {
		return
	}
	for rows.Next() {
		var ti TaxonomyItem
		rows.Scan(&ti.Name, &ti.Count)
		contexts = append(contexts, ti)
	}
	rows.Close()

	rows, err = s.DB.QueryContext(ctx, `
SELECT t.name, COUNT(tt.todo_id) FROM tags t
LEFT JOIN todo_tags tt ON tt.tag_id = t.id
GROUP BY t.id, t.name ORDER BY COUNT(tt.todo_id) DESC, t.name`)
	if err != nil {
		return
	}
	for rows.Next() {
		var ti TaxonomyItem
		rows.Scan(&ti.Name, &ti.Count)
		tags = append(tags, ti)
	}
	rows.Close()
	return
}

func (s *Store) GetFilterValues(ctx context.Context) (projects, contexts, tags []string, err error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT name FROM projects ORDER BY name")
	if err != nil {
		return
	}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		projects = append(projects, n)
	}
	rows.Close()
	rows, err = s.DB.QueryContext(ctx, "SELECT name FROM contexts ORDER BY name")
	if err != nil {
		return
	}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		contexts = append(contexts, n)
	}
	rows.Close()
	rows, err = s.DB.QueryContext(ctx, "SELECT name FROM tags ORDER BY name")
	if err != nil {
		return
	}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tags = append(tags, n)
	}
	rows.Close()
	return
}

// updateFTS writes (or replaces) the FTS5 row for an item. Called after
// CreateItem and UpdateItem since v8 dropped the trigger-based sync in favor
// of standalone FTS5 with app-layer writes.
func (s *Store) updateFTS(ctx context.Context, id int64, title, text string) error {
	if _, err := s.DB.ExecContext(ctx, "DELETE FROM todos_fts WHERE rowid = ?", id); err != nil {
		return fmt.Errorf("updateFTS delete: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, "INSERT INTO todos_fts(rowid, title, notes_text) VALUES (?,?,?)", id, title, text); err != nil {
		return fmt.Errorf("updateFTS insert: %w", err)
	}
	return nil
}

// deleteFTS removes the FTS5 row for a deleted item.
func (s *Store) deleteFTS(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM todos_fts WHERE rowid = ?", id)
	return err
}

// derivePlainText returns the text projection used for FTS5 indexing. Walks
// the PM JSON doc tree concatenating text nodes; returns empty for an empty
// or unparseable doc (FTS row will then match by title only).
func derivePlainText(notesDoc string) string {
	if notesDoc == "" {
		return ""
	}
	text, err := ingest.DocToPlainText(notesDoc)
	if err != nil {
		return ""
	}
	return text
}

// IsValidKind returns true if the named kind exists in the kinds registry.
// Validation lives at the Go boundary; the SQL layer does not enforce a FK
// against the registry in this release (deferred to follow-up).
func (s *Store) IsValidKind(ctx context.Context, name string) (bool, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM kinds WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ErrInvalidKind is returned by CreateItem / UpdateItem when the supplied
// kind isn't in the kinds registry. Callers should surface this as a 4xx /
// validation error rather than a 5xx.
var ErrInvalidKind = errors.New("invalid kind: not in kinds registry")

// ErrInvalidUpdatedSince is returned (wrapped, via fmt.Errorf's %w) by Search
// when SearchRequest.UpdatedSince is non-empty and not valid RFC3339. Callers
// (apiserver) use errors.Is to map this to a 400 rather than a 500.
var ErrInvalidUpdatedSince = errors.New("invalid updated_since: want RFC3339 (e.g. 2026-08-01T00:00:00Z)")

// validateKindOrDefault enforces that t.Kind matches a registered kind.
// Empty kind is filled with "todo" as the canonical default. An unknown
// non-empty kind is rejected outright — we'd rather block the write than
// silently coerce, which would hide caller typos.
func (s *Store) validateKindOrDefault(ctx context.Context, kind string) (string, error) {
	if kind == "" {
		kind = "todo"
	}
	ok, err := s.IsValidKind(ctx, kind)
	if err != nil {
		return "", fmt.Errorf("validate kind: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrInvalidKind, kind)
	}
	return kind, nil
}

// ListKinds returns all registered kinds (core first, then plugin-registered).
func (s *Store) ListKinds(ctx context.Context) ([]Kind, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, name, display_name, icon, default_view, plugin_id, is_core, created_at, updated_at
FROM kinds ORDER BY is_core DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Kind
	for rows.Next() {
		var k Kind
		var pluginID sql.NullString
		if err := rows.Scan(&k.ID, &k.Name, &k.DisplayName, &k.Icon, &k.DefaultView, &pluginID, &k.IsCore, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		if pluginID.Valid {
			s := pluginID.String
			k.PluginID = &s
		}
		out = append(out, k)
	}
	if out == nil {
		out = []Kind{}
	}
	return out, nil
}

