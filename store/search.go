package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	feotel "github.com/hollis-labs/go-otel"
)

// ErrInvalidUpdatedSince is returned (wrapped, via fmt.Errorf's %w) by Search
// when SearchRequest.UpdatedSince is non-empty and not valid RFC3339. Callers
// (apiserver) use errors.Is to map this to a 400 rather than a 500.
var ErrInvalidUpdatedSince = errors.New("invalid updated_since: want RFC3339 (e.g. 2026-08-01T00:00:00Z)")

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

	sqlStr := "SELECT t.id, t.title, t.priority, t.completed, t.archived, t.created_at, t.updated_at, t.due_at, t.threshold_at, t.recurrence_rule, t.source_line, t.notes_doc, t.notes_html, t.notes_html_version, t.section, t.pinned, t.kind, t.inbox, t.api_source, t.external_ref FROM todos t "
	if len(q.joins) > 0 {
		sqlStr += strings.Join(q.joins, " ") + " "
	}
	sqlStr += " WHERE " + strings.Join(q.where, " AND ") + q.order + q.limit //nolint:gosec // pre-existing: WHERE/ORDER fragments are static predicates built from qparts; all values are passed via placeholders in q.args, never concatenated directly. Out of scope for this file-split refactor.

	rows, err := s.DB.QueryContext(ctx, sqlStr, q.args...)
	if err != nil {
		return []Item{}, err
	}
	defer func() { _ = rows.Close() }()

	var out []Item
	for rows.Next() {
		var t Item
		var pri *string
		var section string
		var source sql.NullString
		var notesDoc, notesHTML sql.NullString
		var notesHTMLVersion sql.NullInt64
		var apiSource, externalRef sql.NullString
		err := rows.Scan(&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt, &t.DueAt, &t.Threshold, &t.Recur, &source,
			&notesDoc, &notesHTML, &notesHTMLVersion,
			&section, &t.Pinned, &t.Kind, &t.Inbox, &apiSource, &externalRef)
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

// ListItemIDs returns the id + updated_at of every item currently in the
// store — nothing else (no title, no notes_doc/notes_html, no taxonomy).
// It exists for the deletion/change-signal use case (see the epic's Option B
// decision, documented on the HTTP handler in apiserver.go): DeleteItem
// performs a real, hard DELETE with no tombstone, so an external sync
// consumer has no other way to learn an item is gone — it just silently
// stops appearing. This endpoint gives that consumer a cheap way to fetch
// the FULL current ID set and diff it against its own known-ID set: any ID
// it previously saw that's absent here has been genuinely deleted.
//
// Deliberately unfiltered by inbox/archived/completed status: none of those
// states mean "deleted" (the row still exists), so filtering on them here
// would produce false-positive deletion signals for a consumer that synced
// an item before it was archived, completed, or triaged into the inbox.
// Only a row's absence from the todos table (a genuine DELETE) should read
// as "deleted".
//
// kind: "" or "all" returns every kind (unlike Search's kind="" -> "todo"
// backward-compat default — there's no legacy caller to preserve here, and
// "does this ID exist" is naturally a whole-vault question). Any other
// value filters to just that kind, for a consumer that only tracks one
// kind's IDs. No validation against the kinds registry, matching Search's
// existing behavior — an unknown kind simply yields an empty result, not an
// error.
func (s *Store) ListItemIDs(ctx context.Context, kind string) ([]ItemIDStamp, error) {
	ctx, span := feotel.StartSpan(ctx, "nil.item.list_ids")
	defer span.End()

	q := "SELECT id, updated_at FROM todos"
	var args []any
	if kind != "" && kind != "all" {
		q += " WHERE kind = ?"
		args = append(args, kind)
	}
	q += " ORDER BY id"

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return []ItemIDStamp{}, err
	}
	defer func() { _ = rows.Close() }()

	var out []ItemIDStamp
	for rows.Next() {
		var it ItemIDStamp
		if err := rows.Scan(&it.ID, &it.UpdatedAt); err != nil {
			return []ItemIDStamp{}, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return []ItemIDStamp{}, err
	}
	if out == nil {
		return []ItemIDStamp{}, nil
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

	sqlStr := "SELECT t.id, t.title, t.priority, t.completed, t.archived, t.created_at, t.updated_at, t.due_at, t.threshold_at, t.recurrence_rule, t.source_line, t.notes_doc, t.notes_html, t.notes_html_version, t.section, t.pinned, t.kind, t.inbox, t.api_source, t.external_ref FROM todos t "
	if len(q.joins) > 0 {
		sqlStr += strings.Join(q.joins, " ") + " "
	}
	sqlStr += " WHERE " + strings.Join(q.where, " AND ") + q.order + q.limit //nolint:gosec // pre-existing: WHERE/ORDER fragments are static predicates built from qparts; all values are passed via placeholders in q.args, never concatenated directly. Out of scope for this file-split refactor.

	rows, err := s.DB.QueryContext(ctx, sqlStr, q.args...)
	if err != nil {
		return []Item{}, err
	}
	defer func() { _ = rows.Close() }()

	var out []Item
	for rows.Next() {
		var t Item
		var pri *string
		var section string
		var source sql.NullString
		var notesDoc, notesHTML sql.NullString
		var notesHTMLVersion sql.NullInt64
		var apiSource, externalRef sql.NullString
		err := rows.Scan(&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt, &t.DueAt, &t.Threshold, &t.Recur, &source,
			&notesDoc, &notesHTML, &notesHTMLVersion,
			&section, &t.Pinned, &t.Kind, &t.Inbox, &apiSource, &externalRef)
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

// ProcessInboxItem clears the inbox flag for the given item.
func (s *Store) ProcessInboxItem(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE todos SET inbox = 0 WHERE id = ?", id)
	return err
}
