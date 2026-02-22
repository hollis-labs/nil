---
intent: project_doc
class: prd_lite
feature: F5 — Vault Chat
created_at: 2026-02-22
iteration: 1
---

# PRD-lite — F5: Vault Chat

## Problem

NANITE users accumulate many items across vaults but retrieval requires knowing exact syntax (FTS5 queries, tag filters). There's no way to ask "what was I working on last week for project X?" without manually constructing filters. Mutations require navigating to the correct item and using the edit modal.

## Users

Single-user desktop app. The "user" is always the app owner — there is no auth or multi-user concept. The chat interface is for the local user only.

## User Stories

### Search
- **US-001**: As a user, I want to type a natural-language query in a chat input and receive matching vault items, so I can find items without knowing FTS5 syntax.
- **US-002**: As a user, I want search results to show item title, section, tags, priority, and a snippet of the notes, so I can identify the right item without opening it.
- **US-003**: As a user, I want the AI to translate my query into a structured `SearchRequest` and show me what filters it applied, so I can understand why I got those results.

### Mutating Actions
- **US-004**: As a user, I want to say "create a todo to review the Q1 report, priority A, +work" and see an ActionProposal before anything is saved, so I can verify the AI understood me correctly.
- **US-005**: As a user, I want to say "mark the 'buy tickets' todo as complete" and see an ActionProposal showing the exact item it found and the field it will change, before approving.
- **US-006**: As a user, I want to say "delete the note about the old vendor" and have the AI show me the specific note it matched with a confirm-before-delete proposal.
- **US-007**: As a user, I want to approve an ActionProposal with a single click/keypress, so the action runs without extra navigation.
- **US-008**: As a user, I want to deny an ActionProposal and optionally type a correction, so I can redirect the AI without losing context.

### Safety & Audit
- **US-009**: As a user, I want a dry-run mode where the AI proposes actions but never executes them, so I can safely explore the feature.
- **US-010**: As a user, I want to see an audit log of all proposed and executed actions (timestamp, action type, item affected, outcome), so I have a reliable record.
- **US-011**: As a user, I want to configure which vaults chat can write to (and which are read-only), so I don't accidentally mutate a vault I want to protect.

### Configuration
- **US-012**: As a user, I want to enter my Claude API key in NANITE Settings and have chat use it, without touching environment variables.
- **US-013**: As a user, I want to choose which Claude model chat uses (e.g. Haiku vs Sonnet), for cost/quality trade-off.

## Constraints

1. **No vault pollution**: Chat transcripts must NOT be stored in any user vault. Use a separate `chat.db` in the config directory.
2. **Approval required**: Every mutation must go through ActionProposal → Approval before execution. No "fast path" bypass.
3. **Vault capability config**: Each vault has explicit `read|write|delete` capability flags for chat. Default: read-only.
4. **No external network calls** except Claude API (no telemetry, no phoning home).
5. **API key security**: Key stored in NANITE config (same location as `apiKey` today); never logged; redacted in run logs.
6. **Existing patterns**: Chat backend must use existing `store.Store` methods — no direct DB queries in chat code.
7. **Wails bindings**: New Go methods must be added to `app.go` and bindings regenerated with `wails generate module`.
8. **Styling**: Chat UI uses `var(--term-*)` CSS variables — no hardcoded colors, no Tailwind (not yet installed).

## Acceptance Criteria

| # | Criterion |
|---|---|
| AC-1 | Chat opens on `Cmd+Shift+C`; closes on same or Escape |
| AC-2 | NL query returns matching items via FTS5 within 500ms on typical vault |
| AC-3 | Every mutating request produces an ActionProposal; no direct execution |
| AC-4 | Approve button executes action; deny button discards; both update audit log |
| AC-5 | Dry-run mode prevents all execution; proposals still shown |
| AC-6 | Audit log persists in `chat.db`; readable from Settings or debug view |
| AC-7 | Missing API key shows error message in chat; no crash |
| AC-8 | Vault write capability defaults to off; user must explicitly enable per vault |
| AC-9 | Chat DB lives in config dir, never in user vault dir |

## Evidence
- Created: 2026-02-22 (Inception iteration 1)
