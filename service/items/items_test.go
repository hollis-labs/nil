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
