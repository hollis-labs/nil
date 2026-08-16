# ADR-0005: MCP surface — `nil-mcp` is a thin HTTP-proxying stdio server, zero direct DB dependency

## Status

Accepted

## Date

2026-08-16 (documenting a decision made 2026-04-24, commit `4f61fbb`/
`083f003` "feat: add MCP stdio server, extend HTTP API, remove plugin
system")

## Context

NIL wants to expose vault content and actions to external AI agents via
the Model Context Protocol (MCP), which speaks JSON-RPC 2.0 over stdio.
`cmd/nil-mcp` needs to read/write the same vault data as the GUI and CLI,
but it runs as its own separate binary/process — it cannot share
in-process Go objects with whatever NIL process (if any) is already
running.

## Decision

`nil-mcp` (`cmd/nil-mcp/main.go` + `mcp.go`) is a small, standalone binary
that speaks MCP over stdio and, for every one of its tools, proxies to the
local HTTP API ([ADR-0004](0004-loopback-http-api.md)) via `apiDo` —
setting `X-API-Key` and `X-Agent-Source: nil-mcp` on every request. It has
zero direct dependency on `store`, `vault`, or `service/items`; it only
knows HTTP and JSON. This was true from the very first commit that
introduced it (`4f61fbb`'s own message: "Reads NIL config for API port and
key; warns on startup if API unreachable") and remains true today across
all 15 registered tools. See the architecture doc's
[§2.4](../architecture/ARCHITECTURE.md) and
[§6](../architecture/ARCHITECTURE.md) for the full mechanics.

## Consequences

- Single source of truth: auth (`X-API-Key` check), business-logic
  defaulting (`service/items.Service` — notes resolution, search
  defaults), and validation all live in one place (the HTTP API layer,
  [ADR-0004](0004-loopback-http-api.md)). `nil-mcp` cannot drift from the
  CLI's or GUI's create/update semantics because it never reimplements
  them.
- `nil-mcp` stays tiny and trivially distributable as its own binary — no
  SQLite driver, no `vault`/`store` package compiled in.
- `nil-mcp` works transparently against either the embedded (GUI-hosted)
  or headless (`nil serve-api`) API. Commit `54fb3c9`'s message states
  this directly: "nil-mcp needs no changes — it already only reads
  config.json for port/key and proxies HTTP; works unmodified against
  either server."
- The cost is a double-hop on every tool call: JSON-RPC parse → HTTP
  request → apiserver → store, and back — versus a direct DB call.
- `nil-mcp` inherits the local API's full trust model with none of its
  own. Per the architecture doc (§6/§7): it "trusts the local API
  endpoint and API key it read from `config.json` unconditionally" and has
  "no independent authentication of its own" — whoever can run `nil-mcp`
  with read access to `config.json` has full read/write/delete access to
  every vault.

## Alternatives Considered

- **Embed `store`/`vault` directly in `nil-mcp`** (link against the same
  Go packages the GUI uses, open SQLite files itself). Would eliminate
  the double-hop latency and let `nil-mcp` run without a running API
  server, but would duplicate every bit of business logic the API layer
  already centralizes (notes-input resolution, search defaulting, auth),
  and would need its own strategy for concurrent access to vault files
  the GUI/CLI/API might also have open — rather than delegating that to
  whichever process (GUI or `nil serve-api`) is already managing it.

**Honest gap**: no commit message specifically frames this as "proxy vs.
embed" — the proxy shape was simply how `nil-mcp` was built from its first
commit. The rationale above is inferred by extension of the same
single-source-of-truth pattern `service/items.Service`'s own doc comment
names as its reason for existing (avoiding "three independent notes input
resolvers, four kind defaulters" that pre-existed and caused a real
production bug — see `service/items/items.go`'s doc comment and the
architecture doc §2.6), applied naturally to `nil-mcp` once the HTTP API
existed as a stable target to proxy through.

## References

- [Architecture doc](../architecture/ARCHITECTURE.md) §2.4 "MCP stdio server", §2.6 "The shared layer — service/items.Service", §6 "MCP/chat surfaces", §7 "Security / trust boundaries"
