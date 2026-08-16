# ADR-0004: Local HTTP API — loopback-only, shared-secret auth, alongside the Wails IPC bridge

## Status

Accepted

## Date

2026-08-16 (documenting a decision made 2026-02-22, commit `55f6d73` "Add
vault manager, HTTP API server, and CLI subcommands"; extracted to a
standalone package and given a headless entry point 2026-08-16, commit
`54fb3c9`)

## Context

The GUI already has a same-process IPC channel to the Go backend (Wails
generated bindings — see [ADR-0002](0002-wails-desktop-shell.md)). That
channel only exists inside the GUI process, though: the CLI, a future MCP
server, external scripts, and a possible future sync client are all
separate OS processes (or, in the CLI's case, short-lived one-shot
invocations) that need to read/write the same vault data without linking
against Wails or necessarily being Go at all.

## Decision

A local HTTP API (`apiserver/apiserver.go`) exists as a second,
independent surface, bound to `127.0.0.1` only, authenticated by a single
shared secret (`X-API-Key`, checked against `cfg.APIKey`). `apiserver.New
(cfg, vm) *http.Server` is constructed identically from two call sites —
embedded inside the GUI process (gated on `cfg.APIEnabled`) or run fully
headless via `nil serve-api` — so route/handler/auth logic is never
duplicated between them. Full mechanics, route surface, and the detailed
trust-boundary trade-offs (shared-secret comparison isn't constant-time,
one key for every caller, cleartext storage, wide-open CORS mitigated
only by loopback binding) are already written up in the architecture
doc's [§2.2](../architecture/ARCHITECTURE.md), [§5](../architecture/ARCHITECTURE.md),
and especially [§7 Security / trust boundaries](../architecture/ARCHITECTURE.md)
— linked here rather than re-derived.

The headless extraction itself is documented directly in its own commit
message (`54fb3c9`): "the HTTP API server (routes/handlers/auth) had no
real Wails coupling — it only depended on config/service/store/vault, all
plain Go. Extracting it to run standalone was a mechanical move, not a
restructure."

## Consequences

- CLI, `nil-mcp` (see [ADR-0005](0005-mcp-thin-http-proxy.md)), external
  scripts, and a future sync client can all reach vault data through one
  documented HTTP protocol, without any of them needing to be Go
  processes linked against `vault`/`store`.
- `nil serve-api` gives NIL a fully headless deployment story (CI,
  scripted access, a server running with no GUI at all) using the exact
  same route/handler/auth code the GUI's embedded server uses — verified
  directly by the `54fb3c9` commit's testing notes (a real `serve-api` run
  against a fresh `$HOME`, i.e. the GUI had never launched).
- The API is a second, independent trust boundary from the Wails IPC
  bridge — everything named in architecture doc §7 (single shared key, no
  per-caller authorization tier, cleartext key storage) applies to
  *every* caller of this surface, not just external ones.
- Both the embedded and headless code paths read/write the same
  `config.json` and the same vault SQLite files, so `EnsureDefaults()`
  seeds a working API port/key on first run regardless of which path
  (GUI or CLI) touches the machine first.

## Alternatives Considered

- **No separate HTTP surface — every non-GUI consumer links `vault.Manager`
  /`store.Store` directly as a Go library.** This is actually what most of
  `cli/cli.go`'s own commands already do for same-process, one-shot
  invocations (`main.go` constructs a `vault.Manager` directly before
  dispatching to `cli.Run`). It doesn't generalize to `nil-mcp` (a
  separate long-running binary/process — see ADR-0005) or to any future
  non-Go consumer, both of which need a real inter-process boundary.
- **Expose the API beyond loopback (LAN-reachable).** Not done —
  deliberately bound to `127.0.0.1` only (architecture doc §7). Would
  require fixing the non-constant-time auth comparison and adding
  per-caller credentials first.

## References

- [Architecture doc](../architecture/ARCHITECTURE.md) §2.2 "Local HTTP API", §5 "Local API — embedded vs. headless duality", §7 "Security / trust boundaries" (full trust-boundary detail — not repeated here)
