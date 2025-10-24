package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	DatabasePath string `json:"databasePath"`
}

// getConfigDir returns the OS-appropriate config directory
func getConfigDir() string {
	switch runtime.GOOS {
	case "darwin":
		home := os.Getenv("HOME")
		return filepath.Join(home, ".config", "planck")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appdata, "Planck")
	default: // linux
		home := os.Getenv("HOME")
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(home, ".config")
		}
		return filepath.Join(configHome, "planck")
	}
}

// getDefaultDatabasePath returns the OS-appropriate default database directory
func getDefaultDatabasePath() string {
	switch runtime.GOOS {
	case "darwin":
		home := os.Getenv("HOME")
		return filepath.Join(home, "Library", "Application Support", "Planck")
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appdata, "Planck", "data")
	default: // linux
		home := os.Getenv("HOME")
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			dataHome = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(dataHome, "planck")
	}
}

// getConfigPath returns the full path to config.json
func getConfigPath() string {
	return filepath.Join(getConfigDir(), "config.json")
}

// Load reads the config from disk, returns default if not found
func Load() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		// Config doesn't exist yet - return empty config (signals first run)
		return &Config{DatabasePath: ""}, nil
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

// LoadOrDefault loads config or returns default values
func LoadOrDefault() *Config {
	cfg, err := Load()
	if err != nil || cfg.DatabasePath == "" {
		return &Config{
			DatabasePath: getDefaultDatabasePath(),
		}
	}
	return cfg
}

// Save writes the config to disk
func Save(cfg *Config) error {
	configDir := getConfigDir()

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	configPath := getConfigPath()
	return os.WriteFile(configPath, data, 0644)
}

// GetDefaultDatabasePath exposes the default path for UI display
func GetDefaultDatabasePath() string {
	return getDefaultDatabasePath()
}
