package vault

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/store"
)

// Manager owns all open vault connections and the shared inbox store.
// It is safe for concurrent use.
type Manager struct {
	cfg        *config.Config
	ctx        context.Context
	inboxStore *store.Store
	activeID   string
	stores     map[string]*store.Store
	mu         sync.RWMutex
}

// NewManager opens the active vault and the shared inbox, returning a ready Manager.
func NewManager(ctx context.Context, cfg *config.Config) (*Manager, error) {
	m := &Manager{
		cfg:    cfg,
		ctx:    ctx,
		stores: make(map[string]*store.Store),
	}

	// Open shared inbox store
	if cfg.InboxPath != "" {
		s, err := store.Open(ctx, cfg.InboxPath)
		if err != nil {
			return nil, fmt.Errorf("opening inbox store: %w", err)
		}
		m.inboxStore = s
	}

	// Open active vault store
	if cfg.ActiveVaultID != "" {
		s, err := m.openVaultByID(cfg.ActiveVaultID)
		if err != nil {
			return nil, fmt.Errorf("opening active vault %q: %w", cfg.ActiveVaultID, err)
		}
		m.stores[cfg.ActiveVaultID] = s
		m.activeID = cfg.ActiveVaultID
	}

	return m, nil
}

// ActiveStore returns the currently active vault's store.
func (m *Manager) ActiveStore() *store.Store {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stores[m.activeID]
}

// InboxStore returns the shared inbox store.
func (m *Manager) InboxStore() *store.Store {
	return m.inboxStore
}

// GetActiveVaultID returns the ID of the currently active vault.
func (m *Manager) GetActiveVaultID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeID
}

// GetActiveVault returns a copy of the active vault metadata.
func (m *Manager) GetActiveVault() *config.Vault {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.findVault(m.activeID)
}

// GetVaults returns a copy of the full vault registry.
func (m *Manager) GetVaults() []config.Vault {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]config.Vault, len(m.cfg.Vaults))
	copy(result, m.cfg.Vaults)
	return result
}

// StoreForID returns the store for the given vault ID, opening it lazily if needed.
func (m *Manager) StoreForID(vaultID string) (*store.Store, error) {
	m.mu.RLock()
	s, ok := m.stores[vaultID]
	m.mu.RUnlock()
	if ok {
		return s, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock
	if s, ok := m.stores[vaultID]; ok {
		return s, nil
	}
	s, err := m.openVaultByID(vaultID)
	if err != nil {
		return nil, err
	}
	m.stores[vaultID] = s
	return s, nil
}

// SwitchVault changes the active vault, opening it if not already open.
// Persists the change to config.json.
func (m *Manager) SwitchVault(id string) error {
	if _, err := m.StoreForID(id); err != nil {
		return err
	}
	m.mu.Lock()
	m.activeID = id
	m.cfg.ActiveVaultID = id
	m.mu.Unlock()
	return config.Save(m.cfg)
}

// CreateVault registers a new vault, opens its store, and persists the registry.
func (m *Manager) CreateVault(name, dir string) (*config.Vault, error) {
	v := config.Vault{
		ID:        config.GenerateID(),
		Name:      name,
		Path:      dir,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	s, err := store.Open(m.ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("creating vault store: %w", err)
	}

	m.mu.Lock()
	m.stores[v.ID] = s
	m.cfg.Vaults = append(m.cfg.Vaults, v)
	m.mu.Unlock()

	if err := config.Save(m.cfg); err != nil {
		return nil, err
	}
	return &v, nil
}

// RenameVault updates the display name of a vault and persists the change.
func (m *Manager) RenameVault(id, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.cfg.Vaults {
		if m.cfg.Vaults[i].ID == id {
			m.cfg.Vaults[i].Name = name
			return config.Save(m.cfg)
		}
	}
	return fmt.Errorf("vault %q not found", id)
}

// DeleteVault removes a vault from the registry (does not delete files on disk).
// The active vault cannot be deleted.
func (m *Manager) DeleteVault(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == m.activeID {
		return fmt.Errorf("cannot delete the active vault")
	}
	if len(m.cfg.Vaults) <= 1 {
		return fmt.Errorf("cannot delete the only vault")
	}
	// Close open connection if any
	if s, ok := m.stores[id]; ok {
		_ = s.Close()
		delete(m.stores, id)
	}
	// Remove from registry
	for i, v := range m.cfg.Vaults {
		if v.ID == id {
			m.cfg.Vaults = append(m.cfg.Vaults[:i], m.cfg.Vaults[i+1:]...)
			return config.Save(m.cfg)
		}
	}
	return fmt.Errorf("vault %q not found", id)
}

// MoveItemToVault copies an item from the inbox store to the target vault,
// then deletes it from the inbox store.
func (m *Manager) MoveItemToVault(ctx context.Context, id int64, targetVaultID string) error {
	if m.inboxStore == nil {
		return fmt.Errorf("inbox store not initialized")
	}
	item, err := m.inboxStore.GetItem(ctx, id)
	if err != nil {
		return fmt.Errorf("item %d not found in inbox: %w", id, err)
	}

	target, err := m.StoreForID(targetVaultID)
	if err != nil {
		return err
	}

	// Create in target vault with inbox cleared
	item.ID = 0
	item.Inbox = false
	if item.Section == "" {
		item.Section = "anytime"
	}
	if _, err := target.CreateItem(ctx, item); err != nil {
		return fmt.Errorf("creating item in target vault: %w", err)
	}

	// Delete from inbox
	return m.inboxStore.DeleteItem(ctx, id)
}

// CloseAll closes all open store connections. Call during app shutdown.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.stores {
		_ = s.Close()
	}
	if m.inboxStore != nil {
		_ = m.inboxStore.Close()
	}
}

// findVault returns a pointer into cfg.Vaults for the given ID, or nil.
// Caller must hold at least a read lock.
func (m *Manager) findVault(id string) *config.Vault {
	for i := range m.cfg.Vaults {
		if m.cfg.Vaults[i].ID == id {
			v := m.cfg.Vaults[i]
			return &v
		}
	}
	return nil
}

// openVaultByID opens the store for a vault by looking up its path in the registry.
// Caller should hold a write lock or call before the manager is shared.
func (m *Manager) openVaultByID(id string) (*store.Store, error) {
	v := m.findVault(id)
	if v == nil {
		return nil, fmt.Errorf("vault %q not found in registry", id)
	}
	return store.Open(m.ctx, v.Path)
}
