package chat

import "nanite/store"

// ChatSession represents one open chat session against a vault.
type ChatSession struct {
	ID        int64  `json:"id"`
	VaultID   string `json:"vault_id"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
	DryRun    bool   `json:"dry_run"`
}

// ChatMessage is a single turn in a chat session.
type ChatMessage struct {
	ID        int64  `json:"id"`
	SessionID int64  `json:"session_id"`
	Role      string `json:"role"` // user | assistant | system
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// ActionProposal is a structured mutation the LLM has proposed.
// It awaits explicit user approval before execution.
type ActionProposal struct {
	ID         int64          `json:"id"`
	SessionID  int64          `json:"session_id"`
	MessageID  *int64         `json:"message_id,omitempty"`
	ActionType string         `json:"action_type"` // create | update | delete
	ItemType   string         `json:"item_type"`   // todo | note
	VaultID    string         `json:"vault_id"`
	Payload    store.Item     `json:"payload"`
	Diff       map[string]any `json:"diff,omitempty"` // for update: {field: [old, new]}
	Status     string         `json:"status"`         // pending | approved | denied | executed | failed | dry_run
	CreatedAt  string         `json:"created_at"`
	ResolvedAt string         `json:"resolved_at,omitempty"`
	ErrorMsg   string         `json:"error_msg,omitempty"`
}

// AuditEntry is an immutable record of a proposed or executed action.
type AuditEntry struct {
	ID         int64  `json:"id"`
	ProposalID *int64 `json:"proposal_id,omitempty"`
	ActionType string `json:"action_type"`
	ItemType   string `json:"item_type"`
	VaultID    string `json:"vault_id"`
	ItemID     *int64 `json:"item_id,omitempty"` // nil for dry_run / denied / failed
	Outcome    string `json:"outcome"`           // proposed | approved | denied | executed | failed | dry_run
	Payload    string `json:"payload_json"`      // JSON snapshot of the payload at proposal time
	Actor      string `json:"actor"`             // "user" | "ai"
	Timestamp  string `json:"timestamp"`
}

// ChatResponse is returned by SendChatMessage. It contains the assistant's
// reply message and, if the LLM emitted a mutating action, a pending proposal.
type ChatResponse struct {
	Message    ChatMessage     `json:"message"`
	ProposalID *int64          `json:"proposal_id,omitempty"`
	Proposal   *ActionProposal `json:"proposal,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// ActionResult is returned by ApproveChatAction.
type ActionResult struct {
	ProposalID int64  `json:"proposal_id"`
	ItemID     *int64 `json:"item_id,omitempty"`
	DryRun     bool   `json:"dry_run"`
	Outcome    string `json:"outcome"` // executed | dry_run
}
