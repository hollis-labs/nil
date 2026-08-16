package vault

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/store"
)

// newTestConfig builds a minimal, self-contained Config rooted under a fresh
// t.TempDir(), mirroring the fixture apiserver_test.go uses for its own
// vault.NewManager wiring. It also redirects HOME (and platform equivalents)
// to a scratch directory: several Manager methods (CreateVault, SwitchVault,
// RenameVault, DeleteVault) call config.Save/config.Load internally, which
// resolve the OS-appropriate config dir from the real environment — without
// this redirect, running these tests would read/write the developer's actual
// ~/.config/nil/config.json.
func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	root := t.TempDir()
	cfg := &config.Config{
		ActiveVaultID: "v1",
		InboxPath:     filepath.Join(root, "inbox"),
		Vaults: []config.Vault{
			{ID: "v1", Name: "Primary", Path: filepath.Join(root, "primary")},
		},
	}
	cfg.EnsureDefaults()
	return cfg
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	cfg := newTestConfig(t)
	mgr, err := NewManager(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(mgr.CloseAll)
	return mgr
}

func TestNewManagerOpensActiveVaultAndInbox(t *testing.T) {
	mgr := newTestManager(t)

	if mgr.GetActiveVaultID() != "v1" {
		t.Errorf("GetActiveVaultID() = %q, want %q", mgr.GetActiveVaultID(), "v1")
	}
	if mgr.ActiveStore() == nil {
		t.Fatal("ActiveStore() = nil, want the opened v1 store")
	}
	if mgr.InboxStore() == nil {
		t.Fatal("InboxStore() = nil, want the opened shared inbox store")
	}

	av := mgr.GetActiveVault()
	if av == nil {
		t.Fatal("GetActiveVault() = nil, want the v1 vault metadata")
	}
	if av.ID != "v1" || av.Name != "Primary" {
		t.Errorf("GetActiveVault() = %+v, want ID=v1 Name=Primary", av)
	}
}

func TestGetVaultsReturnsACopyNotTheRegistry(t *testing.T) {
	mgr := newTestManager(t)

	vaults := mgr.GetVaults()
	if len(vaults) != 1 {
		t.Fatalf("GetVaults() = %+v, want exactly one vault", vaults)
	}

	// Mutating the returned slice/struct must not affect the manager's
	// internal registry — GetVaults documents itself as returning a copy.
	vaults[0].Name = "mutated"

	again := mgr.GetVaults()
	if again[0].Name != "Primary" {
		t.Errorf("GetVaults() Name = %q after external mutation, want unaffected %q (GetVaults should return a defensive copy)", again[0].Name, "Primary")
	}
}

func TestCreateVaultRegistersAndOpensStore(t *testing.T) {
	mgr := newTestManager(t)
	root := t.TempDir()
	newDir := filepath.Join(root, "second-vault")

	v, err := mgr.CreateVault("Second", newDir)
	if err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if v.ID == "" {
		t.Fatal("CreateVault returned a vault with empty ID")
	}
	if v.Name != "Second" {
		t.Errorf("CreateVault Name = %q, want %q", v.Name, "Second")
	}
	if v.Path != newDir {
		t.Errorf("CreateVault Path = %q, want %q", v.Path, newDir)
	}

	vaults := mgr.GetVaults()
	if len(vaults) != 2 {
		t.Fatalf("GetVaults() after CreateVault = %+v, want 2 vaults", vaults)
	}

	// The new vault's store should already be open (no lazy-open needed) —
	// StoreForID must return the same store without erroring.
	s, err := mgr.StoreForID(v.ID)
	if err != nil {
		t.Fatalf("StoreForID(%q): %v", v.ID, err)
	}
	if s == nil {
		t.Fatal("StoreForID returned nil store for freshly created vault")
	}

	// Persisted to config.json on disk via config.Save — reload and confirm.
	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	found := false
	for _, rv := range reloaded.Vaults {
		if rv.ID == v.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("reloaded config.Vaults = %+v, want it to include newly created vault %q", reloaded.Vaults, v.ID)
	}
}

