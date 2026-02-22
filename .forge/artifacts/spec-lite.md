---
intent: project_doc
class: spec_lite
feature: F5 — Vault Chat
created_at: 2026-02-22
iteration: 1
---

# Spec-lite — F5: Vault Chat

## Data Model

### chat.db (separate SQLite, in config dir)

```sql
-- Chat sessions (one per open/close cycle or named session in future)
CREATE TABLE IF NOT EXISTS chat_sessions (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  vault_id TEXT NOT NULL,              -- active vault ID at session open
  started_at TEXT NOT NULL DEFAULT (datetime('now')),
  ended_at   TEXT DEFAULT NULL,
  dry_run    INTEGER NOT NULL DEFAULT 0
);

-- Chat messages (user and assistant turns)
CREATE TABLE IF NOT EXISTS chat_messages (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id  INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
  role        TEXT NOT NULL CHECK(role IN ('user','assistant','system')),
  content     TEXT NOT NULL,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Action proposals (may be approved, denied, or pending)
CREATE TABLE IF NOT EXISTS action_proposals (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id   INTEGER NOT NULL REFERENCES chat_sessions(id),
  message_id   INTEGER REFERENCES chat_messages(id),
  action_type  TEXT NOT NULL,    -- create|update|delete
  item_type    TEXT NOT NULL,    -- todo|note
  vault_id     TEXT NOT NULL,
  payload_json TEXT NOT NULL,    -- proposed fields (title, priority, tags, etc.)
  diff_json    TEXT DEFAULT NULL,-- for updates: {field: [old, new]}
  status       TEXT NOT NULL DEFAULT 'pending',  -- pending|approved|denied|executed|failed
  created_at   TEXT NOT NULL DEFAULT (datetime('now')),
  resolved_at  TEXT DEFAULT NULL,
  error_msg    TEXT DEFAULT NULL
);

-- Audit log (immutable, append-only)
CREATE TABLE IF NOT EXISTS action_audit (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  proposal_id     INTEGER REFERENCES action_proposals(id),
  action_type     TEXT NOT NULL,
  item_type       TEXT NOT NULL,
  vault_id        TEXT NOT NULL,
  item_id         INTEGER DEFAULT NULL,  -- NULL if create failed or dry_run
  outcome         TEXT NOT NULL,         -- proposed|approved|denied|executed|failed|dry_run
  payload_json    TEXT NOT NULL,
  actor           TEXT NOT NULL DEFAULT 'user',  -- 'user' or 'ai'
  timestamp       TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### Config Additions (config.Config)
```go
// ChatConfig holds settings for the Vault Chat addon.
type ChatConfig struct {
  Enabled       bool              `json:"enabled"`
  APIKey        string            `json:"apiKey"`        // Claude API key
  Model         string            `json:"model"`         // e.g. "claude-haiku-4-5-20251001"
  DryRun        bool              `json:"dryRun"`
  VaultCaps     map[string]VaultCap `json:"vaultCaps"`  // key: vault ID
}

type VaultCap struct {
  Read   bool `json:"read"`
  Write  bool `json:"write"`
  Delete bool `json:"delete"`
}
```

## Go Structs (new package: `chat/`)

```go
// ChatSession — open session state
type ChatSession struct {
  ID        int64  `json:"id"`
  VaultID   string `json:"vault_id"`
  StartedAt string `json:"started_at"`
  DryRun    bool   `json:"dry_run"`
}

// ChatMessage — one turn
type ChatMessage struct {
  ID        int64  `json:"id"`
  SessionID int64  `json:"session_id"`
  Role      string `json:"role"`       // user|assistant|system
  Content   string `json:"content"`
  CreatedAt string `json:"created_at"`
}

