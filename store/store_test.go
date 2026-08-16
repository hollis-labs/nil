package store

import (
	"context"
	"errors"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	st, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("opening test store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// insertWithUpdatedAt inserts a row directly (bypassing CreateItem) with an
// explicit updated_at, so the todos_update_ts trigger (which only fires
// AFTER UPDATE, not AFTER INSERT) doesn't stomp the value back to
// datetime('now'). This is how the updated_since tests get a row with a
// deterministic, old timestamp to filter against.
func insertWithUpdatedAt(t *testing.T, st *Store, title, kind, updatedAt string) int64 {
	t.Helper()
	res, err := st.DB.Exec(
		`INSERT INTO todos(title, kind, section, created_at, updated_at) VALUES(?, ?, 'anytime', ?, ?)`,
		title, kind, updatedAt, updatedAt,
	)
	if err != nil {
		t.Fatalf("insertWithUpdatedAt(%q): %v", title, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// TestSearchDefaultBehaviorUnchanged locks in the pre-existing store-level
// default: an empty Kind filters to "todo" (backward compat for callers that
// query the store directly, bypassing service/items.Service's "all"
// override), and an empty UpdatedSince applies no time filter at all —
// adding UpdatedSince support must not change either default.
func TestSearchDefaultBehaviorUnchanged(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	insertWithUpdatedAt(t, st, "old todo", "todo", "2020-01-01 00:00:00")
	if _, err := st.CreateItem(ctx, &Item{Title: "new todo", Kind: "todo"}); err != nil {
		t.Fatalf("CreateItem todo: %v", err)
	}
	if _, err := st.CreateItem(ctx, &Item{Title: "new note", Kind: "note"}); err != nil {
		t.Fatalf("CreateItem note: %v", err)
	}

	// No Kind, no UpdatedSince: defaults to kind="todo" (both todos
	// returned, note excluded), no time filter (old row not excluded).
	items, err := st.Search(ctx, SearchRequest{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2 (default kind=todo, no time filter); items=%+v", len(items), items)
	}
	for _, it := range items {
		if it.Kind != "todo" {
			t.Errorf("unexpected kind %q in default-kind results", it.Kind)
		}
	}
}

// TestSearchUpdatedSinceExcludesOlderRows is the core new-behavior test:
// updated_since should exclude rows updated before the given instant and
// include rows updated at/after it.
func TestSearchUpdatedSinceExcludesOlderRows(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	insertWithUpdatedAt(t, st, "ancient note", "note", "2020-01-01 00:00:00")
	newItem, err := st.CreateItem(ctx, &Item{Title: "recent note", Kind: "note"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	items, err := st.Search(ctx, SearchRequest{Kind: "all", UpdatedSince: "2024-01-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1 (only the recent row); items=%+v", len(items), items)
	}
	if items[0].ID != newItem.ID {
		t.Errorf("got item id=%d, want %d (recent note)", items[0].ID, newItem.ID)
	}

	// Sanity: without the filter, both rows are visible.
	all, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search (no filter): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d items without updated_since, want 2", len(all))
	}
}

// TestSearchUpdatedSinceInvalidFormat confirms a non-RFC3339 value fails
// loudly (wrapping ErrInvalidUpdatedSince) instead of silently ignoring the
// filter or panicking.
func TestSearchUpdatedSinceInvalidFormat(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	_, err := st.Search(ctx, SearchRequest{UpdatedSince: "not-a-timestamp"})
	if err == nil {
		t.Fatal("expected error for invalid updated_since, got nil")
	}
	if !errors.Is(err, ErrInvalidUpdatedSince) {
		t.Errorf("err=%v, want wrapping ErrInvalidUpdatedSince", err)
	}
}

// TestSearchKindAllReturnsEveryKind confirms kind="all" skips the kind
// filter entirely and returns items of every kind in one call — the
// cross-kind half of the bulk-sync acceptance criteria.
func TestSearchKindAllReturnsEveryKind(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if _, err := st.CreateItem(ctx, &Item{Title: "a todo", Kind: "todo"}); err != nil {
		t.Fatalf("CreateItem todo: %v", err)
	}
	if _, err := st.CreateItem(ctx, &Item{Title: "a note", Kind: "note"}); err != nil {
		t.Fatalf("CreateItem note: %v", err)
	}
	if _, err := st.CreateItem(ctx, &Item{Title: "a scratch", Kind: "scratch"}); err != nil {
		t.Fatalf("CreateItem scratch: %v", err)
	}

	items, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3 (kind=all across todo/note/scratch); items=%+v", len(items), items)
	}
	seen := map[string]bool{}
	for _, it := range items {
		seen[it.Kind] = true
	}
	for _, k := range []string{"todo", "note", "scratch"} {
		if !seen[k] {
			t.Errorf("kind=all results missing kind %q", k)
		}
	}
}