func TestSwitchVaultChangesActiveAndPersists(t *testing.T) {
	mgr := newTestManager(t)
	root := t.TempDir()

	v2, err := mgr.CreateVault("Second", filepath.Join(root, "second-vault"))
	if err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	if err = mgr.SwitchVault(v2.ID); err != nil {
		t.Fatalf("SwitchVault: %v", err)
	}

	if mgr.GetActiveVaultID() != v2.ID {
		t.Errorf("GetActiveVaultID() = %q after SwitchVault, want %q", mgr.GetActiveVaultID(), v2.ID)
	}
	if mgr.ActiveStore() == nil {
		t.Fatal("ActiveStore() = nil after SwitchVault")
	}
	active := mgr.GetActiveVault()
	if active == nil || active.ID != v2.ID {
		t.Errorf("GetActiveVault() = %+v, want ID=%q", active, v2.ID)
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if reloaded.ActiveVaultID != v2.ID {
		t.Errorf("persisted ActiveVaultID = %q, want %q", reloaded.ActiveVaultID, v2.ID)
	}
}

func TestSwitchVaultToUnknownIDFails(t *testing.T) {
	mgr := newTestManager(t)

	if err := mgr.SwitchVault("does-not-exist"); err == nil {
		t.Fatal("SwitchVault(unknown id) = nil error, want an error")
	}
	// Active vault must remain unchanged after a failed switch.
	if mgr.GetActiveVaultID() != "v1" {
		t.Errorf("GetActiveVaultID() = %q after failed SwitchVault, want unchanged %q", mgr.GetActiveVaultID(), "v1")
	}
}

func TestStoreForIDLazilyOpensRegisteredButUnopenedVault(t *testing.T) {
	// Build a config with two registered vaults but only construct the
	// Manager against the active one — NewManager only opens ActiveVaultID
	// up front, so v2's store must be lazily opened on first StoreForID call.
	root := t.TempDir()
	cfg := &config.Config{
		ActiveVaultID: "v1",
		InboxPath:     filepath.Join(root, "inbox"),
		Vaults: []config.Vault{
			{ID: "v1", Name: "Primary", Path: filepath.Join(root, "primary")},
			{ID: "v2", Name: "Secondary", Path: filepath.Join(root, "secondary")},
		},
	}
	cfg.EnsureDefaults()

	mgr, err := NewManager(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(mgr.CloseAll)

	s, err := mgr.StoreForID("v2")
	if err != nil {
		t.Fatalf("StoreForID(v2): %v", err)
	}
	if s == nil {
		t.Fatal("StoreForID(v2) = nil store")
	}

	// Same store instance returned on a second call (cached, not reopened).
	s2, err := mgr.StoreForID("v2")
	if err != nil {
		t.Fatalf("StoreForID(v2) second call: %v", err)
	}
	if s != s2 {
		t.Error("StoreForID(v2) returned a different instance on second call, want the cached store")
	}
}

func TestStoreForIDUnknownVaultFails(t *testing.T) {
	mgr := newTestManager(t)

	if _, err := mgr.StoreForID("nope"); err == nil {
		t.Fatal("StoreForID(unknown) = nil error, want an error")
	}
}

func TestRenameVaultUpdatesRegistryAndPersists(t *testing.T) {
	mgr := newTestManager(t)

	if err := mgr.RenameVault("v1", "Renamed"); err != nil {
		t.Fatalf("RenameVault: %v", err)
	}

	vaults := mgr.GetVaults()
	if len(vaults) != 1 || vaults[0].Name != "Renamed" {
		t.Errorf("GetVaults() = %+v, want vault v1 renamed to %q", vaults, "Renamed")
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(reloaded.Vaults) != 1 || reloaded.Vaults[0].Name != "Renamed" {
		t.Errorf("persisted Vaults = %+v, want renamed vault", reloaded.Vaults)
	}
}

func TestRenameVaultUnknownIDFails(t *testing.T) {
	mgr := newTestManager(t)

	if err := mgr.RenameVault("nope", "New Name"); err == nil {
		t.Fatal("RenameVault(unknown id) = nil error, want an error")
	}
}

func TestDeleteVaultRefusesActiveVault(t *testing.T) {
	mgr := newTestManager(t)
	root := t.TempDir()

	if _, err := mgr.CreateVault("Second", filepath.Join(root, "second-vault")); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	if err := mgr.DeleteVault("v1"); err == nil {
		t.Fatal("DeleteVault(active vault) = nil error, want a refusal error")
	}
	if len(mgr.GetVaults()) != 2 {
		t.Errorf("GetVaults() after refused delete = %+v, want unchanged 2 vaults", mgr.GetVaults())
	}
}

func TestDeleteVaultRefusesLastRemainingVault(t *testing.T) {
	mgr := newTestManager(t)

	// v1 is both the only vault and the active vault; either guard could
	// legitimately fire first, but a delete of the sole vault must fail.
	if err := mgr.DeleteVault("v1"); err == nil {
		t.Fatal("DeleteVault(only vault) = nil error, want a refusal error")
	}
}

func TestDeleteVaultRemovesNonActiveVault(t *testing.T) {
	mgr := newTestManager(t)
	root := t.TempDir()

	v2, err := mgr.CreateVault("Second", filepath.Join(root, "second-vault"))
	if err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	if err = mgr.DeleteVault(v2.ID); err != nil {
		t.Fatalf("DeleteVault(%q): %v", v2.ID, err)
	}

	vaults := mgr.GetVaults()
	if len(vaults) != 1 || vaults[0].ID != "v1" {
		t.Errorf("GetVaults() after DeleteVault = %+v, want only v1 remaining", vaults)
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(reloaded.Vaults) != 1 {
		t.Errorf("persisted Vaults after delete = %+v, want 1 vault", reloaded.Vaults)
	}

	// StoreForID must now fail — the store was closed and deregistered.
	if _, err := mgr.StoreForID(v2.ID); err == nil {
		t.Error("StoreForID(deleted vault) = nil error, want an error since it was removed from the registry")
	}
}

func TestDeleteVaultUnknownIDFails(t *testing.T) {
	mgr := newTestManager(t)
	root := t.TempDir()

	if _, err := mgr.CreateVault("Second", filepath.Join(root, "second-vault")); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	if err := mgr.DeleteVault("does-not-exist"); err == nil {
		t.Fatal("DeleteVault(unknown id) = nil error, want an error")
	}
}

func TestMoveItemToVaultMovesFromInboxToTarget(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	inbox := mgr.InboxStore()
	created, err := inbox.CreateItem(ctx, &store.Item{Title: "captured idea", Inbox: true})
	if err != nil {
		t.Fatalf("CreateItem in inbox: %v", err)
	}

	if err = mgr.MoveItemToVault(ctx, created.ID, "v1"); err != nil {
		t.Fatalf("MoveItemToVault: %v", err)
	}

	// Gone from the inbox.
	if _, err = inbox.GetItem(ctx, created.ID); err == nil {
		t.Error("GetItem on inbox after move = nil error, want the item to be deleted from the inbox")
	}

	// Present in the target vault, with Inbox cleared and a default section.
	target := mgr.ActiveStore()
	items, err := target.Search(ctx, store.SearchRequest{Kind: "all"})
	if err != nil {
		t.Fatalf("Search target vault: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("target vault items = %+v, want exactly 1 moved item", items)
	}
	moved := items[0]
	if moved.Title != "captured idea" {
		t.Errorf("moved item Title = %q, want %q", moved.Title, "captured idea")
	}
	if moved.Inbox {
		t.Error("moved item Inbox = true, want false after moving out of the inbox")
	}
	if moved.Section != "anytime" {
		t.Errorf("moved item Section = %q, want default %q", moved.Section, "anytime")
	}
}

func TestMoveItemToVaultUnknownItemFails(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if err := mgr.MoveItemToVault(ctx, 999999, "v1"); err == nil {
		t.Fatal("MoveItemToVault(unknown item id) = nil error, want an error")
	}
}

func TestMoveItemToVaultUnknownTargetVaultFails(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	inbox := mgr.InboxStore()
	created, err := inbox.CreateItem(ctx, &store.Item{Title: "orphaned idea", Inbox: true})
	if err != nil {
		t.Fatalf("CreateItem in inbox: %v", err)
	}

	if err := mgr.MoveItemToVault(ctx, created.ID, "does-not-exist"); err == nil {
		t.Fatal("MoveItemToVault(unknown target vault) = nil error, want an error")
	}

	// Item must still be present in the inbox since the move failed before
	// the target-vault CreateItem/inbox-delete pair completed.
	if _, err := inbox.GetItem(ctx, created.ID); err != nil {
		t.Errorf("GetItem on inbox after failed move: %v, want item still present", err)
	}
}

func TestCloseAllClosesEveryOpenStore(t *testing.T) {
	mgr := newTestManager(t)
	root := t.TempDir()

	v2, err := mgr.CreateVault("Second", filepath.Join(root, "second-vault"))
	if err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	mgr.CloseAll()

	// Closed *store.Store should error on further use. Use GetItem as a
	// cheap probe against both the active and secondary store, plus inbox.
	ctx := context.Background()
	if _, err := mgr.ActiveStore().GetItem(ctx, 1); err == nil {
		t.Error("ActiveStore().GetItem after CloseAll = nil error, want an error on a closed DB handle")
	}
	if s, ok := storeForIDNoOpen(mgr, v2.ID); ok {
		if _, err := s.GetItem(ctx, 1); err == nil {
			t.Error("secondary store GetItem after CloseAll = nil error, want an error on a closed DB handle")
		}
	}
	if _, err := mgr.InboxStore().GetItem(ctx, 1); err == nil {
		t.Error("InboxStore().GetItem after CloseAll = nil error, want an error on a closed DB handle")
	}
}

// storeForIDNoOpen reads the manager's already-open store for id directly,
// without going through StoreForID's lazy-open path (which would reopen a
// fresh, non-closed connection and defeat the CloseAll assertion above).
func storeForIDNoOpen(mgr *Manager, id string) (*store.Store, bool) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	s, ok := mgr.stores[id]
	return s, ok
}
