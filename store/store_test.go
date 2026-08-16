package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
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

// TestForeignKeysPragmaEnabled locks in the fix for the DSN bug where
// Open's dbPath used the nonexistent "_fk=1" query param instead of the
// driver-recognized "_pragma=foreign_keys(1)" syntax. modernc.org/sqlite's
// applyQueryParams silently ignores unrecognized params rather than
// erroring, so "_fk=1" was a no-op and foreign key enforcement (which the
// refs table's ON DELETE CASCADE depends on) was never actually turned on.
// This queries PRAGMA foreign_keys directly rather than trusting that the
// DSN was merely accepted without error.
func TestForeignKeysPragmaEnabled(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	var enabled int
	if err := st.DB.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatalf("querying PRAGMA foreign_keys: %v", err)
	}
	if enabled != 1 {
		t.Fatalf("PRAGMA foreign_keys reported %d, want 1 (foreign key enforcement not active)", enabled)
	}
}

// TestDeleteItemCascadesRefs confirms that, with foreign key enforcement
// actually active (see TestForeignKeysPragmaEnabled), deleting a todos row
// that's the target of a refs row cascades per schema.sql's
// "ON DELETE CASCADE" instead of leaving an orphaned refs row behind. Before
// the "_fk=1" DSN fix, DeleteItem's plain `DELETE FROM todos` never
// triggered SQLite's cascade (foreign keys were off), so the refs row would
// have survived pointing at a now-nonexistent todos.id.
func TestDeleteItemCascadesRefs(t *testing.T) {
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

	// Confirm the refs row exists before the delete, so the post-delete
	// assertion actually proves a cascade happened (not just that there was
	// never a row to begin with).
	var preCount int
	if err := st.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM refs WHERE source_id = ? AND target_id = ?", linker.ID, target.ID).Scan(&preCount); err != nil {
		t.Fatalf("checking pre-delete refs row: %v", err)
	}
	if preCount != 1 {
		t.Fatalf("refs row not created by wikilink sync; preCount=%d", preCount)
	}

	if err := st.DeleteItem(ctx, target.ID); err != nil {
		t.Fatalf("DeleteItem(target): %v", err)
	}

	var postCount int
	if err := st.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM refs WHERE source_id = ? AND target_id = ?", linker.ID, target.ID).Scan(&postCount); err != nil {
		t.Fatalf("checking post-delete refs row: %v", err)
	}
	if postCount != 0 {
		t.Fatalf("refs row survived target deletion (ON DELETE CASCADE did not fire); postCount=%d", postCount)
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

// ---------------------------------------------------------------------------
// CW-20260816-0045: batch create + external_ref idempotent round-tripping
// ---------------------------------------------------------------------------

// TestCreateItemsBatchCreatesAll confirms a batch of N items are all
// created in one call, preserve field values, and land in the same order
// they were submitted.
func TestCreateItemsBatchCreatesAll(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	input := []Item{
		{Title: "batch item 1", Kind: "todo"},
		{Title: "batch item 2", Kind: "note"},
		{Title: "batch item 3", Kind: "scratch"},
	}
	created, err := st.CreateItemsBatch(ctx, input)
	if err != nil {
		t.Fatalf("CreateItemsBatch: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("got %d created items, want 3", len(created))
	}
	for i, want := range input {
		if created[i].Title != want.Title {
			t.Errorf("created[%d].Title=%q, want %q", i, created[i].Title, want.Title)
		}
		if created[i].Kind != want.Kind {
			t.Errorf("created[%d].Kind=%q, want %q", i, created[i].Kind, want.Kind)
		}
		if created[i].ID == 0 {
			t.Errorf("created[%d].ID is zero, want a real assigned ID", i)
		}
	}

	// Confirm they're actually persisted — visible via a fresh GetItem, not
	// just returned in the batch response.
	for _, c := range created {
		if _, err = st.GetItem(ctx, c.ID); err != nil {
			t.Errorf("GetItem(%d) after batch create: %v", c.ID, err)
		}
	}

	all, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d items in vault after batch create, want 3", len(all))
	}
}

// TestCreateItemsBatchEmptyInput confirms an empty batch is a no-op: no
// transaction side effects, empty (non-nil) result, no error.
func TestCreateItemsBatchEmptyInput(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	created, err := st.CreateItemsBatch(ctx, nil)
	if err != nil {
		t.Fatalf("CreateItemsBatch(nil): %v", err)
	}
	if created == nil {
		t.Error("CreateItemsBatch(nil) returned a nil slice, want empty non-nil")
	}
	if len(created) != 0 {
		t.Errorf("got %d items, want 0", len(created))
	}
}

// TestCreateItemsBatchAllOrNothing is the core partial-failure-handling
// contract: a batch where one item has an invalid kind must fail the WHOLE
// batch — none of the other, individually-valid items in that same call may
// be created. This locks in the all-or-nothing (not best-effort) design
// decision (see CreateItemsBatch's doc comment for the rationale).
func TestCreateItemsBatchAllOrNothing(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	before, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search (before): %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("got %d pre-existing items, want 0 (clean store)", len(before))
	}

	input := []Item{
		{Title: "valid item 1", Kind: "todo"},
		{Title: "invalid item", Kind: "not-a-real-kind"},
		{Title: "valid item 2", Kind: "note"},
	}
	created, err := st.CreateItemsBatch(ctx, input)
	if err == nil {
		t.Fatalf("CreateItemsBatch: expected an error for the invalid-kind item, got created=%+v", created)
	}
	if !errors.Is(err, ErrInvalidKind) {
		t.Errorf("err=%v, want it to wrap ErrInvalidKind", err)
	}
	if !strings.Contains(err.Error(), "item 1") {
		t.Errorf("err=%q, want it to name the failing item's index (item 1)", err.Error())
	}

	// Nothing from the batch — including the two individually-valid items —
	// may have been committed.
	after, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search (after): %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("got %d items after a failed all-or-nothing batch, want 0 (nothing committed); items=%+v", len(after), after)
	}
}