// ActionProposal — structured action awaiting approval
type ActionProposal struct {
  ID         int64           `json:"id"`
  SessionID  int64           `json:"session_id"`
  ActionType string          `json:"action_type"` // create|update|delete
  ItemType   string          `json:"item_type"`   // todo|note
  VaultID    string          `json:"vault_id"`
  Payload    store.Item      `json:"payload"`
  Diff       map[string]any  `json:"diff,omitempty"`
  Status     string          `json:"status"`
}
```

## Action Proposal → Approval → Execute Flow

```
User types request
       ↓
LLM interprets → emits StructuredResponse{
    text: "I'll create a todo: ...",
    action?: ActionProposal (action_type, item_type, payload)
}
       ↓
If action present:
  → Store in action_proposals (status=pending)
  → Audit: outcome=proposed
  → Return to frontend with proposal_id
       ↓
Frontend renders ActionCard (field diffs, approve/deny buttons)
       ↓
User approves:
  → Check DryRun flag
  → If dry_run: audit(outcome=dry_run), show "Dry run — not executed"
  → Else: execute via store.Store method (CreateItem/UpdateItem/DeleteItem)
  → Update action_proposals(status=executed)
  → Audit: outcome=executed, item_id=<result id>
       ↓
User denies:
  → Update action_proposals(status=denied)
  → Audit: outcome=denied
```

## LLM Bridge Contract

### System Prompt (injected per request)
```
You are a NANITE vault assistant. You have access to the user's vault items.
When the user asks a question, search the vault and return relevant items.
When the user requests a mutation (create/edit/delete), emit a structured JSON action.
Never execute mutations directly — always emit an ActionProposal for user approval.
Vault: {vault_name} | Mode: {dry_run ? "DRY RUN — proposals only" : "live"}
Capabilities: {read: true, write: {write_cap}, delete: {delete_cap}}
```

### LLM Response Format
The LLM responds with either:
- Plain text (for search/read queries)
- JSON block embedded in response for mutations:
```json
{
  "action": {
    "type": "create|update|delete",
    "item_type": "todo|note",
    "vault_id": "...",
    "payload": { ...store.Item fields... },
    "diff": { "field": ["old", "new"] }
  }
}
```
The Go bridge parses this, extracts the action, and returns a `StructuredResponse`.

## Wails-Bound Methods (new, on `*App`)

```go
// Chat session management
func (a *App) StartChatSession() (*chat.ChatSession, error)
func (a *App) EndChatSession(sessionID int64) error

// Send message — returns assistant response + optional proposal
func (a *App) SendChatMessage(sessionID int64, content string) (*chat.ChatResponse, error)

// Action flow
func (a *App) ApproveChatAction(proposalID int64) (*chat.ActionResult, error)
func (a *App) DenyChatAction(proposalID int64) error

// History + audit
func (a *App) GetChatHistory(sessionID int64) ([]chat.ChatMessage, error)
func (a *App) GetActionAudit(limit int) ([]chat.AuditEntry, error)

// Config
func (a *App) GetChatConfig() (*config.ChatConfig, error)
func (a *App) SetChatConfig(cfg config.ChatConfig) error
```

## Frontend Components

### `ChatPanel` (`src/components/ChatPanel.tsx`)
- Keyboard shortcut: `Cmd+Shift+C` toggles open/close
- Layout: floating panel or side panel (TBD by iteration 6)
- Contains: message thread, input box, session status indicator

### `ChatMessage` (`src/components/ChatMessage.tsx`)
- Renders a single turn: user or assistant
- Assistant messages may include an embedded `ActionCard`

### `ActionCard` (`src/components/ActionCard.tsx`)
- Renders: action type, item type, vault name, field diffs
- Controls: "Approve" button, "Deny" button
- State: pending | approved | denied | executed | dry_run

## Capability Defaults
- All vaults: `read: true, write: false, delete: false` by default
- User must explicitly enable write/delete per vault in Settings → Chat

## Evidence
- Created: 2026-02-22 (Inception iteration 1)
- Design informed by: store/models.go, config/config.go, app.go patterns, CLAUDE.md conventions
