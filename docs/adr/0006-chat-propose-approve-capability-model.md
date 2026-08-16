# ADR-0006: Chat permission/safety model — propose/approve flow gated by per-vault capabilities

## Status

Accepted

## Date

2026-08-16 (documenting a decision made 2026-02-22, commit `142d417`
"forge(chat): M0 foundation — chat package, bridge, action flow, Wails
bindings")

## Context

NIL's chat addon ([ADR-0001](0001-chat-bridge-direct-api.md)) lets an LLM
read vault content and suggest mutating actions (create/update/delete
items). Unlike NIL's other four surfaces (GUI, HTTP API, CLI, MCP), where
the caller is a human or a human-operated tool acting with full trust once
authenticated, chat's caller is a model whose specific output for a given
turn isn't predictable or reviewable in advance the way a typed CLI
command or a GUI button click is.

## Decision

`chat/actions.go`'s `ActionRunner` implements a **Propose → Approve/Deny →
Execute** flow:

- The model calls `Propose`, which persists an `ActionProposal`
  (`status=pending`) and appends an audit-trail entry
  (`outcome=proposed`). Nothing has touched a vault yet.
- A human must call `Approve`, which re-checks the proposal against the
  vault's configured `config.VaultCap` (`Read`/`Write`/`Delete`/
  `DirectCreate`, `config/config.go`) via `checkCaps` before executing.
  Every outcome — `dry_run`, `executed`, `failed`, or (via `Deny`)
  `denied` — is appended to an immutable `action_audit` trail.
- **One carve-out**: if a vault's `DirectCreate` capability is granted,
  `create_item` tool calls execute immediately, bypassing propose/approve
  (`bridge.go`'s `executeCreateItem`, gated on `caps.DirectCreate`) — an
  explicit, per-vault, human-configured opt-in, not a default.

`VaultCap`'s zero value is read-only (`config/config.go`'s own comment:
"Default (zero value) is read-only") — a vault with no capabilities
configured is fully read-only to the model.

## Consequences

- Default posture is human-in-the-loop for every mutating action; a vault
  the user hasn't explicitly configured stays read-only to the model.
- Every proposal outcome — including denials and failures, not just
  successful executions — is captured in an append-only audit trail
  (`chat/actions.go`'s `AppendAudit` calls on every branch of `Propose`/
  `Approve`/`Deny`), independent of whether anything actually executed.
- Capability is per-vault, so a user can grant broader trust to a
  low-stakes scratch vault while keeping a work vault locked down.
- This is the one surface whose create/update path does **not** route
  through `service/items.Service` the way the other four do — `Approve`
  calls `vaultStore.CreateItem`/`UpdateItem`/`DeleteItem` more directly.
  The architecture doc flags this explicitly as a deliberate-but-tracked
  inconsistency, not an oversight (§2.5/§2.6, "What does not go through
  this layer"): chat doesn't get the same notes-input-resolution/
  defaulting guarantees the other four surfaces share by construction.
- `DirectCreate` is an all-or-nothing bypass per vault — there's no
  finer-grained scoping (e.g. auto-approve only below some size/impact
  threshold) if a user wants partial trust rather than binary trust.

## Alternatives Considered

- **Trust the model the way the other four surfaces trust their caller**
  (execute every action immediately once the chat session/API key is
  valid). Not chosen — an LLM's output for a given turn is not equivalent
  to a human explicitly invoking a CLI command or clicking a GUI button;
  the whole premise of a chat surface is that the human describes intent
  in natural language and the model fills in structured actions on their
  behalf, which is exactly the "acts for me, but I want to see it first"
  case propose/approve exists for.
- **No capability tiers — a vault is either fully open to the model or
  fully closed.** Not chosen. `Read`/`Write`/`Delete`/`DirectCreate` as
  four independent flags lets a user grant, for example, "the model can
  read and propose writes, but must never delete without approval
  elsewhere first" postures that a single on/off switch couldn't express.

## References

- [Architecture doc](../architecture/ARCHITECTURE.md) §2.5 "AI chat bridge", §2.6 "The shared layer" (the propose/approve-vs-service-layer callout), §6 "MCP/chat surfaces", §7 "Security / trust boundaries"
- [ADR-0001](0001-chat-bridge-direct-api.md) for the transport choice this permission model sits on top of.