// TestCreateItemExternalRefIdempotentRepush is the core idempotent-write
// contract: creating an item with a given external_ref, then "re-pushing"
// (calling CreateItem again) with the same external_ref but different field
// values, must update the existing row in place — not insert a duplicate.
func TestCreateItemExternalRefIdempotentRepush(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	first, err := st.CreateItem(ctx, &Item{
		Title:       "original title",
		Kind:        "note",
		ExternalRef: "fe-doc-123",
		Tags:        []string{"draft"},
	})
	if err != nil {
		t.Fatalf("CreateItem (first push): %v", err)
	}

	countBefore, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search (before re-push): %v", err)
	}
	if len(countBefore) != 1 {
		t.Fatalf("got %d items after first push, want 1", len(countBefore))
	}

	second, err := st.CreateItem(ctx, &Item{
		Title:       "updated title",
		Kind:        "note",
		ExternalRef: "fe-doc-123",
		Tags:        []string{"final"},
	})
	if err != nil {
		t.Fatalf("CreateItem (re-push): %v", err)
	}

	// Same row, not a new one.
	if second.ID != first.ID {
		t.Errorf("re-push created a new row (id=%d), want the existing row (id=%d) updated in place", second.ID, first.ID)
	}
	if second.Title != "updated title" {
		t.Errorf("re-push Title=%q, want %q (fields should be overwritten)", second.Title, "updated title")
	}
	if len(second.Tags) != 1 || second.Tags[0] != "final" {
		t.Errorf("re-push Tags=%v, want [\"final\"]", second.Tags)
	}

	// Item count must not have grown.
	countAfter, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search (after re-push): %v", err)
	}
	if len(countAfter) != 1 {
		t.Fatalf("got %d items after re-push, want 1 (no duplicate); items=%+v", len(countAfter), countAfter)
	}

	// A fresh GetItem confirms the update was actually persisted, not just
	// reflected in the returned struct.
	refetched, err := st.GetItem(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetItem after re-push: %v", err)
	}
	if refetched.Title != "updated title" {
		t.Errorf("persisted Title=%q, want %q", refetched.Title, "updated title")
	}
}

// TestCreateItemExternalRefRepushPreservesCompletedAndArchived confirms the
// full-overwrite re-push semantics deliberately exclude completed/archived:
// items.CreateInput (what every create surface funnels through) has no
// representation for those fields, so a re-push must not silently undo a
// user's local completion/archival of the item.
func TestCreateItemExternalRefRepushPreservesCompletedAndArchived(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	created, err := st.CreateItem(ctx, &Item{Title: "task", Kind: "todo", ExternalRef: "fe-task-9"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if err = st.ToggleComplete(ctx, created.ID, true); err != nil {
		t.Fatalf("ToggleComplete: %v", err)
	}
	if err = st.Archive(ctx, created.ID, true); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	repushed, err := st.CreateItem(ctx, &Item{Title: "task (updated)", Kind: "todo", ExternalRef: "fe-task-9"})
	if err != nil {
		t.Fatalf("CreateItem (re-push): %v", err)
	}
	if repushed.ID != created.ID {
		t.Fatalf("re-push created a new row, want the same row updated in place")
	}
	if !repushed.Completed {
		t.Error("re-push cleared Completed; it must be preserved (not part of the create payload)")
	}
	if !repushed.Archived {
		t.Error("re-push cleared Archived; it must be preserved (not part of the create payload)")
	}
	if repushed.Title != "task (updated)" {
		t.Errorf("Title=%q, want the re-pushed value to have been applied", repushed.Title)
	}
}

// TestCreateItemNoExternalRefUnaffected confirms the default, most common
// path — no external_ref supplied — is completely unchanged: repeated plain
// creates never dedup against each other, even with identical titles/kinds.
func TestCreateItemNoExternalRefUnaffected(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := st.CreateItem(ctx, &Item{Title: "plain item", Kind: "todo"}); err != nil {
			t.Fatalf("CreateItem #%d: %v", i, err)
		}
	}

	all, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d items, want 3 distinct rows (no external_ref means no dedup)", len(all))
	}
}

