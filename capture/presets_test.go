package capture

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestPresetPrecedenceSnapshotAndImmutableQueueAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.SavePreset(ctx, "", 0, "Work", "(B) +work #keep #removed @desk")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SelectPreset(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	d, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d.Snapshot.Preset == nil || d.Snapshot.Preset.Revision != 1 || d.Snapshot.AppliedBy != "manual" || d.QuickAdd != p.QuickAdd || d.Snapshot.Context.State != "unavailable" {
		t.Fatalf("initial snapshot: %+v", d)
	}
	d, err = s.SaveContent(ctx, d.CaptureID, d.Revision, "Original **Markdown**", "(A) Explicit title +mine #keep @home")
	if err != nil {
		t.Fatal(err)
	}
	env, err := s.Queue(ctx, d.CaptureID, d.Revision)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(env)
	if _, err = s.SaveContent(ctx, d.CaptureID, d.Revision, "changed", "#removed"); !errors.Is(err, ErrConflict) {
		t.Fatalf("queued draft editable: %v", err)
	}
	if _, err = s.SavePreset(ctx, p.ID, p.Revision, "New name", "(C) +other #removed @elsewhere"); err != nil {
		t.Fatal(err)
	}
	next, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if next.Snapshot.Preset.Revision != 2 || next.Fields.Projects[0] != "other" {
		t.Fatalf("new draft ignored update: %+v", next)
	}
	if err = s.SelectPreset(ctx, ""); err != nil {
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
	replay, err := s.Queue(ctx, d.CaptureID, d.Revision)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(replay)
	if string(before) != string(after) {
		t.Fatal("immutable envelope changed after preset switch/restart")
	}
	got, err := s.Get(ctx, d.CaptureID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LocalQueued || got.Snapshot.Preset.Name != "Work" || got.Snapshot.Preset.Revision != 1 || got.Snapshot.Context != d.Snapshot.Context || len(got.Fields.Tags) != 1 || got.Fields.Tags[0] != "keep" || got.Fields.Contexts[0] != "home" || *got.Fields.Priority != "A" {
		t.Fatalf("explicit removal/default drift: %+v", got)
	}
	blank, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if blank.Snapshot.Preset != nil || blank.QuickAdd != "" {
		t.Fatal("cleared selection restored old defaults")
	}
}

func TestPresetCASAndLegacyDraftIsolation(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	legacy, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DELETE FROM capture_snapshots WHERE capture_id=?`, legacy.CaptureID); err != nil {
		t.Fatal(err)
	}
	p, err := s.SavePreset(ctx, "", 0, "Later", "#later")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SelectPreset(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SavePreset(ctx, p.ID, 0, "stale", "#wrong"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale preset update: %v", err)
	}
	if err = s.SelectPreset(ctx, "missing"); err == nil {
		t.Fatal("accepted nonexistent preset")
	}
	got, err := s.Get(ctx, legacy.CaptureID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Snapshot.Preset != nil || got.QuickAdd != "" || len(got.Fields.Tags) != 0 {
		t.Fatal("later defaults applied retroactively")
	}
	selection, err := s.Presets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if selection.SelectedID != p.ID || len(selection.Presets) != 1 {
		t.Fatal("failed selection changed durable choice")
	}
}
