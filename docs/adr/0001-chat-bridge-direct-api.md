# ADR-0001: Chat bridge calls the Anthropic API directly, not a CLI subprocess

## Status

Accepted

## Date

2026-08-16 (documenting a decision made when the chat addon was built,
2026-02-22 — commits `ed0e984` "forge(inception): install Forge, bootstrap F5
Vault Chat addon" and `142d417` "forge(chat): M0 foundation")

## Context

NIL's chat addon (F5 "Vault Chat") lets a user converse with an LLM about
their vault content and have it propose actions (create/update/delete
items). `chat/bridge.go` needed some way to call a model. Two broad shapes
were available: shell out to an external CLI tool (e.g. a `claude`-style
binary) as a subprocess and parse its output, or call the model provider's
HTTP API directly from Go.

## Decision

`chat/bridge.go` calls the Anthropic Messages API directly over HTTPS
(`claudeAPIURL = "https://api.anthropic.com/v1/messages"`, `bridge.go:23`)
using the Go standard library's `net/http` — no Anthropic SDK, no CLI
subprocess. `Bridge.Send` runs a bounded agentic tool-use loop
(`maxToolIterations = 5`) in-process: call the model, execute any `tool_use`
blocks locally via `executeTool`, feed results back, repeat. See the
[architecture doc](../architecture/ARCHITECTURE.md), §2.5,
for the full mechanics.

## Consequences

- The desktop app bundle has no dependency on an external CLI being
  installed, authenticated, or on `$PATH` on the user's machine — chat "just
  works" as part of the NIL binary once an API key is configured in
  Settings.
- NIL owns the full tool-use loop in Go: it can bound iterations
  (`maxToolIterations`), inject vault-specific context
  (`buildSystemPrompt`), and dispatch `tool_use` blocks to local handlers
  (`executeSearchVault`, `executeCreateItem`, etc.) without managing a
  subprocess's stdio framing or lifecycle.
- NIL also owns everything an SDK would otherwise provide: wire-format
  version pinning (`claudeAPIVersion` const), request/response shapes, and
  error handling are hand-rolled rather than inherited from a maintained
  client library.
- The Anthropic API key (`cfg.Chat.APIKey`) is stored and managed by NIL
  itself — cleartext in `config.json`, same exposure profile as the local
  API key (see [architecture doc](../architecture/ARCHITECTURE.md) §7)
  — rather than being handled by an external tool's own credential store.
- Chat is GUI-only today (no CLI/MCP exposure) — a direct consequence of
  `Bridge.Send`'s only caller being `App.SendChatMessage`.

## Alternatives Considered

- **Shell out to an external CLI tool as a subprocess.** Would add an
  external binary dependency, complicate cross-platform packaging
  (macOS/Windows/Linux all need the tool present, on `$PATH`, and
  authenticated), and require managing subprocess stdio framing for what is
  otherwise a small, bounded tool-use loop.
- **Use the official Anthropic Go SDK instead of raw `net/http`.** Not
  used; `bridge.go` implements the wire protocol directly.

**Honest gap**: the exact reasoning for choosing direct HTTP over a CLI
subprocess (and over the official SDK) is not explicitly recorded in this
repo's commit history — the commits that introduced the chat addon
(`ed0e984`, `142d417`) describe what was built, not why this particular
shape was chosen over the alternatives. The rationale above is inferred
from the architecture (a GUI-embedded feature needing a small, controllable
in-process tool-use loop, bundled as part of a single cross-platform
desktop binary) rather than quoted from a design discussion.

## References

- [Architecture doc](../architecture/ARCHITECTURE.md) §2.5 "AI chat bridge", §6 "MCP/chat surfaces", §7 "Security / trust boundaries"
- See [ADR-0006](0006-chat-propose-approve-capability-model.md) for the permission/safety model layered on top of this transport choice.
