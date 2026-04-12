package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Vault represents a named SQLite database in the vault registry.
type Vault struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

// VaultCap describes which chat operations are permitted on a given vault.
// Default (zero value) is read-only.
type VaultCap struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Delete       bool `json:"delete"`
	DirectCreate bool `json:"directCreate"` // allow create_item tool to bypass the Propose→Approve flow
}

// ChatConfig holds settings for the Vault Chat addon (F5).
type ChatConfig struct {
	Enabled   bool                `json:"enabled"`
	APIKey    string              `json:"apiKey"`
	Model     string              `json:"model"` // e.g. "claude-haiku-4-5-20251001"
	DryRun    bool                `json:"dryRun"`
	VaultCaps map[string]VaultCap `json:"vaultCaps"` // key: vault ID
}

type Config struct {
	// Legacy field — kept for migration from pre-vault installs only.
	DatabasePath string `json:"databasePath,omitempty"`

	// Vault registry
	ActiveVaultID string  `json:"activeVaultId"`
	InboxPath     string  `json:"inboxPath"`
	Vaults        []Vault `json:"vaults"`

	// API config
	APIEnabled bool   `json:"apiEnabled"`
	APIPort    int    `json:"apiPort"`
	APIKey     string `json:"apiKey"`

	// Chat addon config
	Chat ChatConfig `json:"chat"`
}

// EnsureDefaults sets APIPort and APIKey if they are zero/empty.
// Returns true if any field was changed.
func (c *Config) EnsureDefaults() bool {
	changed := false
	if c.APIPort == 0 {
		c.APIPort = 7765
		changed = true
	}
	if c.APIKey == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		c.APIKey = hex.EncodeToString(b)
		changed = true
	}
	return changed
}

// GenerateID creates a random 8-byte hex ID suitable for vault IDs.
func GenerateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// getConfigDir returns the OS-appropriate config directory
func getConfigDir() string {
	switch runtime.GOOS {
	case "darwin":
		home := os.Getenv("HOME")
		return filepath.Join(home, ".config", "nil")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appdata, "Nil")
	default: // linux
		home := os.Getenv("HOME")
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		return filepath.Join(configHome, "nil")
	}
}

// getDefaultDatabasePath returns the OS-appropriate default database directory
func getDefaultDatabasePath() string {
	switch runtime.GOOS {
	case "darwin":
		home := os.Getenv("HOME")
		return filepath.Join(home, "Library", "Application Support", "Nil")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appdata, "Nil", "data")
	default: // linux
		home := os.Getenv("HOME")
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			dataHome = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(dataHome, "nil")
	}
}

// getConfigPath returns the full path to config.json
func getConfigPath() string {
	return filepath.Join(getConfigDir(), "config.json")
}

// Load reads the config from disk, returns empty config if not found
func Load() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadOrDefault loads config, performs any needed migrations, and ensures
// at least one vault and an inbox path are configured. Always returns a usable config.
func LoadOrDefault() *Config {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{}
	}

	changed := false

	// Migrate legacy single-vault config (pre-vault-registry installs)
	if cfg.DatabasePath != "" && len(cfg.Vaults) == 0 {
		id := GenerateID()
		cfg.Vaults = []Vault{{
			ID:        id,
			Name:      "Default",
			Path:      cfg.DatabasePath,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}}
		cfg.ActiveVaultID = id
		cfg.DatabasePath = "" // clear legacy field
		changed = true
	}

	// Fresh install: create the default vault
	if len(cfg.Vaults) == 0 {
		id := GenerateID()
		cfg.Vaults = []Vault{{
			ID:        id,
			Name:      "Default",
			Path:      getDefaultDatabasePath(),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}}
		cfg.ActiveVaultID = id
		changed = true
	}

	// Ensure ActiveVaultID is set
	if cfg.ActiveVaultID == "" && len(cfg.Vaults) > 0 {
		cfg.ActiveVaultID = cfg.Vaults[0].ID
		changed = true
	}

	// Ensure InboxPath is set (co-located with first vault)
	if cfg.InboxPath == "" {
		cfg.InboxPath = filepath.Join(cfg.Vaults[0].Path, "inbox")
		changed = true
	}

	if changed {
		_ = Save(cfg)
	}

	return cfg
}

// Save writes the config to disk
func Save(cfg *Config) error {
	configDir := getConfigDir()

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	configPath := getConfigPath()
	return os.WriteFile(configPath, data, 0644)
}

// GetDefaultDatabasePath exposes the default path for UI display
func GetDefaultDatabasePath() string {
	return getDefaultDatabasePath()
}

// GetConfigDir exposes the OS-appropriate config directory for use by sub-packages.
func GetConfigDir() string {
	return getConfigDir()
}
