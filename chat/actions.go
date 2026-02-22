package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"nanite/store"
)

// ActionRunner wires together ChatStore (persistence) and store.Store (vault mutations).
// It implements the Propose → Approve/Deny → Execute flow with audit trail.
type ActionRunner struct {
	chatStore *ChatStore
}

// NewActionRunner creates an ActionRunner backed by the given ChatStore.
func NewActionRunner(cs *ChatStore) *ActionRunner {
	return &ActionRunner{chatStore: cs}
}

// Propose persists a new ActionProposal (status=pending) and appends an audit record
// with outcome=proposed. Returns the proposal ID.
func (r *ActionRunner) Propose(ctx context.Context, p *ActionProposal) (int64, error) {
	proposalID, err := r.chatStore.CreateProposal(ctx, p)
	if err != nil {
		return 0, fmt.Errorf("chat/actions: persist proposal: %w", err)
	}

	payloadJSON, _ := json.Marshal(p.Payload)
	if err := r.chatStore.AppendAudit(ctx, &AuditEntry{
		ProposalID: &proposalID,
		ActionType: p.ActionType,
		ItemType:   p.ItemType,
		VaultID:    p.VaultID,
		Outcome:    "proposed",
		Payload:    string(payloadJSON),
		Actor:      "ai",
	}); err != nil {
		return proposalID, fmt.Errorf("chat/actions: append proposed audit: %w", err)
	}

	return proposalID, nil
}

// Approve executes the proposal (or marks it as dry_run) and records the outcome.
// caps controls whether the vault allows the requested operation.
func (r *ActionRunner) Approve(
	ctx context.Context,
	proposalID int64,
	vaultStore *store.Store,
	dryRun bool,
	caps VaultCaps,
) (*ActionResult, error) {
	p, err := r.chatStore.GetProposal(ctx, proposalID)
	if err != nil {
		return nil, fmt.Errorf("chat/actions: get proposal %d: %w", proposalID, err)
	}
	if p.Status != "pending" {
		return nil, fmt.Errorf("chat/actions: proposal %d is not pending (status=%s)", proposalID, p.Status)
	}

	// Capability check.
	if err := checkCaps(p.ActionType, caps); err != nil {
		_ = r.chatStore.UpdateProposalStatus(ctx, proposalID, "failed", err.Error())
		return nil, err
	}

	payloadJSON, _ := json.Marshal(p.Payload)

	if dryRun {
		_ = r.chatStore.UpdateProposalStatus(ctx, proposalID, "dry_run", "")
		_ = r.chatStore.AppendAudit(ctx, &AuditEntry{
			ProposalID: &proposalID,
			ActionType: p.ActionType,
			ItemType:   p.ItemType,
			VaultID:    p.VaultID,
			Outcome:    "dry_run",
			Payload:    string(payloadJSON),
			Actor:      "user",
		})
		return &ActionResult{
			ProposalID: proposalID,
			DryRun:     true,
			Outcome:    "dry_run",
		}, nil
	}

	// Execute.
	var itemID *int64
	var execErr error

	switch p.ActionType {
	case "create":
		item := p.Payload
		item.Type = p.ItemType
		created, err := vaultStore.CreateItem(ctx, &item)
		if err != nil {
			execErr = err
		} else {
			id := created.ID
			itemID = &id
		}

	case "update":
		item := p.Payload
		execErr = vaultStore.UpdateItem(ctx, &item)
		if execErr == nil {
			id := item.ID
			itemID = &id
		}

	case "delete":
		execErr = vaultStore.DeleteItem(ctx, p.Payload.ID)
		if execErr == nil {
			id := p.Payload.ID
			itemID = &id
		}

	default:
		execErr = fmt.Errorf("chat/actions: unknown action type %q", p.ActionType)
	}

	if execErr != nil {
		errMsg := execErr.Error()
		_ = r.chatStore.UpdateProposalStatus(ctx, proposalID, "failed", errMsg)
		_ = r.chatStore.AppendAudit(ctx, &AuditEntry{
			ProposalID: &proposalID,
			ActionType: p.ActionType,
			ItemType:   p.ItemType,
			VaultID:    p.VaultID,
			Outcome:    "failed",
			Payload:    string(payloadJSON),
			Actor:      "user",
		})
		return nil, fmt.Errorf("chat/actions: execute %s: %w", p.ActionType, execErr)
	}

	_ = r.chatStore.UpdateProposalStatus(ctx, proposalID, "executed", "")
	_ = r.chatStore.AppendAudit(ctx, &AuditEntry{
		ProposalID: &proposalID,
		ActionType: p.ActionType,
		ItemType:   p.ItemType,
		VaultID:    p.VaultID,
		ItemID:     itemID,
		Outcome:    "executed",
		Payload:    string(payloadJSON),
		Actor:      "user",
	})

	return &ActionResult{
		ProposalID: proposalID,
		ItemID:     itemID,
		DryRun:     false,
		Outcome:    "executed",
	}, nil
}

// Deny marks a pending proposal as denied and appends an audit record.
func (r *ActionRunner) Deny(ctx context.Context, proposalID int64) error {
	p, err := r.chatStore.GetProposal(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("chat/actions: get proposal %d: %w", proposalID, err)
	}
	if p.Status != "pending" {
		return fmt.Errorf("chat/actions: proposal %d is not pending (status=%s)", proposalID, p.Status)
	}

	if err := r.chatStore.UpdateProposalStatus(ctx, proposalID, "denied", ""); err != nil {
		return err
	}

	payloadJSON, _ := json.Marshal(p.Payload)
	return r.chatStore.AppendAudit(ctx, &AuditEntry{
		ProposalID: &proposalID,
		ActionType: p.ActionType,
		ItemType:   p.ItemType,
		VaultID:    p.VaultID,
		Outcome:    "denied",
		Payload:    string(payloadJSON),
		Actor:      "user",
	})
}

// checkCaps returns an error if the vault capability config disallows the action.
func checkCaps(actionType string, caps VaultCaps) error {
	switch actionType {
	case "create", "update":
		if !caps.Write {
			return fmt.Errorf("chat: vault does not allow write operations — enable in Settings → Chat")
		}
	case "delete":
		if !caps.Delete {
			return fmt.Errorf("chat: vault does not allow delete operations — enable in Settings → Chat")
		}
	}
	return nil
}
