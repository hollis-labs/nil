package capture

import (
	"context"
	"testing"

	"github.com/hollis-labs/nil/capture/entry"
)

func TestEntryPreferenceDefaultsAndRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	shortcut, err := s.EntryShortcut(ctx)
	if err != nil || shortcut != entry.Default() || shortcut.Enabled {
		t.Fatal("fresh store enabled global shortcut", err)
	}
	shortcut.Enabled = true
	shortcut.Key = "K"
	if err = s.SaveEntryShortcut(ctx, shortcut); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	s, err = Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	restored, err := s.EntryShortcut(ctx)
	if err != nil || restored != shortcut {
		t.Fatal("shortcut preference lost on restart", err)
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE capture_preferences SET value='invalid' WHERE key='entry_shortcut'`); err != nil {
		t.Fatal(err)
	}
	restored, err = s.EntryShortcut(ctx)
	if err == nil || restored.Enabled {
		t.Fatal("invalid preference enabled shortcut")
	}
}