// TestCreateItemsBatchExternalRefRepush confirms the batch path shares the
// exact same external_ref dedup semantics as a single CreateItem call: a
// batch that re-pushes a previously-seen external_ref updates that row
// in-place rather than duplicating it, alongside a genuinely new item in
// the same call.
func TestCreateItemsBatchExternalRefRepush(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	existing, err := st.CreateItem(ctx, &Item{Title: "already pushed", Kind: "note", ExternalRef: "fe-777"})
	if err != nil {
		t.Fatalf("CreateItem (seed): %v", err)
	}

	created, err := st.CreateItemsBatch(ctx, []Item{
		{Title: "already pushed (updated)", Kind: "note", ExternalRef: "fe-777"},
		{Title: "brand new item", Kind: "note"},
	})
	if err != nil {
		t.Fatalf("CreateItemsBatch: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("got %d results, want 2", len(created))
	}
	if created[0].ID != existing.ID {
		t.Errorf("batch entry matching external_ref got a new id=%d, want the existing id=%d", created[0].ID, existing.ID)
	}
	if created[0].Title != "already pushed (updated)" {
		t.Errorf("matched entry Title=%q, want the re-pushed value", created[0].Title)
	}

	all, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d items in vault, want 2 (one updated in place, one newly created); items=%+v", len(all), all)
	}
}

// ---------------------------------------------------------------------------
// migration v11 -> v12 (external_ref column + unique index)
// ---------------------------------------------------------------------------

// preV12TodosSchema is a frozen snapshot of the todos table (and its
// supporting tables) exactly as they existed at schema_version 11 — i.e.
// with no external_ref column and no todos_external_ref_idx index. It's
// deliberately a static literal rather than derived from the current
// schema.sql: the point of this test is to simulate a real user's on-disk
// v11 database file, which will never spontaneously pick up unrelated
// schema.sql edits made after this migration shipped.
const preV12TodosSchema = `
CREATE TABLE IF NOT EXISTS todos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  priority TEXT CHECK (length(priority) = 1) DEFAULT NULL,
  completed INTEGER NOT NULL DEFAULT 0,
  archived  INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  due_at     TEXT DEFAULT NULL,
  threshold_at TEXT DEFAULT NULL,
  recurrence_rule TEXT DEFAULT NULL,
  source_line TEXT,
  notes_md TEXT DEFAULT '',
  notes_doc TEXT DEFAULT '',
  notes_html TEXT DEFAULT '',
  notes_html_version INTEGER NOT NULL DEFAULT 0,
  section TEXT DEFAULT 'anytime',
  pinned INTEGER NOT NULL DEFAULT 0,
  kind TEXT NOT NULL DEFAULT 'todo',
  inbox INTEGER NOT NULL DEFAULT 0,
  api_source TEXT DEFAULT NULL
);
CREATE TRIGGER IF NOT EXISTS todos_update_ts
AFTER UPDATE ON todos FOR EACH ROW
BEGIN
  UPDATE todos SET updated_at = datetime('now') WHERE id = NEW.id;
END;
CREATE TABLE IF NOT EXISTS projects (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS contexts (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS tags (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS todo_projects (
  todo_id INTEGER, project_id INTEGER,
  PRIMARY KEY (todo_id, project_id),
  FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE,
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS todo_contexts (
  todo_id INTEGER, context_id INTEGER,
  PRIMARY KEY (todo_id, context_id),
  FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE,
  FOREIGN KEY (context_id) REFERENCES contexts(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS todo_tags (
  todo_id INTEGER, tag_id INTEGER,
  PRIMARY KEY (todo_id, tag_id),
  FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS refs (
  source_id INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
  target_id INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
  PRIMARY KEY (source_id, target_id)
);
CREATE TABLE IF NOT EXISTS kinds (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  icon TEXT NOT NULL DEFAULT '',
  default_view TEXT NOT NULL DEFAULT '',
  plugin_id TEXT DEFAULT NULL,
  is_core INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
INSERT OR IGNORE INTO kinds (name, display_name, icon, is_core) VALUES
  ('todo',    'Todo',    'check-square', 1),
  ('note',    'Note',    'file-text',    1),
  ('scratch', 'Scratch', 'edit',         1);
CREATE VIRTUAL TABLE IF NOT EXISTS todos_fts USING fts5(title, notes_text);
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER NOT NULL,
  applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`

