package main

import (
	"fmt"

	"github.com/hollis-labs/nil/config"
)

// --- Vault management methods (Wails-bound) ---

// GetVaults returns all registered vaults.
func (a *App) GetVaults() ([]config.Vault, error) {
	if a.vaultMgr == nil {
		return nil, fmt.Errorf("vault manager not initialized")
	}
	return a.vaultMgr.GetVaults(), nil
}

// GetActiveVault returns the currently active vault metadata.
func (a *App) GetActiveVault() (*config.Vault, error) {
	if a.vaultMgr == nil {
		return nil, fmt.Errorf("vault manager not initialized")
	}
	v := a.vaultMgr.GetActiveVault()
	if v == nil {
		return nil, fmt.Errorf("no active vault")
	}
	return v, nil
}

// CreateVault registers a new vault at the given directory path.
func (a *App) CreateVault(name, dir string) (*config.Vault, error) {
	if a.vaultMgr == nil {
		return nil, fmt.Errorf("vault manager not initialized")
	}
	return a.vaultMgr.CreateVault(name, dir)
}

// RenameVault updates the display name of a vault.
func (a *App) RenameVault(id, name string) error {
	if a.vaultMgr == nil {
		return fmt.Errorf("vault manager not initialized")
	}
	return a.vaultMgr.RenameVault(id, name)
}

// DeleteVault removes a vault from the registry (files on disk are preserved).
func (a *App) DeleteVault(id string) error {
	if a.vaultMgr == nil {
		return fmt.Errorf("vault manager not initialized")
	}
	return a.vaultMgr.DeleteVault(id)
}

// SwitchVault changes the active vault. Returns the new active vault's metadata.
func (a *App) SwitchVault(id string) (*config.Vault, error) {
	if a.vaultMgr == nil {
		return nil, fmt.Errorf("vault manager not initialized")
	}
	if err := a.vaultMgr.SwitchVault(id); err != nil {
		return nil, err
	}
	return a.vaultMgr.GetActiveVault(), nil
}
