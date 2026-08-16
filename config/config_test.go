package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// withConfigHome points HOME (and XDG_CONFIG_HOME on linux) at a temp
// directory for the duration of the test, so getConfigDir()/getConfigPath()
// resolve inside a throwaway location instead of the real user config dir.
// runtime.GOOS is fixed at compile time, so on darwin only HOME matters; on
// linux XDG_CONFIG_HOME is also cleared/pointed so a developer's real
// ~/.config/nil is never touched by the test suite regardless of platform.
func withConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("APPDATA", filepath.Join(dir, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	return dir
}

func TestEnsureDefaultsSetsPortAndKeyWhenZero(t *testing.T) {
	cfg := &Config{}

	changed := cfg.EnsureDefaults()

	if !changed {
		t.Fatal("EnsureDefaults() = false, want true for a zero-value Config")
	}
	if cfg.APIPort != 7765 {
		t.Errorf("APIPort = %d, want 7765", cfg.APIPort)
	}
	if cfg.APIKey == "" {
		t.Error("APIKey is empty, want a generated hex key")
	}
	if len(cfg.APIKey) != 64 { // 32 bytes hex-encoded
		t.Errorf("APIKey length = %d, want 64 (32 bytes hex-encoded)", len(cfg.APIKey))
	}
}

func TestEnsureDefaultsLeavesExistingValuesAlone(t *testing.T) {
	cfg := &Config{APIPort: 9999, APIKey: "existing-key"}

	changed := cfg.EnsureDefaults()

	if changed {
		t.Fatal("EnsureDefaults() = true, want false when port and key are already set")
	}
	if cfg.APIPort != 9999 {
		t.Errorf("APIPort = %d, want unchanged 9999", cfg.APIPort)
	}
	if cfg.APIKey != "existing-key" {
		t.Errorf("APIKey = %q, want unchanged %q", cfg.APIKey, "existing-key")
	}
}

func TestEnsureDefaultsIsIdempotentOnSecondCall(t *testing.T) {
	cfg := &Config{}
	cfg.EnsureDefaults()
	port, key := cfg.APIPort, cfg.APIKey

	changed := cfg.EnsureDefaults()

	if changed {
		t.Fatal("second EnsureDefaults() = true, want false once defaults are already populated")
	}
	if cfg.APIPort != port || cfg.APIKey != key {
		t.Errorf("second EnsureDefaults() mutated values: port %d->%d, key %q->%q", port, cfg.APIPort, key, cfg.APIKey)
	}
}

func TestEnsureDefaultsGeneratesDistinctKeys(t *testing.T) {
	a := &Config{}
	b := &Config{}
	a.EnsureDefaults()
	b.EnsureDefaults()

	if a.APIKey == b.APIKey {
		t.Error("two independent EnsureDefaults() calls generated the same APIKey, want distinct random keys")
	}
}

func TestGenerateIDProducesDistinctHexIDs(t *testing.T) {
	a := GenerateID()
	b := GenerateID()

	if a == b {
		t.Fatalf("GenerateID() returned the same value twice: %q", a)
	}
	if len(a) != 16 { // 8 bytes hex-encoded
		t.Errorf("GenerateID() length = %d, want 16 (8 bytes hex-encoded)", len(a))
	}
}

func TestLoadReturnsEmptyConfigWhenFileMissing(t *testing.T) {
	withConfigHome(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil when config.json does not exist", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config, want a usable empty *Config")
	}
	if cfg.ActiveVaultID != "" || len(cfg.Vaults) != 0 {
		t.Errorf("Load() on missing file = %+v, want zero-value Config", cfg)
	}
}

