package capture

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDraftRestartAndProjection(t *testing.T) {
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
	source := "---\nunknown: true\n---\n\n# Hello 世界\n\n<script>alert('x')</script>\n[bad](javascript:alert(1))\n\n| unsupported | table |\n"
	saved, err := s.Save(ctx, d.CaptureID, d.Revision, source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, d.CaptureID, d.Revision, "stale"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale update: %v", err)
	}
	if _, err = s.db.Exec(`UPDATE capture_drafts SET html='<script>stale</script>',renderer_version='old',content_hash='old'`); err != nil {
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
	if got.Markdown != source || got.ContentHash != hash(source) || got.Revision != saved.Revision {
		t.Fatalf("source changed across restart: %+v", got)
	}
	if strings.Contains(got.HTML, "<script>") || got.RendererVersion != RendererVersion {
		t.Fatalf("unsafe/stale preview: %s", got.HTML)
	}
	drafts, err := s.List(ctx)
	if err != nil || len(drafts) != 1 {
		t.Fatalf("recovery: %v %v", drafts, err)
	}
}

func TestRejectInvalidSource(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{string([]byte{0xff}), strings.Repeat("x", MaxMarkdownBytes+1)} {
		if _, err = s.Save(ctx, d.CaptureID, d.Revision, source); err == nil {
			t.Fatal("accepted invalid source")
		}
	}
}
