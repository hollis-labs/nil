package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/hollis-labs/nil/ingest"
)

const currentSchemaVersion = 13

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
	{version: 12, sql: ""},
	{version: 13, sql: ""},
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	// Create schema_version table if it doesn't exist
	//
	// topErr (rather than the more idiomatic "err") is deliberate: this
	// function-scoped variable is reused (via "=", not ":=") at several
	// points below (the fresh-install fast path, the v12/v13 branches, and
	// the generic tail path). Every one of the many per-version branches in
	// the loop below declares its own block-scoped "err" via ":=" for its
	// own local checks — giving the function-scoped variable a distinct
	// name avoids those from shadowing this one (govet's shadow check),
	// without having to rename every one of those local, single-purpose
	// declarations instead.
	_, topErr := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if topErr != nil {
		return topErr
	}

	// Get current version
	var currentVersion int
	topErr = db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if topErr != nil {
		return topErr
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
			if _, iErr := db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", currentSchemaVersion); iErr != nil {
				return iErr
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
			cErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='pinned'").Scan(&count)
			if cErr == nil && count > 0 {
				_, cErr = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				if cErr != nil {
					return cErr
				}
				continue
			}
		}
		if m.version == 2 {
			var count int
			cErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='type'").Scan(&count)
			if cErr == nil && count > 0 {
				_, cErr = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				if cErr != nil {
					return cErr
				}
				continue
			}
		}
		if m.version == 3 {
			var count int
			cErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='refs'").Scan(&count)
			if cErr == nil && count > 0 {
				_, cErr = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				if cErr != nil {
					return cErr
				}
				continue
			}
		}
		if m.version == 4 {
			var count int
			cErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='inbox'").Scan(&count)
			if cErr == nil && count > 0 {
				_, cErr = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
				if cErr != nil {
					return cErr
				}
				continue
			}
		}
		if m.version == 5 {
			var count int
			cErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='api_source'").Scan(&count)
			if cErr == nil && count > 0 {
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
			_ = bRows.Close()
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
		if m.version == 12 {
			if topErr = migrateV12(ctx, db); topErr != nil {
				return topErr
			}
			if _, topErr = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version); topErr != nil {
				return topErr
			}
			continue
		}
		if m.version == 13 {
			if topErr = migrateV13CleanupOrphanedRefs(ctx, db); topErr != nil {
				return topErr
			}
			if _, topErr = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version); topErr != nil {
				return topErr
			}
			continue
		}

		// Run migration
		_, topErr = db.ExecContext(ctx, m.sql)
		if topErr != nil {
			return topErr
		}

		// Record migration
		_, topErr = db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", m.version)
		if topErr != nil {
			return topErr
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
// updateFTS (and DeleteItem's own inline FTS delete).
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
	_ = rows.Close()
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
		if scanErr := rows.Scan(&c.id, &c.title, &c.notesMD); scanErr != nil {
			scanErrs++
			fmt.Fprintf(os.Stderr, "migration v11: scan error on a todos row, skipping: %v\n", scanErr)
			continue
		}
		candidates = append(candidates, c)
	}
	_ = rows.Close()
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

// migrateV12 adds the external_ref column, following the exact same
// ALTER-TABLE-plus-existence-check idempotency pattern as migration v5's
// api_source column (see the `m.version == 5` branch above): add the column
// only if it isn't already there.
//
// The partial unique index on this column is deliberately NOT created here.
// It's created once in Open(), after runMigrations returns — see Open's doc
// comment for why: schemaSQL (which also declares the index's rationale in
// its comments) runs unconditionally before migrations on every Open() call,
// so creating the index inside this migration would still leave a window
// (this migration hasn't run yet on THIS call, but schemaSQL already tried
// to reference the column) if it were duplicated there too. Centralizing it
// in Open(), strictly after migrations complete, avoids that ordering
// hazard entirely.
func migrateV12(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='external_ref'").Scan(&count); err != nil {
		return fmt.Errorf("migration v12 check column: %w", err)
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, "ALTER TABLE todos ADD COLUMN external_ref TEXT DEFAULT NULL"); err != nil {
			return fmt.Errorf("migration v12 add column: %w", err)
		}
	}
	return nil
}

// migrateV13CleanupOrphanedRefs is a one-time data cleanup (not a schema
// shape change) for damage caused by CW-20260816-0059: Open's dbPath used
// to build the DSN with a "_fk=1" query param, which modernc.org/sqlite's
// applyQueryParams has never recognized (the real syntax is
// "_pragma=foreign_keys(1)"). Unrecognized params are silently ignored
// rather than erroring, so foreign key enforcement — which refs'
// "ON DELETE CASCADE" (schema.sql) depends on — was never actually active.
// DeleteItem's plain `DELETE FROM todos` therefore never cascaded to refs
// rows pointing at the deleted id, leaving orphans behind. This was
// confirmed against a real vault during the fix: 11 of 15 refs rows were
// orphaned. Now that foreign_keys is genuinely enabled (see Open), no new
// orphans can be created, so this only ever has real work to do once per
// vault; idempotent because a second run simply deletes zero rows.
func migrateV13CleanupOrphanedRefs(ctx context.Context, db *sql.DB) error {
	res, err := db.ExecContext(ctx, `
DELETE FROM refs
WHERE source_id NOT IN (SELECT id FROM todos)
   OR target_id NOT IN (SELECT id FROM todos)`)
	if err != nil {
		return fmt.Errorf("migration v13 cleanup orphaned refs: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		fmt.Fprintf(os.Stderr, "migration v13: removed %d orphaned refs row(s) left behind while foreign key enforcement was inactive (see CW-20260816-0059)\n", n)
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
