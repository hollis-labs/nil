package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hollis-labs/nil/apiserver"
	"github.com/hollis-labs/nil/chat"
	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/vault"
)

// svc is the package-level items service used by every Wails-bound handler.
// Stateless; safe to share.
var svc = items.New()

type App struct {
	ctx        context.Context
	vaultMgr   *vault.Manager
	needsSetup bool
	apiServer  *http.Server
	apiMu      sync.Mutex

	// Chat addon (F5)
	chatStore       *chat.ChatStore
	chatRunner      *chat.ActionRunner
	chatBridge      *chat.Bridge
	sessionCaches   map[int64]*chat.ToolCache
	sessionCachesMu sync.Mutex
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load config (with migration and auto-defaults)
	cfg := config.LoadOrDefault()

	// Ensure API defaults (port, key) are present; save if anything changed
	if cfg.EnsureDefaults() {
		_ = config.Save(cfg)
	}

	// Initialize vault manager (opens active vault + shared inbox)
	vm, err := vault.NewManager(ctx, cfg)
	if err != nil {
		println("Failed to initialize vault manager:", err.Error())
		a.needsSetup = true
		return
	}
	a.vaultMgr = vm

	if cfg.APIEnabled {
		a.startAPIServer(cfg)
	}

	// Initialise chat addon
	cs, err := chat.Open(ctx, config.GetConfigDir())
	if err != nil {
		println("Failed to initialize chat store:", err.Error())
	} else {
		a.chatStore = cs
		a.chatRunner = chat.NewActionRunner(cs)
		a.chatBridge = &chat.Bridge{}
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.stopAPIServer()
	if a.vaultMgr != nil {
		a.vaultMgr.CloseAll()
	}
	if a.chatStore != nil {
		_ = a.chatStore.Close()
	}
}

func (a *App) startAPIServer(cfg *config.Config) {
	if a.vaultMgr == nil {
		return
	}
	a.apiMu.Lock()
	defer a.apiMu.Unlock()
	if a.apiServer != nil {
		return // already running
	}
	srv := apiserver.New(cfg, a.vaultMgr)
	a.apiServer = srv
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			println("API server error:", err.Error())
		}
	}()
}

func (a *App) stopAPIServer() {
	a.apiMu.Lock()
	defer a.apiMu.Unlock()
	if a.apiServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = a.apiServer.Shutdown(ctx)
	a.apiServer = nil
}

// --- Setup / config methods ---

// NeedsSetup returns true if the app needs initial database configuration
func (a *App) NeedsSetup() bool {
	return a.needsSetup
}

// GetDatabasePath returns the active vault's directory path
func (a *App) GetDatabasePath() (string, error) {
	if a.vaultMgr != nil {
		if v := a.vaultMgr.GetActiveVault(); v != nil {
			return v.Path, nil
		}
	}
	return config.GetDefaultDatabasePath(), nil
}

// GetDefaultDatabasePath returns the OS-appropriate default path
func (a *App) GetDefaultDatabasePath() string {
	return config.GetDefaultDatabasePath()
}

// SetDatabasePath updates the active vault's path. Kept for backward compat
// with the setup/settings UI; new code should use vault CRUD methods.
func (a *App) SetDatabasePath(path string) error {
	// Validate path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}
	testFile := path + "/.nil-write-test"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("directory not writable: %s", path)
	}
	os.Remove(testFile)

	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	// Update the active vault's path in registry
	for i := range cfg.Vaults {
		if cfg.Vaults[i].ID == cfg.ActiveVaultID {
			cfg.Vaults[i].Path = path
			break
		}
	}
	// Fallback: legacy single-path field
	if len(cfg.Vaults) == 0 {
		cfg.DatabasePath = path
	}
	return config.Save(cfg)
}

// --- API config ---

// APIConfigResult is the shape returned to the Settings UI.
type APIConfigResult struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	APIKey  string `json:"api_key"`
}

// GetAPIConfig returns the current API configuration for display in Settings.
func (a *App) GetAPIConfig() (*APIConfigResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	cfg.EnsureDefaults()
	return &APIConfigResult{
		Enabled: cfg.APIEnabled,
		Port:    cfg.APIPort,
		APIKey:  cfg.APIKey,
	}, nil
}

// SetAPIEnabled enables or disables the local HTTP API at runtime.
func (a *App) SetAPIEnabled(enabled bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.EnsureDefaults()
	cfg.APIEnabled = enabled
	if err := config.Save(cfg); err != nil {
		return err
	}
	if enabled {
		a.startAPIServer(cfg)
	} else {
		a.stopAPIServer()
	}
	return nil
}

// Restart restarts the application
func (a *App) Restart() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	cmd := exec.Command(executable, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart: %w", err)
	}
	os.Exit(0)
	return nil
}
