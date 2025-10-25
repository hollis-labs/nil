package store

import (
	"context"
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

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
	dbPath := filepath.Join(dataDir, "todo.db?_fk=1")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better cloud sync compatibility
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return nil, err
	}

	// Migration: Add pinned column if it doesn't exist
	var columnExists bool
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='pinned'").Scan(&columnExists)
	if err == nil && !columnExists {
		if _, err := db.ExecContext(ctx, "ALTER TABLE todos ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0"); err != nil {
			return nil, err
		}
	}

	return &Store{DB: db}, nil
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

func (s *Store) hydrate(ctx context.Context, t *Todo) error {
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

// CreateTodo creates a todo with normalized links.
func (s *Store) CreateTodo(ctx context.Context, t *Todo) (*Todo, error) {
	if t.Section == "" {
		t.Section = "anytime"
	}
	res, err := s.DB.ExecContext(ctx, `
INSERT INTO todos(title, priority, completed, archived, due_at, threshold_at, recurrence_rule, source_line, notes_md, section, pinned)
VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		t.Title, t.Priority, t.Completed, t.Archived, t.DueAt, t.Threshold, t.Recur, t.Source, t.NotesMD, t.Section, t.Pinned,
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

	_ = s.hydrate(ctx, t)
	return t, nil
}

func (s *Store) UpdateTodo(ctx context.Context, t *Todo) error {
	if t.Section == "" {
		t.Section = "anytime"
	}
	_, err := s.DB.ExecContext(ctx, `
UPDATE todos SET title=?, priority=?, completed=?, archived=?, due_at=?, threshold_at=?, recurrence_rule=?, notes_md=?, section=?, pinned=? WHERE id=?`,
		t.Title, t.Priority, t.Completed, t.Archived, t.DueAt, t.Threshold, t.Recur, t.NotesMD, t.Section, t.Pinned, t.ID,
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
	return nil
}

func (s *Store) ToggleComplete(ctx context.Context, id int64, completed bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE todos SET completed=? WHERE id=?`, completed, id)
	return err
}
func (s *Store) Archive(ctx context.Context, id int64, archived bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE todos SET archived=? WHERE id=?`, archived, id)
	return err
}

func (s *Store) DeleteTodo(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM todos WHERE id=?`, id)
	return err
}

type qparts struct {
	where []string
	args  []any
	joins []string
	order string
	limit string
}

func (s *Store) Search(ctx context.Context, req SearchRequest) ([]Todo, error) {
	q := qparts{}
	q.where = append(q.where, "(1=1)")

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

	sqlStr := "SELECT t.id, t.title, t.priority, t.completed, t.archived, t.created_at, t.updated_at, t.due_at, t.threshold_at, t.recurrence_rule, t.source_line, t.notes_md, t.section, t.pinned FROM todos t "
	if len(q.joins) > 0 {
		sqlStr += strings.Join(q.joins, " ") + " "
	}
	sqlStr += " WHERE " + strings.Join(q.where, " AND ") + q.order + q.limit

	println("SQL Query:", sqlStr)
	rows, err := s.DB.QueryContext(ctx, sqlStr, q.args...)
	if err != nil {
		return []Todo{}, err
	}
	defer rows.Close()

	var out []Todo
	for rows.Next() {
		var t Todo
		var pri *string
		var section string
		err := rows.Scan(&t.ID, &t.Title, &pri, &t.Completed, &t.Archived, &t.CreatedAt, &t.UpdatedAt, &t.DueAt, &t.Threshold, &t.Recur, &t.Source, &t.NotesMD, &section, &t.Pinned)
		if err != nil {
			return []Todo{}, err
		}
		t.Priority = pri
		t.Section = section
		if t.Section == "" {
			t.Section = "anytime"
		}
		if err := s.hydrate(ctx, &t); err != nil {
			return []Todo{}, err
		}
		out = append(out, t)
	}
	if out == nil {
		return []Todo{}, nil
	}
	return out, nil
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
