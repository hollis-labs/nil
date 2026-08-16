package items

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hollis-labs/nil/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("opening test store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestResolveNotesInputPrecedence(t *testing.T) {
	svc := New()

	tests := []struct {
		name      string
		doc       string
		md        string
		html      string
		wantEmpty bool
		wantInDoc string // substring expected in returned doc JSON
	}{
		{name: "all empty", wantEmpty: true},
		{name: "doc wins over md and html", doc: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]}`, md: "ignored", html: "<p>ignored</p>", wantInDoc: `"hello"`},
		{name: "md wins over html", md: "# Hello", html: "<p>ignored</p>", wantInDoc: "Hello"},
		{name: "html alone", html: "<p>only html</p>", wantInDoc: "only html"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, html, version, err := svc.ResolveNotesInput(tc.doc, tc.md, tc.html)
			if err != nil {
				t.Fatalf("ResolveNotesInput: %v", err)
			}
			if tc.wantEmpty {
				if doc != "" || html != "" || version != 0 {
					t.Errorf("expected all empty, got doc=%q html=%q version=%d", doc, html, version)
				}
				return
			}
			if !strings.Contains(doc, tc.wantInDoc) {
				t.Errorf("doc=%q does not contain %q", doc, tc.wantInDoc)
			}
			if html == "" {
				t.Errorf("expected non-empty html cache")
			}
			if version != NotesHTMLVersion {
				t.Errorf("version=%d, want %d", version, NotesHTMLVersion)
			}
		})
	}
}

func TestNotesInputFromCLI(t *testing.T) {
	svc := New()

	t.Run("empty body returns all empty", func(t *testing.T) {
		doc, md, html, err := svc.NotesInputFromCLI("", "md")
		if err != nil || doc != "" || md != "" || html != "" {
			t.Errorf("got (%q, %q, %q, %v); want all zero", doc, md, html, err)
		}
	})

	t.Run("default format routes to md", func(t *testing.T) {
		doc, md, html, err := svc.NotesInputFromCLI("# Title", "")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if md != "# Title" || doc != "" || html != "" {
			t.Errorf("got doc=%q md=%q html=%q", doc, md, html)
		}
	})

	t.Run("html format routes to html field", func(t *testing.T) {
		doc, md, html, err := svc.NotesInputFromCLI("<p>x</p>", "html")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if html != "<p>x</p>" || doc != "" || md != "" {
			t.Errorf("got doc=%q md=%q html=%q", doc, md, html)
		}
	})

	t.Run("json format routes to doc field", func(t *testing.T) {
		doc, md, html, err := svc.NotesInputFromCLI(`{"type":"doc"}`, "json")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if doc != `{"type":"doc"}` || md != "" || html != "" {
			t.Errorf("got doc=%q md=%q html=%q", doc, md, html)
		}
	})

	t.Run("unknown format errors", func(t *testing.T) {
		_, _, _, err := svc.NotesInputFromCLI("body", "rst")
		if !errors.Is(err, ErrInvalidNotesFormat) {
			t.Errorf("err=%v, want ErrInvalidNotesFormat", err)
		}
	})
}

func TestCreateDefaultsKindAndSection(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	item, err := svc.Create(ctx, st, CreateInput{Title: "minimal"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if item.Kind != "todo" {
		t.Errorf("Kind=%q, want todo", item.Kind)
	}
	if item.Section != "anytime" {
		t.Errorf("Section=%q, want anytime", item.Section)
	}
	if item.NotesDoc != "" || item.NotesHTML != "" {
		t.Errorf("empty notes input should yield empty stored notes")
	}
}

func TestCreateWithMarkdownNotes(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	item, err := svc.Create(ctx, st, CreateInput{
		Title:   "with body",
		Kind:    "note",
		NotesMD: "# Heading\n\nParagraph.",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.Contains(item.NotesDoc, "Heading") {
		t.Errorf("NotesDoc missing Heading: %q", item.NotesDoc)
	}
	if !strings.Contains(item.NotesHTML, "<h1>") {
		t.Errorf("NotesHTML missing <h1>: %q", item.NotesHTML)
	}
	if item.NotesHTMLVersion != NotesHTMLVersion {
		t.Errorf("NotesHTMLVersion=%d, want %d", item.NotesHTMLVersion, NotesHTMLVersion)
	}
}

func TestUpdatePatchTitleOnly(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	created, err := svc.Create(ctx, st, CreateInput{Title: "before", NotesMD: "body"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	newTitle := "after"
	updated, err := svc.UpdatePatch(ctx, st, created.ID, UpdatePatch{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdatePatch: %v", err)
	}
	if updated.Title != "after" {
		t.Errorf("Title=%q, want after", updated.Title)
	}
	if updated.NotesDoc != created.NotesDoc {
		t.Errorf("notes unexpectedly changed: before=%q after=%q", created.NotesDoc, updated.NotesDoc)
	}
}

func TestUpdatePatchClearsNullableWithExplicitNil(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	p := "A"
	created, err := svc.Create(ctx, st, CreateInput{Title: "priority test", Priority: &p})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Priority == nil || *created.Priority != "A" {
		t.Fatalf("Priority not stored: %v", created.Priority)
	}

	// Clear: Set=true, Value=nil
	updated, err := svc.UpdatePatch(ctx, st, created.ID, UpdatePatch{
		Priority: NullableString{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatalf("UpdatePatch: %v", err)
	}
	if updated.Priority != nil {
		t.Errorf("Priority not cleared: %v", *updated.Priority)
	}
}

func TestUpdatePatchLeavesNullableUntouchedWhenUnset(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	p := "B"
	created, err := svc.Create(ctx, st, CreateInput{Title: "untouched", Priority: &p})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	newTitle := "renamed"
	updated, err := svc.UpdatePatch(ctx, st, created.ID, UpdatePatch{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdatePatch: %v", err)
	}
	if updated.Priority == nil || *updated.Priority != "B" {
		t.Errorf("Priority unexpectedly cleared/changed: %v", updated.Priority)
	}
}

func TestUpdatePatchReResolvesNotes(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	created, err := svc.Create(ctx, st, CreateInput{Title: "notes", NotesMD: "first"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	newMD := "# new heading"
	updated, err := svc.UpdatePatch(ctx, st, created.ID, UpdatePatch{NotesMD: &newMD})
	if err != nil {
		t.Fatalf("UpdatePatch: %v", err)
	}
	if !strings.Contains(updated.NotesHTML, "<h1>") {
		t.Errorf("NotesHTML should reflect new markdown: %q", updated.NotesHTML)
	}
}

func TestSearchAppliesKindAllDefault(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	if _, err := svc.Create(ctx, st, CreateInput{Title: "a todo", Kind: "todo"}); err != nil {
		t.Fatalf("Create todo: %v", err)
	}
	if _, err := svc.Create(ctx, st, CreateInput{Title: "a note", Kind: "note"}); err != nil {
		t.Fatalf("Create note: %v", err)
	}

	// Empty kind → service defaults to "all" → both rows returned.
	items, err := svc.Search(ctx, st, store.SearchRequest{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2 (kind=all default)", len(items))
	}
}

// TestSearchDefaultBehaviorUnchangedByUpdatedSince confirms the new
// UpdatedSince field is additive: omitting it (the existing/default call
// shape) returns everything regardless of update time, exactly like before
// this field existed.
func TestSearchDefaultBehaviorUnchangedByUpdatedSince(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	if _, err := svc.Create(ctx, st, CreateInput{Title: "a todo", Kind: "todo"}); err != nil {
		t.Fatalf("Create todo: %v", err)
	}
	if _, err := svc.Create(ctx, st, CreateInput{Title: "a note", Kind: "note"}); err != nil {
		t.Fatalf("Create note: %v", err)
	}

	items, err := svc.Search(ctx, st, store.SearchRequest{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2 (no updated_since → no time filter)", len(items))
	}
}

// TestSearchUpdatedSinceFutureExcludesEverything confirms the service passes
// UpdatedSince through to the store unmodified: a threshold in the future
// (after every row's updated_at) should exclude all rows created just now.
func TestSearchUpdatedSinceFutureExcludesEverything(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	if _, err := svc.Create(ctx, st, CreateInput{Title: "a todo", Kind: "todo"}); err != nil {
		t.Fatalf("Create todo: %v", err)
	}

	items, err := svc.Search(ctx, st, store.SearchRequest{UpdatedSince: "2099-01-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0 (updated_since in the future excludes everything)", len(items))
	}
}

// TestPlainTextRendersDocBody confirms PlainText reuses ingest.DocToPlainText
// to produce readable text from a stored notes_doc, and that the text
// reflects real body content (not just present-but-empty).
func TestPlainTextRendersDocBody(t *testing.T) {
	svc := New()
	doc := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hello world"}]}]}`
	got := svc.PlainText(doc)
	if !strings.Contains(got, "hello world") {
		t.Errorf("PlainText(%q) = %q, want it to contain %q", doc, got, "hello world")
	}
}