// TestMigrationV11ToV12AddsExternalRef simulates a real user's on-disk
// database that's already at schema_version 11 (pre-external_ref) and
// confirms opening it with the current code: adds the external_ref column,
// creates the partial unique index, stamps schema_version 12, preserves
// pre-existing data untouched, and leaves the store fully functional
// afterward (a subsequent CreateItem with ExternalRef set works normally).
func TestMigrationV11ToV12AddsExternalRef(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "todo.db")

	// Build a raw v11-shaped database file directly (bypassing store.Open,
	// which would apply the CURRENT schema.sql — including external_ref —
	// to a fresh DB).
	raw, err := sql.Open("sqlite", dbPath+"?_fk=1")
	if err != nil {
		t.Fatalf("sql.Open (raw v11 db): %v", err)
	}
	if _, err = raw.Exec(preV12TodosSchema); err != nil {
		t.Fatalf("creating pre-v12 schema: %v", err)
	}
	if _, err = raw.Exec(`INSERT INTO todos (title, kind, section) VALUES ('pre-existing item', 'todo', 'anytime')`); err != nil {
		t.Fatalf("seeding pre-existing row: %v", err)
	}
	if _, err = raw.Exec(`INSERT INTO schema_version (version) VALUES (11)`); err != nil {
		t.Fatalf("stamping schema_version=11: %v", err)
	}
	var preCount int
	if err = raw.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='external_ref'`).Scan(&preCount); err != nil {
		t.Fatalf("checking pre-migration column: %v", err)
	}
	if preCount != 0 {
		t.Fatalf("test fixture bug: external_ref already present before migration")
	}
	if err = raw.Close(); err != nil {
		t.Fatalf("closing raw v11 db: %v", err)
	}

	// Now open it through the real code path — this is what happens on a
	// real user's next app launch after upgrading.
	ctx := context.Background()
	st, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open (migrating v11 -> v12): %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	var colCount int
	if err = st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('todos') WHERE name='external_ref'`).Scan(&colCount); err != nil {
		t.Fatalf("checking post-migration column: %v", err)
	}
	if colCount != 1 {
		t.Fatalf("external_ref column missing after migration")
	}

	var idxCount int
	if err = st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='todos_external_ref_idx'`).Scan(&idxCount); err != nil {
		t.Fatalf("checking post-migration index: %v", err)
	}
	if idxCount != 1 {
		t.Fatalf("todos_external_ref_idx missing after migration")
	}

	var version int
	if err = st.DB.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil {
		t.Fatalf("checking schema_version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("schema_version=%d after migration, want %d", version, currentSchemaVersion)
	}

	// Pre-existing data must survive the migration untouched.
	all, err := st.Search(ctx, SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search after migration: %v", err)
	}
	if len(all) != 1 || all[0].Title != "pre-existing item" {
		t.Fatalf("pre-existing row not preserved across migration: %+v", all)
	}

	// The store must be fully functional post-migration: a normal
	// external_ref create/dedup round-trip should work exactly as it does
	// on a fresh install.
	created, err := st.CreateItem(ctx, &Item{Title: "post-migration item", Kind: "todo", ExternalRef: "post-mig-1"})
	if err != nil {
		t.Fatalf("CreateItem with ExternalRef after migration: %v", err)
	}
	repushed, err := st.CreateItem(ctx, &Item{Title: "post-migration item v2", Kind: "todo", ExternalRef: "post-mig-1"})
	if err != nil {
		t.Fatalf("CreateItem re-push after migration: %v", err)
	}
	if repushed.ID != created.ID {
		t.Errorf("post-migration external_ref dedup didn't work: got new id=%d, want existing id=%d", repushed.ID, created.ID)
	}
}

// TestMigrationV11ToV12IsIdempotent confirms running the migration path
// twice (e.g. app restarts mid-upgrade, or a second CLI process opens the
// same vault) doesn't error and doesn't duplicate the index/column.
func TestMigrationV11ToV12IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	st1, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open (first): %v", err)
	}
	if err = st1.Close(); err != nil {
		t.Fatalf("Close (first): %v", err)
	}

	// Re-open the same on-disk vault — exercises runMigrations again
	// against a DB that's already fully migrated.
	st2, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open (second, idempotency check): %v", err)
	}
	t.Cleanup(func() { _ = st2.Close() })

	var idxCount int
	if err := st2.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='todos_external_ref_idx'`).Scan(&idxCount); err != nil {
		t.Fatalf("checking index count: %v", err)
	}
	if idxCount != 1 {
		t.Fatalf("todos_external_ref_idx count=%d after re-open, want exactly 1 (no duplicate)", idxCount)
	}
}

