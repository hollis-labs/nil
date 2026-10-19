package capture

import (
	"context"
	"testing"
)

func TestInputAndReferencesSurviveRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	suggestion := ItemSuggestion("vault/one", 42, "revision1", "Plan")
	body := "---\nunknown: untouched\n---\n\n@body +notmetadata #notatag\n\n[Plan](" + suggestion.URL + ")\n\n`[Code](" + suggestion.URL + ")`\n"
	d, err = s.SaveContent(ctx, d.CaptureID, d.Revision, body, "(B) Follow up +work #next @desk due:2026-10-20 t:2026-10-19")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Get(ctx, d.CaptureID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Markdown != body || got.QuickAdd != d.QuickAdd || got.Fields.Title != "Follow up" || len(got.Fields.Projects) != 1 || got.Fields.Projects[0] != "work" || got.Fields.Contexts[0] != "desk" {
		t.Fatalf("source/default drift: %+v", got)
	}
	if got.Fields.Priority == nil || *got.Fields.Priority != "B" || got.Fields.DueDate == nil || *got.Fields.DueDate != "2026-10-20" {
		t.Fatalf("quick fields: %+v", got.Fields)
	}
	if len(got.References) != 1 || got.References[0].Instance != "vault/one" || got.References[0].ID != "42" || got.References[0].Revision != "revision1" {
		t.Fatalf("identity: %+v", got.References)
	}
	edited, err := s.SaveContent(ctx, got.CaptureID, got.Revision, "@desk [web](https://example.test)\n```md\n[Plan]("+suggestion.URL+")\n```", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(edited.References) != 0 || len(edited.Fields.Contexts) != 0 {
		t.Fatalf("stale refs/defaults: %+v", edited)
	}
}

func TestInputTableAddedWithoutChangingSourceRows(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DROP TABLE capture_inputs`); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Get(ctx, d.CaptureID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CaptureID != d.CaptureID || got.Revision != d.Revision || got.ContentHash != d.ContentHash {
		t.Fatal("source row changed")
	}
}
