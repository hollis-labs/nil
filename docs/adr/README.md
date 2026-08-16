# Architecture Decision Records

This directory records the major architectural decisions behind NIL, in
lightweight [MADR](https://adr.github.io/madr/)-style format (Status,
Context, Decision, Consequences, Alternatives Considered). Each ADR
captures a decision and its rationale — not the mechanics of how it works
today, which live in [`docs/architecture/ARCHITECTURE.md`](../architecture/ARCHITECTURE.md)
and are linked from each ADR's References section rather than duplicated.

Numbering is chronological by when this documentation set was written
(2026-08-16), not strictly by when each underlying decision was originally
made — several of these decisions predate this repo's earliest commit or
were made in earlier sessions without a written ADR at the time. Each
file's Date section notes the original decision date where known.

| # | Title | Summary |
|---|-------|---------|
| [0001](0001-chat-bridge-direct-api.md) | Chat bridge calls the Anthropic API directly, not a CLI subprocess | `chat/bridge.go` talks to Anthropic's Messages API over `net/http`, with no external CLI dependency and no Anthropic SDK. |
| [0002](0002-wails-desktop-shell.md) | Desktop shell — Wails v2, same-process IPC over generated bindings | NIL ships as a Go backend + WebView-rendered React app via Wails v2, with the GUI talking to the backend over generated in-process bindings rather than HTTP. |
| [0003](0003-vault-per-file-sqlite-single-kind-table.md) | Vault/data model — per-vault SQLite files, one kind-discriminated table | Each vault is an independent SQLite file (enabling folder-based cloud sync); todos/notes/scratch share one `kind`-discriminated table instead of separate tables. |
| [0004](0004-loopback-http-api.md) | Local HTTP API — loopback-only, shared-secret auth, alongside the Wails IPC bridge | A second surface (`apiserver/apiserver.go`), embeddable in the GUI or run headless via `nil serve-api`, gives non-GUI processes (CLI, MCP, scripts, future sync) access to vault data. |
| [0005](0005-mcp-thin-http-proxy.md) | MCP surface — `nil-mcp` is a thin HTTP-proxying stdio server | `nil-mcp` has zero direct dependency on `store`/`vault`; every tool call proxies to the local HTTP API, keeping business logic and auth centralized in one place. |
| [0006](0006-chat-propose-approve-capability-model.md) | Chat permission/safety model — propose/approve gated by per-vault capabilities | The chat addon's mutating actions default to human-in-the-loop approval, gated by per-vault `Read`/`Write`/`Delete`/`DirectCreate` capability flags — a different trust model than NIL's other four surfaces. |

## Adding a new ADR

Use the next sequential number, zero-padded to 4 digits, plus a short
kebab-case slug: `NNNN-short-slug.md`. Follow the section structure in any
existing ADR (Status / Date / Context / Decision / Consequences /
Alternatives Considered / References). If you genuinely can't determine
*why* a decision was made — only *what* was decided — say so explicitly in
Context rather than inventing a plausible-sounding rationale; several ADRs
in this set do exactly that.