func TestLoadReturnsErrorOnMalformedJSON(t *testing.T) {
	withConfigHome(t)

	dir := getConfigDir()
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(getConfigPath(), []byte("{not valid json"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want an error for malformed config.json")
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	withConfigHome(t)

	cfg := &Config{
		ActiveVaultID: "vault-1",
		InboxPath:     "/tmp/inbox",
		Vaults: []Vault{
			{ID: "vault-1", Name: "Default", Path: "/tmp/vault-1", CreatedAt: "2026-01-01T00:00:00Z"},
		},
		APIEnabled: true,
		APIPort:    8080,
		APIKey:     "round-trip-key",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Confirm the file actually landed where getConfigPath() says it should.
	if _, err := os.Stat(getConfigPath()); err != nil {
		t.Fatalf("expected config.json at %s: %v", getConfigPath(), err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ActiveVaultID != cfg.ActiveVaultID {
		t.Errorf("ActiveVaultID = %q, want %q", loaded.ActiveVaultID, cfg.ActiveVaultID)
	}
	if loaded.InboxPath != cfg.InboxPath {
		t.Errorf("InboxPath = %q, want %q", loaded.InboxPath, cfg.InboxPath)
	}
	if loaded.APIEnabled != cfg.APIEnabled {
		t.Errorf("APIEnabled = %v, want %v", loaded.APIEnabled, cfg.APIEnabled)
	}
	if loaded.APIPort != cfg.APIPort {
		t.Errorf("APIPort = %d, want %d", loaded.APIPort, cfg.APIPort)
	}
	if loaded.APIKey != cfg.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, cfg.APIKey)
	}
	if len(loaded.Vaults) != 1 || loaded.Vaults[0].ID != "vault-1" {
		t.Errorf("Vaults = %+v, want one vault with ID vault-1", loaded.Vaults)
	}
}

func TestSavePersistsIndentedJSON(t *testing.T) {
	withConfigHome(t)

	cfg := &Config{ActiveVaultID: "abc"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var probe map[string]any
	if err := json.Unmarshal(data, &probe); err != nil {
		t.Fatalf("saved config.json is not valid JSON: %v", err)
	}
	if probe["activeVaultId"] != "abc" {
		t.Errorf("activeVaultId = %v, want %q", probe["activeVaultId"], "abc")
	}
}

func TestLoadOrDefaultFreshInstallCreatesDefaultVault(t *testing.T) {
	withConfigHome(t)

	cfg := LoadOrDefault()

	if len(cfg.Vaults) != 1 {
		t.Fatalf("Vaults = %+v, want exactly one default vault on fresh install", cfg.Vaults)
	}
	if cfg.Vaults[0].Name != "Default" {
		t.Errorf("Vaults[0].Name = %q, want %q", cfg.Vaults[0].Name, "Default")
	}
	if cfg.ActiveVaultID == "" || cfg.ActiveVaultID != cfg.Vaults[0].ID {
		t.Errorf("ActiveVaultID = %q, want it to match the created vault's ID %q", cfg.ActiveVaultID, cfg.Vaults[0].ID)
	}
	if cfg.InboxPath == "" {
		t.Error("InboxPath is empty, want it co-located with the default vault")
	}
	wantInbox := filepath.Join(cfg.Vaults[0].Path, "inbox")
	if cfg.InboxPath != wantInbox {
		t.Errorf("InboxPath = %q, want %q", cfg.InboxPath, wantInbox)
	}

	// LoadOrDefault must have persisted the generated config to disk.
	if _, err := os.Stat(getConfigPath()); err != nil {
		t.Fatalf("expected config.json to be written by LoadOrDefault: %v", err)
	}
}

func TestLoadOrDefaultIsStableAcrossCalls(t *testing.T) {
	withConfigHome(t)

	first := LoadOrDefault()
	second := LoadOrDefault()

	if first.ActiveVaultID != second.ActiveVaultID {
		t.Errorf("ActiveVaultID changed across LoadOrDefault() calls: %q vs %q", first.ActiveVaultID, second.ActiveVaultID)
	}
	if len(second.Vaults) != 1 {
		t.Errorf("second LoadOrDefault() Vaults = %+v, want the same single vault re-read from disk, not a new one", second.Vaults)
	}
}

func TestLoadOrDefaultMigratesLegacyDatabasePath(t *testing.T) {
	withConfigHome(t)

	legacy := &Config{DatabasePath: "/legacy/path/to/db"}
	if err := Save(legacy); err != nil {
		t.Fatalf("Save legacy config: %v", err)
	}

	cfg := LoadOrDefault()

	if cfg.DatabasePath != "" {
		t.Errorf("DatabasePath = %q, want cleared after legacy migration", cfg.DatabasePath)
	}
	if len(cfg.Vaults) != 1 {
		t.Fatalf("Vaults = %+v, want exactly one migrated vault", cfg.Vaults)
	}
	if cfg.Vaults[0].Path != "/legacy/path/to/db" {
		t.Errorf("Vaults[0].Path = %q, want migrated legacy path %q", cfg.Vaults[0].Path, "/legacy/path/to/db")
	}
	if cfg.Vaults[0].Name != "Default" {
		t.Errorf("Vaults[0].Name = %q, want %q", cfg.Vaults[0].Name, "Default")
	}
	if cfg.ActiveVaultID != cfg.Vaults[0].ID {
		t.Errorf("ActiveVaultID = %q, want it to match the migrated vault's ID %q", cfg.ActiveVaultID, cfg.Vaults[0].ID)
	}
}

func TestLoadOrDefaultFillsMissingActiveVaultID(t *testing.T) {
	withConfigHome(t)

	cfg := &Config{
		Vaults: []Vault{
			{ID: "v-existing", Name: "Existing", Path: "/some/path"},
		},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := LoadOrDefault()

	if loaded.ActiveVaultID != "v-existing" {
		t.Errorf("ActiveVaultID = %q, want backfilled to the only registered vault %q", loaded.ActiveVaultID, "v-existing")
	}
}

func TestLoadOrDefaultFillsMissingInboxPath(t *testing.T) {
	withConfigHome(t)

	cfg := &Config{
		ActiveVaultID: "v-existing",
		Vaults: []Vault{
			{ID: "v-existing", Name: "Existing", Path: "/some/vault/path"},
		},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := LoadOrDefault()

	want := filepath.Join("/some/vault/path", "inbox")
	if loaded.InboxPath != want {
		t.Errorf("InboxPath = %q, want %q", loaded.InboxPath, want)
	}
}

func TestGetDefaultDatabasePathAndConfigDirAreNonEmpty(t *testing.T) {
	withConfigHome(t)

	if GetDefaultDatabasePath() == "" {
		t.Error("GetDefaultDatabasePath() is empty")
	}
	if GetConfigDir() == "" {
		t.Error("GetConfigDir() is empty")
	}
}
