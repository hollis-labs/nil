package main

import (
	"github.com/hollis-labs/nil/store"
)

// --- Inbox methods ---

func (a *App) GetInboxCount() (int, error) {
	return svc.InboxCount(a.ctx, a.vaultMgr.InboxStore())
}

func (a *App) GetInboxItems(req store.SearchRequest) ([]store.Item, error) {
	return svc.ListInbox(a.ctx, a.vaultMgr.InboxStore(), req)
}

// ProcessInboxItem moves an inbox item to the target vault.
// If targetVaultID is empty, defaults to the currently active vault.
func (a *App) ProcessInboxItem(id int64, targetVaultID string) error {
	if targetVaultID == "" {
		targetVaultID = a.vaultMgr.GetActiveVaultID()
	}
	return a.vaultMgr.MoveItemToVault(a.ctx, id, targetVaultID)
}
