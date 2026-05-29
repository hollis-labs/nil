// nil-recover: backfill notes_doc + notes_html for rows that still have
// legacy notes_md content. Idempotent — only touches rows where
// notes_doc IS NULL OR notes_doc = ''. Run once per vault DB.
//
// Usage:
//   nil-recover <path-to-todo.db> [<path-to-todo.db> ...]
//
// Why this exists: the v1.3.0 migration moved notes content from HTML
// (notes_md column) to TipTap PM JSON (notes_doc column). The original
// design ran the conversion in the frontend on first launch, but that
// approach failed silently for at least one user — the "done" flag got
// set without the work happening. This tool runs the same conversion
// server-side via the ingest package and is safe to run repeatedly.

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hollis-labs/nil/ingest"
	"github.com/hollis-labs/nil/store"
	_ "modernc.org/sqlite"
)

const notesHTMLVersion = 1

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: nil-recover <path-to-todo.db> [<more.db> ...]")
		os.Exit(2)
	}

	ctx := context.Background()
	totalFixed := 0
	totalErrs := 0
	for _, path := range os.Args[1:] {
		fixed, errs, err := recover(ctx, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", path, err)
			totalErrs++
			continue
		}
		fmt.Printf("  %s: %d rows converted, %d row-level errors\n", path, fixed, errs)
		totalFixed += fixed
		totalErrs += errs
	}
	fmt.Printf("\nDone. %d rows converted across %d DB(s). %d errors.\n",
		totalFixed, len(os.Args)-1, totalErrs)
	if totalErrs > 0 {
		os.Exit(1)
	}
}

func recover(ctx context.Context, dbPath string) (fixed int, errCount int, err error) {
	abs, _ := filepath.Abs(dbPath)
	fmt.Printf("Opening %s\n", abs)

	// Apply pending migrations first by going through store.Open. That gives
	// us v7-v10 (notes_doc column, kind rename, kinds registry, standalone
	// FTS5) for any DB that hasn't been opened by the upgraded app yet.
	dir := filepath.Dir(dbPath)
	st, err := store.Open(ctx, dir)
	if err != nil {
		return 0, 0, fmt.Errorf("migrate via store.Open: %w", err)
	}
	st.Close()

	// Reopen raw — we want direct SQL access for the backfill loop without
	// going through store's CRUD helpers (which would also drag in
	// app-layer concerns we don't need here).
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return 0, 0, fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	// Sanity: notes_doc must exist now (post-migration).
	var hasNotesDoc int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='notes_doc'").Scan(&hasNotesDoc); err != nil {
		return 0, 0, fmt.Errorf("check schema: %w", err)
	}
	if hasNotesDoc == 0 {
		return 0, 0, fmt.Errorf("notes_doc column still missing after store.Open — migration logic may be broken")
	}

	rows, err := db.QueryContext(ctx, `
SELECT id, title, notes_md FROM todos
WHERE notes_md IS NOT NULL AND notes_md != ''
  AND (notes_doc IS NULL OR notes_doc = '')`)
	if err != nil {
		return 0, 0, fmt.Errorf("query candidates: %w", err)
	}

	type cand struct {
		id      int64
		title   string
		notesMD string
	}
	var candidates []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.title, &c.notesMD); err != nil {
			errCount++
			continue
		}
		candidates = append(candidates, c)
	}
	rows.Close()
	if len(candidates) == 0 {
		fmt.Println("  (no candidates — already up to date)")
		return 0, 0, nil
	}
	fmt.Printf("  %d candidates...\n", len(candidates))

	// Wrap conversion in a transaction so the FTS5 sync is atomic per item.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin tx: %w", err)
	}
	rolled := false
	defer func() {
		if rolled {
			return
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, c := range candidates {
		// notes_md is already HTML (it's TipTap's getHTML() output, despite the
		// misleading column name). Convert HTML → PM JSON; keep the original
		// HTML as the notes_html cache.
		doc, ierr := ingest.HTMLToDoc(c.notesMD)
		if ierr != nil {
			fmt.Fprintf(os.Stderr, "  id=%d: %v\n", c.id, ierr)
			errCount++
			continue
		}
		// Sanity check: must be valid JSON parseable into a doc node.
		var sanity any
		if jerr := json.Unmarshal([]byte(doc), &sanity); jerr != nil {
			fmt.Fprintf(os.Stderr, "  id=%d: produced invalid JSON: %v\n", c.id, jerr)
			errCount++
			continue
		}
		if _, uerr := tx.ExecContext(ctx, `
UPDATE todos SET notes_doc = ?, notes_html = ?, notes_html_version = ?
WHERE id = ?`, doc, c.notesMD, notesHTMLVersion, c.id); uerr != nil {
			fmt.Fprintf(os.Stderr, "  id=%d: update failed: %v\n", c.id, uerr)
			errCount++
			continue
		}
		// Refresh FTS5 from the converted doc — keeps search consistent.
		plain, _ := ingest.DocToPlainText(doc)
		_, _ = tx.ExecContext(ctx, "DELETE FROM todos_fts WHERE rowid = ?", c.id)
		_, _ = tx.ExecContext(ctx, "INSERT INTO todos_fts(rowid, title, notes_text) VALUES (?, ?, ?)", c.id, c.title, plain)
		fixed++
	}

	if cerr := tx.Commit(); cerr != nil {
		rolled = true
		return fixed, errCount, fmt.Errorf("commit: %w", cerr)
	}
	return fixed, errCount, nil
}
