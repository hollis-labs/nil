package store

import (
	"context"
	"errors"
	"testing"

	"github.com/hollis-labs/nil/ingest"
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

// wikilinkDoc builds a notes_doc PM-JSON string containing a single wikilink
// node pointing at targetID, matching the shape the TipTap WikilinkExtension
// produces client-side (see ingest.ExtractRefIDs / render.go).
func wikilinkDoc(t *testing.T, targetID int64, label string) string {
	t.Helper()
	doc := ingest.Node{
		Type: "doc",
		Content: []ingest.Node{
			{
				Type: "paragraph",
				Content: []ingest.Node{
					{Type: "text", Text: "See "},
					{Type: "wikilink", Attrs: map[string]any{
						"id":      targetID,
						"label":   label,
						"refType": "note",
					}},
				},
			},
		},
	}
	docJSON, err := ingest.MarshalDoc(doc)
	if err != nil {
		t.Fatalf("MarshalDoc: %v", err)
	}
	return docJSON
}

// TestGetBackrefsReturnsLinkingItems locks in GetBackrefs' contract: given
// item B wikilinks to item A, GetBackrefs(A) must return B. This is the same
// query the GUI's App.GetBackrefs binding uses (app.go), and what the HTTP
// API, CLI, and MCP surfaces added in CW-20260816-0038 all read through.
func TestGetBackrefsReturnsLinkingItems(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	target, err := st.CreateItem(ctx, &Item{Title: "Target note", Kind: "note"})
	if err != nil {
		t.Fatalf("CreateItem target: %v", err)
	}

	linker, err := st.CreateItem(ctx, &Item{
		Title:    "Linking note",
		Kind:     "note",
		NotesDoc: wikilinkDoc(t, target.ID, "Target note"),
	})
	if err != nil {
		t.Fatalf("CreateItem linker: %v", err)
	}

	// A third, unrelated item must not show up in target's backrefs.
	_, err = st.CreateItem(ctx, &Item{Title: "Unrelated note", Kind: "note"})
	if err != nil {
		t.Fatalf("CreateItem unrelated: %v", err)
	}

	backrefs, err := st.GetBackrefs(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetBackrefs: %v", err)
	}
	if len(backrefs) != 1 {
		t.Fatalf("got %d backrefs, want 1; backrefs=%+v", len(backrefs), backrefs)
	}
	if backrefs[0].ID != linker.ID {
		t.Errorf("backref ID=%d, want %d (the linking item)", backrefs[0].ID, linker.ID)
	}
	if backrefs[0].Title != "Linking note" {
		t.Errorf("backref Title=%q, want %q", backrefs[0].Title, "Linking note")
	}
}

// TestGetBackrefsEmptyWhenNoLinks confirms GetBackrefs returns an empty
// slice (not an error) when the target item exists but nothing links to it.
func TestGetBackrefsEmptyWhenNoLinks(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	target, err := st.CreateItem(ctx, &Item{Title: "Lonely note", Kind: "note"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	backrefs, err := st.GetBackrefs(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetBackrefs: %v", err)
	}
	if len(backrefs) != 0 {
		t.Fatalf("got %d backrefs, want 0; backrefs=%+v", len(backrefs), backrefs)
	}
}

// TestGetBackrefsUnknownTargetReturnsEmpty documents GetBackrefs' own
// behavior for a target ID that was never created: the refs join simply
// matches nothing, so it returns an empty slice, not an error. The
// not-found-vs-empty distinction ("does this item even exist?") is handled
// one layer up, at the HTTP API (handleGetBackrefs 404s if the target item
// itself doesn't exist) — see apiserver_test.go.
func TestGetBackrefsUnknownTargetReturnsEmpty(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	backrefs, err := st.GetBackrefs(ctx, 999999)
	if err != nil {
		t.Fatalf("GetBackrefs: %v", err)
	}
	if len(backrefs) != 0 {
		t.Fatalf("got %d backrefs, want 0; backrefs=%+v", len(backrefs), backrefs)
	}
}