// TestPlainTextEmptyDocIsEmptyNotError confirms an empty notes_doc (the
// common case for inbox-capture items with no body) renders to "" rather
// than surfacing an error to the caller.
func TestPlainTextEmptyDocIsEmptyNotError(t *testing.T) {
	svc := New()
	if got := svc.PlainText(""); got != "" {
		t.Errorf("PlainText(\"\") = %q, want \"\"", got)
	}
}

// TestPlainTextMalformedDocIsEmptyNotError confirms a corrupt/malformed
// notes_doc is swallowed to "" instead of erroring, so one bad item can't
// break an entire list/search response for every other item alongside it.
func TestPlainTextMalformedDocIsEmptyNotError(t *testing.T) {
	svc := New()
	if got := svc.PlainText("{not valid json"); got != "" {
		t.Errorf("PlainText(malformed) = %q, want \"\"", got)
	}
}

// TestWithTextIsAdditive confirms WithText preserves every existing
// store.Item field unchanged (via embedding) while adding notes_text
// alongside them — purely additive, no existing-field behavior change.
func TestWithTextIsAdditive(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	created, err := svc.Create(ctx, st, CreateInput{
		Title:   "with body",
		NotesMD: "hello from markdown",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	view := svc.WithText(created)

	if view.ID != created.ID || view.Title != created.Title || view.NotesDoc != created.NotesDoc || view.NotesHTML != created.NotesHTML {
		t.Errorf("WithText did not preserve existing fields: got %+v, from %+v", view.Item, created)
	}
	if !strings.Contains(view.NotesText, "hello from markdown") {
		t.Errorf("NotesText=%q, want it to contain rendered markdown body", view.NotesText)
	}
}

// TestWithTextSliceHandlesEmptyBodyItem confirms an item with no body at all
// (e.g. a blank inbox capture) round-trips through WithTextSlice with an
// empty notes_text instead of breaking the whole list response.
func TestWithTextSliceHandlesEmptyBodyItem(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	withBody, err := svc.Create(ctx, st, CreateInput{Title: "has body", NotesMD: "some content here"})
	if err != nil {
		t.Fatalf("Create withBody: %v", err)
	}
	noBody, err := svc.Create(ctx, st, CreateInput{Title: "", Inbox: true})
	if err != nil {
		t.Fatalf("Create noBody: %v", err)
	}

	views := svc.WithTextSlice([]store.Item{*withBody, *noBody})
	if len(views) != 2 {
		t.Fatalf("got %d views, want 2", len(views))
	}
	if !strings.Contains(views[0].NotesText, "some content here") {
		t.Errorf("views[0].NotesText=%q, want it to contain body content", views[0].NotesText)
	}
	if views[1].NotesText != "" {
		t.Errorf("views[1].NotesText=%q, want \"\" for a bodyless item", views[1].NotesText)
	}
	// Existing fields still present and correct alongside the new field.
	if views[1].Title != "" || !views[1].Inbox {
		t.Errorf("views[1] existing fields not preserved: %+v", views[1].Item)
	}
}

// TestCreateBatchAppliesSameDefaultsAsCreate confirms CreateBatch's
// per-item shaping matches Create's exactly: kind/section defaulting and
// notes-input resolution, for every item in one call.
func TestCreateBatchAppliesSameDefaultsAsCreate(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	created, err := svc.CreateBatch(ctx, st, []CreateInput{
		{Title: "minimal"},
		{Title: "with markdown", NotesMD: "# Heading"},
		{Title: "explicit kind", Kind: "note", Section: "now"},
	})
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("got %d created items, want 3", len(created))
	}
	if created[0].Kind != "todo" || created[0].Section != "anytime" {
		t.Errorf("created[0] defaults not applied: kind=%q section=%q", created[0].Kind, created[0].Section)
	}
	if !strings.Contains(created[1].NotesDoc, "Heading") {
		t.Errorf("created[1].NotesDoc missing resolved markdown: %q", created[1].NotesDoc)
	}
	if created[2].Kind != "note" || created[2].Section != "now" {
		t.Errorf("created[2] explicit kind/section not honored: kind=%q section=%q", created[2].Kind, created[2].Section)
	}
}

// TestCreateBatchExternalRefRepushSharesCreateSemantics confirms
// CreateBatch's external_ref matching produces identical results to a
// single Create call re-push, when called through the service layer (not
// just the store layer directly).
func TestCreateBatchExternalRefRepushSharesCreateSemantics(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	first, err := svc.Create(ctx, st, CreateInput{Title: "seed", ExternalRef: "svc-batch-1"})
	if err != nil {
		t.Fatalf("Create (seed): %v", err)
	}

	created, err := svc.CreateBatch(ctx, st, []CreateInput{
		{Title: "seed (re-pushed)", ExternalRef: "svc-batch-1"},
		{Title: "genuinely new"},
	})
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if created[0].ID != first.ID {
		t.Errorf("CreateBatch re-push got new id=%d, want existing id=%d", created[0].ID, first.ID)
	}
	if created[0].Title != "seed (re-pushed)" {
		t.Errorf("CreateBatch re-push title=%q, want the new value", created[0].Title)
	}

	all, err := svc.Search(ctx, st, store.SearchRequest{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d items, want 2 (one updated in place, one new)", len(all))
	}
}

// TestCreateBatchAllOrNothingOnBadInput confirms a batch containing an
// unresolvable notes input (malformed notes_doc JSON) fails the whole call
// before any item is created — the service-layer half of the all-or-nothing
// contract (input-shaping failures are caught before store.CreateItemsBatch
// ever opens a transaction).
func TestCreateBatchAllOrNothingOnBadInput(t *testing.T) {
	st := openTestStore(t)
	svc := New()
	ctx := context.Background()

	_, err := svc.CreateBatch(ctx, st, []CreateInput{
		{Title: "valid"},
		{Title: "bad doc", NotesDoc: "{not valid json"},
	})
	if err == nil {
		t.Fatal("CreateBatch: expected an error for the malformed notes_doc item")
	}
	if !strings.Contains(err.Error(), "item 1") {
		t.Errorf("err=%q, want it to name the failing item's index (item 1)", err.Error())
	}

	all, err := svc.Search(ctx, st, store.SearchRequest{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("got %d items after a rejected batch, want 0 (nothing committed)", len(all))
	}
}
