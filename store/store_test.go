package store

import (
	"context"
	"errors"
	"strings"
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

// idSet builds a set of the IDs present in an []ItemIDStamp for convenient
// membership checks.
func idSet(items []ItemIDStamp) map[int64]bool {
	out := make(map[int64]bool, len(items))
	for _, it := range items {
		out[it.ID] = true
	}
	return out
}

// TestListItemIDsReflectsCurrentExistence is the core deletion-detection
// contract this endpoint exists to provide: create items across kinds and
// statuses (a plain open todo, a completed todo, an archived note, an inbox
// item, and a scratch item), delete one of them, then confirm
// ListItemIDs' result is exactly "every item that still exists" — the
// deleted item's ID is absent, every other item's ID (regardless of
// completed/archived/inbox state) is present. This is precisely what an
// external sync consumer relies on to infer a hard DELETE occurred.
func TestListItemIDsReflectsCurrentExistence(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	plain, err := st.CreateItem(ctx, &Item{Title: "plain todo", Kind: "todo"})
	if err != nil {
		t.Fatalf("CreateItem plain: %v", err)
	}
	completed, err := st.CreateItem(ctx, &Item{Title: "completed todo", Kind: "todo", Completed: true})
	if err != nil {
		t.Fatalf("CreateItem completed: %v", err)
	}
	archived, err := st.CreateItem(ctx, &Item{Title: "archived note", Kind: "note", Archived: true})
	if err != nil {
		t.Fatalf("CreateItem archived: %v", err)
	}
	inboxItem, err := st.CreateItem(ctx, &Item{Title: "inbox capture", Kind: "todo", Inbox: true})
	if err != nil {
		t.Fatalf("CreateItem inbox: %v", err)
	}
	scratch, err := st.CreateItem(ctx, &Item{Title: "scratch pad", Kind: "scratch"})
	if err != nil {
		t.Fatalf("CreateItem scratch: %v", err)
	}
	toDelete, err := st.CreateItem(ctx, &Item{Title: "about to be deleted", Kind: "note"})
	if err != nil {
		t.Fatalf("CreateItem toDelete: %v", err)
	}

	// Sanity: before deletion, every created item's ID is present.
	before, err := st.ListItemIDs(ctx, "")
	if err != nil {
		t.Fatalf("ListItemIDs (before delete): %v", err)
	}
	beforeIDs := idSet(before)
	for _, want := range []int64{plain.ID, completed.ID, archived.ID, inboxItem.ID, scratch.ID, toDelete.ID} {
		if !beforeIDs[want] {
			t.Errorf("before delete: id %d missing from ListItemIDs result; got %+v", want, before)
		}
	}

	if err = st.DeleteItem(ctx, toDelete.ID); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}

	after, err := st.ListItemIDs(ctx, "")
	if err != nil {
		t.Fatalf("ListItemIDs (after delete): %v", err)
	}
	afterIDs := idSet(after)

	// The deleted item's ID must be genuinely gone.
	if afterIDs[toDelete.ID] {
		t.Errorf("deleted item id %d still present in ListItemIDs result: %+v", toDelete.ID, after)
	}

	// Every other item — including completed, archived, and inbox items,
	// none of which are "deleted" — must still be present.
	for _, want := range []int64{plain.ID, completed.ID, archived.ID, inboxItem.ID, scratch.ID} {
		if !afterIDs[want] {
			t.Errorf("after delete: id %d (still exists) missing from ListItemIDs result; got %+v", want, after)
		}
	}

	if len(after) != len(before)-1 {
		t.Errorf("got %d ids after delete, want %d (one fewer than before)", len(after), len(before)-1)
	}

	// updated_at must be populated (non-empty) on the surviving rows — the
	// whole point of this endpoint is (id, updated_at) pairs, not bare IDs.
	for _, it := range after {
		if strings.TrimSpace(it.UpdatedAt) == "" {
			t.Errorf("item id %d has empty updated_at in ListItemIDs result", it.ID)
		}
	}
}

// TestListItemIDsKindFilter confirms the kind filter narrows the result to
// just that kind, while "" and "all" both mean "every kind" (unlike
// Search's kind="" -> "todo" backward-compat default).
func TestListItemIDsKindFilter(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	todo, err := st.CreateItem(ctx, &Item{Title: "a todo", Kind: "todo"})
	if err != nil {
		t.Fatalf("CreateItem todo: %v", err)
	}
	note, err := st.CreateItem(ctx, &Item{Title: "a note", Kind: "note"})
	if err != nil {
		t.Fatalf("CreateItem note: %v", err)
	}
	scratch, err := st.CreateItem(ctx, &Item{Title: "a scratch", Kind: "scratch"})
	if err != nil {
		t.Fatalf("CreateItem scratch: %v", err)
	}

	todoOnly, err := st.ListItemIDs(ctx, "todo")
	if err != nil {
		t.Fatalf("ListItemIDs kind=todo: %v", err)
	}
	if len(todoOnly) != 1 || todoOnly[0].ID != todo.ID {
		t.Fatalf("kind=todo got %+v, want exactly [%d]", todoOnly, todo.ID)
	}

	for _, kind := range []string{"", "all"} {
		everything, err := st.ListItemIDs(ctx, kind)
		if err != nil {
			t.Fatalf("ListItemIDs kind=%q: %v", kind, err)
		}
		got := idSet(everything)
		for _, want := range []int64{todo.ID, note.ID, scratch.ID} {
			if !got[want] {
				t.Errorf("kind=%q missing id %d; got %+v", kind, want, everything)
			}
		}
		if len(everything) != 3 {
			t.Errorf("kind=%q got %d ids, want 3", kind, len(everything))
		}
	}
}
