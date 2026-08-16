package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// -- Helpers to upsert taxonomy and links

func upsertNameTx(ctx context.Context, dbx dbtx, table string, name string) (int64, error) {
	var id int64
	err := dbx.QueryRowContext(ctx, "SELECT id FROM "+table+" WHERE name = ?", name).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var res sql.Result
		res, err = dbx.ExecContext(ctx, "INSERT INTO "+table+"(name) VALUES(?)", name)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	return id, err
}

func setLinksTx(ctx context.Context, dbx dbtx, table string, linkCol string, todoID int64, names []string) error {
	// delete existing
	if _, err := dbx.ExecContext(ctx, "DELETE FROM "+table+" WHERE todo_id = ?", todoID); err != nil {
		return err
	}
	// insert new
	for _, n := range names {
		var id int64
		var err error
		switch table {
		case "todo_projects":
			id, err = upsertNameTx(ctx, dbx, "projects", n)
		case "todo_contexts":
			id, err = upsertNameTx(ctx, dbx, "contexts", n)
		case "todo_tags":
			id, err = upsertNameTx(ctx, dbx, "tags", n)
		}
		if err != nil {
			return err
		}
		if _, err := dbx.ExecContext(ctx, "INSERT OR IGNORE INTO "+table+"(todo_id, "+linkCol+") VALUES(?,?)", todoID, id); err != nil {
			return err
		}
	}
	return nil
}

func hydrateTx(ctx context.Context, dbx dbtx, t *Item) error {
	// Initialize empty slices to avoid null in JSON
	t.Projects = []string{}
	t.Contexts = []string{}
	t.Tags = []string{}

	// projects
	rows, _ := dbx.QueryContext(ctx, `SELECT p.name FROM projects p JOIN todo_projects tp ON tp.project_id=p.id WHERE tp.todo_id=?`, t.ID)
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			t.Projects = append(t.Projects, name)
		}
	}
	// contexts
	rows2, _ := dbx.QueryContext(ctx, `SELECT c.name FROM contexts c JOIN todo_contexts tc ON tc.context_id=c.id WHERE tc.todo_id=?`, t.ID)
	defer func() {
		if rows2 != nil {
			_ = rows2.Close()
		}
	}()
	for rows2.Next() {
		var name string
		if err := rows2.Scan(&name); err == nil {
			t.Contexts = append(t.Contexts, name)
		}
	}
	// tags
	rows3, _ := dbx.QueryContext(ctx, `SELECT t2.name FROM tags t2 JOIN todo_tags tt ON tt.tag_id=t2.id WHERE tt.todo_id=?`, t.ID)
	defer func() {
		if rows3 != nil {
			_ = rows3.Close()
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

func (s *Store) hydrate(ctx context.Context, t *Item) error {
	return hydrateTx(ctx, s.DB, t)
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
		_ = rows.Scan(&ti.Name, &ti.Count)
		projects = append(projects, ti)
	}
	_ = rows.Close()

	rows, err = s.DB.QueryContext(ctx, `
SELECT c.name, COUNT(tc.todo_id) FROM contexts c
LEFT JOIN todo_contexts tc ON tc.context_id = c.id
GROUP BY c.id, c.name ORDER BY COUNT(tc.todo_id) DESC, c.name`)
	if err != nil {
		return
	}
	for rows.Next() {
		var ti TaxonomyItem
		_ = rows.Scan(&ti.Name, &ti.Count)
		contexts = append(contexts, ti)
	}
	_ = rows.Close()

	rows, err = s.DB.QueryContext(ctx, `
SELECT t.name, COUNT(tt.todo_id) FROM tags t
LEFT JOIN todo_tags tt ON tt.tag_id = t.id
GROUP BY t.id, t.name ORDER BY COUNT(tt.todo_id) DESC, t.name`)
	if err != nil {
		return
	}
	for rows.Next() {
		var ti TaxonomyItem
		_ = rows.Scan(&ti.Name, &ti.Count)
		tags = append(tags, ti)
	}
	_ = rows.Close()
	return
}

func (s *Store) GetFilterValues(ctx context.Context) (projects, contexts, tags []string, err error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT name FROM projects ORDER BY name")
	if err != nil {
		return
	}
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		projects = append(projects, n)
	}
	_ = rows.Close()
	rows, err = s.DB.QueryContext(ctx, "SELECT name FROM contexts ORDER BY name")
	if err != nil {
		return
	}
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		contexts = append(contexts, n)
	}
	_ = rows.Close()
	rows, err = s.DB.QueryContext(ctx, "SELECT name FROM tags ORDER BY name")
	if err != nil {
		return
	}
	for rows.Next() {
		var n string
		_ = rows.Scan(&n)
		tags = append(tags, n)
	}
	_ = rows.Close()
	return
}

// IsValidKind returns true if the named kind exists in the kinds registry.
// Validation lives at the Go boundary; the SQL layer does not enforce a FK
// against the registry in this release (deferred to follow-up).
func (s *Store) IsValidKind(ctx context.Context, name string) (bool, error) {
	return isValidKindTx(ctx, s.DB, name)
}

func isValidKindTx(ctx context.Context, dbx dbtx, name string) (bool, error) {
	var count int
	err := dbx.QueryRowContext(ctx, "SELECT COUNT(*) FROM kinds WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ErrInvalidKind is returned by CreateItem / UpdateItem when the supplied
// kind isn't in the kinds registry. Callers should surface this as a 4xx /
// validation error rather than a 5xx.
var ErrInvalidKind = errors.New("invalid kind: not in kinds registry")

// validateKindOrDefault enforces that t.Kind matches a registered kind.
// Empty kind is filled with "todo" as the canonical default. An unknown
// non-empty kind is rejected outright — we'd rather block the write than
// silently coerce, which would hide caller typos.
func validateKindOrDefaultTx(ctx context.Context, dbx dbtx, kind string) (string, error) {
	if kind == "" {
		kind = "todo"
	}
	ok, err := isValidKindTx(ctx, dbx, kind)
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
	defer func() { _ = rows.Close() }()
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
