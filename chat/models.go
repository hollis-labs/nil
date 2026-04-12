package chat

import (
	"context"
	"encoding/json"
	"github.com/hollis-labs/nil/store"
	"sync"
)

// BridgeStore is the subset of store.Store the Bridge needs for tool execution.
type BridgeStore interface {
	Search(ctx context.Context, req store.SearchRequest) ([]store.Item, error)
	GetItem(ctx context.Context, id int64) (*store.Item, error)
	CreateItem(ctx context.Context, t *store.Item) (*store.Item, error)
	GetTaxonomyWithCounts(ctx context.Context) (projects, contexts, tags []store.TaxonomyItem, err error)
	GetStats(ctx context.Context) (*store.VaultStats, error)
}

// TemplateStore is the subset of ChatStore the Bridge needs for template tool execution.
type TemplateStore interface {
	ListTemplates(ctx context.Context) ([]Template, error)
	GetTemplate(ctx context.Context, slug string) (*Template, error)
}

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

// ToolCallRecord captures one tool invocation made during an agentic loop turn.
type ToolCallRecord struct {
	ToolName   string `json:"tool_name"`
	InputJSON  string `json:"input_json"`
	ResultJSON string `json:"result_json"`
	CacheHit   bool   `json:"cache_hit,omitempty"`
}

// ChatResponse is returned by SendChatMessage. It contains the assistant's
// reply message and, if the LLM emitted a mutating action, a pending proposal.
type ChatResponse struct {
	Message    ChatMessage      `json:"message"`
	ProposalID *int64           `json:"proposal_id,omitempty"`
	Proposal   *ActionProposal  `json:"proposal,omitempty"`
	ToolCalls  []ToolCallRecord `json:"tool_calls,omitempty"`
	Error      string           `json:"error,omitempty"`
}

// ToolCache is an in-memory, session-scoped LRU cache for tool results.
// It prevents redundant DB queries when Claude repeats the same search
// (e.g. "sort that by title instead") within one conversation session.
type ToolCache struct {
	mu      sync.Mutex
	entries []toolCacheEntry
	maxSize int
}

type toolCacheEntry struct {
	key    string // toolName + ":" + inputJSON
	result string
}

// NewToolCache creates a cache that holds up to maxSize recent tool results.
func NewToolCache(maxSize int) *ToolCache {
	if maxSize <= 0 {
		maxSize = 20
	}
	return &ToolCache{maxSize: maxSize}
}

// Get returns a cached result for the given tool call, or ("", false) on miss.
func (c *ToolCache) Get(toolName string, input json.RawMessage) (string, bool) {
	key := toolName + ":" + string(input)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.entries {
		if e.key == key {
			return e.result, true
		}
	}
	return "", false
}

// Set stores a tool result. If the key already exists it is not duplicated.
// The oldest entry is evicted when the cache is full.
func (c *ToolCache) Set(toolName string, input json.RawMessage, result string) {
	key := toolName + ":" + string(input)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.entries {
		if e.key == key {
			return
		}
	}
	if len(c.entries) >= c.maxSize {
		c.entries = c.entries[1:] // evict oldest (FIFO)
	}
	c.entries = append(c.entries, toolCacheEntry{key: key, result: result})
}

// Template is a reusable agentic workflow pattern stored in chat.db.
type Template struct {
	ID           int64  `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"` // generation | query | analysis
	Prompt       string `json:"prompt"`
	Parameters   string `json:"parameters"`    // JSON array of param names
	OutputFormat string `json:"output_format"` // markdown
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// ActionResult is returned by ApproveChatAction.
type ActionResult struct {
	ProposalID int64  `json:"proposal_id"`
	ItemID     *int64 `json:"item_id,omitempty"`
	DryRun     bool   `json:"dry_run"`
	Outcome    string `json:"outcome"` // executed | dry_run
}
