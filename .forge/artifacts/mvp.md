---
intent: project_doc
class: mvp_definition
feature: F5 — Vault Chat
created_at: 2026-02-22
iteration: 1
---

# MVP Definition — F5: Vault Chat

## What "Working" Means

A user can open a chat panel inside NANITE and:

1. **Search** their vault by typing a natural-language query (e.g. "find tasks tagged urgent due this week")
2. **Receive** a response with matching items rendered inline
3. **Request a mutating action** (create/edit/delete a note or todo)
4. **Review an Action Proposal** — a structured card showing exactly what will change (field diffs, target vault)
5. **Approve or deny** the proposal inline
6. **Execute** the approved action — item is created/updated/deleted in the active vault
7. **See an audit entry** confirming what was done, when, and by whom (user vs. AI proposal)

All of this works against any registered vault the user has explicitly enabled for chat.

## MVP Scope

### In Scope
- Single-turn chat (no memory between sessions in MVP)
- NL search: query → FTS5 → render matching items as context
- Mutating actions: CreateItem, UpdateItem, DeleteItem (title + notes + priority + tags/projects/contexts + section + type)
- Action Proposal → Approval → Execute flow (always required for mutations)
- Dry-run mode: propose but never execute (per session or per config)
- Chat transcript stored in a dedicated SQLite DB (`chat.db`) in the config dir — NOT in any user vault
- Vault-scoped capability config: which vaults allow read, write, delete
- Claude API key stored in NANITE config (not env var); basic error handling if missing
- Keyboard shortcut to open/close chat panel (`Cmd+Shift+C` / `Ctrl+Shift+C`)
- Audit log table in `chat.db`: every proposal and every execution recorded

### Non-Goals for MVP
- No streaming responses to UI (full response shown when complete)
- No conversation memory / multi-turn context beyond current session
- No AI-assisted inbox triage or auto-routing
- No third-party LLM providers (Claude only, model configurable)
- No shared or cloud chat history
- No voice input
- No plugin/extension system for custom actions
- No scheduled/recurring actions triggered by chat
- No multi-vault cross-search in single query
- No export of chat transcripts

## Definition of Done (MVP)

- [ ] Chat panel opens via `Cmd+Shift+C`
- [ ] User can type a query; FTS5 search runs against active vault
- [ ] Results render as item cards in the chat response
- [ ] User can type an action request; LLM proposes a structured ActionProposal
- [ ] ActionCard renders proposal with field diffs and approve/deny buttons
- [ ] Approving executes the action via existing Wails backend methods
- [ ] Denying discards the proposal with no side effects
- [ ] Audit table records every proposal + outcome
- [ ] Dry-run mode disables execution (proposals only)
- [ ] API key missing shows graceful error message in chat
- [ ] Chat panel closeable; shortcut toggles it

## Evidence
- Created: 2026-02-22 (Inception iteration 1)
- Input: Feature scope from user, CLAUDE.md architecture, existing store/app patterns