// TestMigrationV13CleanupOrphanedRefs covers the one-time data cleanup added
// for CW-20260816-0059: before the "_fk=1" -> "_pragma=foreign_keys(1)" DSN
// fix, foreign key enforcement was silently never active, so refs' ON
// DELETE CASCADE never fired and DeleteItem could leave orphaned refs rows
// (a real user vault inspected during the fix had 11 of 15 refs rows
// orphaned). This builds a raw pre-v13 database seeded with both a valid
// refs row and orphaned ones (source missing, target missing, and both
// missing), opens it through the real migration path, and confirms only the
// valid row survives while schema_version advances to the current version.
func TestMigrationV13CleanupOrphanedRefs(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "todo.db")

	// Build a raw v12-shaped database directly (bypassing store.Open, and
	// deliberately not enabling the foreign_keys pragma on this raw
	// connection) so orphaned refs rows can be inserted without SQLite
	// rejecting them.
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open (raw v12 db): %v", err)
	}
	if _, err = raw.Exec(schemaSQL); err != nil {
		t.Fatalf("creating v12 schema: %v", err)
	}
	// schema_version isn't part of schemaSQL — runMigrations creates it on
	// demand (see runMigrations' own "CREATE TABLE IF NOT EXISTS
	// schema_version" at the top of that function) — so this raw fixture
	// must create it itself before stamping a version into it.
	if _, err = raw.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("creating schema_version table: %v", err)
	}
	if _, err = raw.Exec(`INSERT INTO schema_version (version) VALUES (12)`); err != nil {
		t.Fatalf("stamping schema_version=12: %v", err)
	}
	if _, err = raw.Exec(`INSERT INTO todos (id, title, kind, section) VALUES
		(1, 'Alive source', 'note', 'anytime'),
		(2, 'Alive target', 'note', 'anytime')`); err != nil {
		t.Fatalf("seeding todos: %v", err)
	}
	if _, err = raw.Exec(`INSERT INTO refs (source_id, target_id) VALUES
		(1, 2),   -- valid: both endpoints exist
		(1, 999), -- orphaned: target missing
		(998, 2), -- orphaned: source missing
		(997, 996) -- orphaned: both missing
	`); err != nil {
		t.Fatalf("seeding refs (incl. orphans): %v", err)
	}
	var preCount int
	if err = raw.QueryRow(`SELECT COUNT(*) FROM refs`).Scan(&preCount); err != nil {
		t.Fatalf("checking pre-migration refs count: %v", err)
	}
	if preCount != 4 {
		t.Fatalf("test fixture bug: seeded refs count=%d, want 4", preCount)
	}
	if err = raw.Close(); err != nil {
		t.Fatalf("closing raw v12 db: %v", err)
	}

	// Now open it through the real code path, exercising migrateV13CleanupOrphanedRefs.
	ctx := context.Background()
	st, err := Open(ctx, dir)
	if err != nil {
		t.Fatalf("Open (migrating v12 -> v13): %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	var version int
	if err = st.DB.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil {
		t.Fatalf("checking schema_version: %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("schema_version=%d after migration, want %d", version, currentSchemaVersion)
	}

	rows, err := st.DB.QueryContext(ctx, `SELECT source_id, target_id FROM refs`)
	if err != nil {
		t.Fatalf("querying refs post-migration: %v", err)
	}
	defer rows.Close()
	type pair struct{ source, target int64 }
	var remaining []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.source, &p.target); err != nil {
			t.Fatalf("scanning refs row: %v", err)
		}
		remaining = append(remaining, p)
	}
	if len(remaining) != 1 || remaining[0] != (pair{1, 2}) {
		t.Fatalf("refs after cleanup = %+v, want exactly [{source:1 target:2}]", remaining)
	}
}
