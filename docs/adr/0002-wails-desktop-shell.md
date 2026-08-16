# ADR-0002: Desktop shell — Wails v2 (Go backend + WebView-rendered React), same-process IPC over generated bindings

## Status

Accepted

## Date

2026-08-16 (documenting a decision already in place at the earliest commit
in this repo, `10589d7` "Initial commit: PLANCK todo app MVP", 2025-10-21)

## Context

NIL needed to ship as a cross-platform (macOS/Windows/Linux) desktop app
with a rich-text editor (TipTap/ProseMirror), local SQLite storage, and a
fast, keyboard-first UI — while also (later) needing to expose the same
data to a CLI, an MCP server, and a local HTTP API (see
[ADR-0004](0004-loopback-http-api.md)). The framework choice predates the
earliest commit in this repository's history: `10589d7` already ships as "a
Wails app with Go backend and React frontend."

## Decision

Build NIL on [Wails v2](https://wails.io): a single Go process owns
application logic and persistence; a native OS WebView (WKWebView on
macOS, WebView2 on Windows, WebKitGTK on Linux) renders a React 18 +
TypeScript + Vite frontend. The two communicate via same-process IPC —
Go methods bound in `main.go`'s `Bind: []interface{}{app}` are
statically analyzed by `wails generate module` into generated TypeScript
wrappers (`frontend/wailsjs/`) that call `window.go.main.App.<Method>(...)`.
This is not a network call: no port, no serialization format the app
controls, no auth boundary — see the architecture doc's
[§1 System overview](../architecture/ARCHITECTURE.md) and
[§3 Generated bindings](../architecture/ARCHITECTURE.md) for the full
mechanics (not re-derived here).

A second, deliberately separate local HTTP API surface
(`apiserver/apiserver.go`) was added later specifically so non-GUI
processes could reach the same data without going through Wails bindings
at all — that's additive, not a replacement: the GUI still uses
same-process IPC for its own calls. See
[ADR-0004](0004-loopback-http-api.md).

## Consequences

- A single static Go binary plus an OS-native WebView means no bundled
  Chromium/Node runtime — a smaller distributable and lower baseline
  memory footprint than an Electron app, and one language (Go) shared
  across the GUI backend, CLI, HTTP API, and MCP server.
- The Go backend directly owns SQLite (`modernc.org/sqlite`, pure Go, no
  CGo) — no IPC hop to a separate process is needed just to persist data.
- Same-process IPC via generated bindings means most GUI calls have
  effectively zero network/serialization overhead and — because the
  WebView and the Go process are one OS process — no meaningful auth
  boundary to defend across that link (architecture doc §7 makes this
  explicit).
- The generated-bindings approach is fragile in one specific way: nothing
  at compile time ties the TypeScript shape to the current Go struct
  shape. If a bound Go method's signature changes and nobody re-runs
  `wails generate module`, `go build`/`go test` pass cleanly while the
  frontend silently compiles against a stale contract — a runtime failure,
  not a build failure (architecture doc §3; `make codegen-check` exists
  specifically to close this gap).
- The app inherits whatever WebView quirks the host OS ships. The
  known wikilink-click-navigation bug (CLAUDE.md's Known Issues) is a
  direct consequence: WKWebView suppresses `click` events after
  ProseMirror's `mousedown` handling in a way that has resisted every
  attempted fix so far.

## Alternatives Considered

- **Electron (Chromium + Node.js).** Not chosen. The specific reasoning is
  not recorded in this repo's history — the choice predates the earliest
  commit. Inferred trade-off: Electron bundles a full Chromium + Node
  runtime per app (larger distributable, higher baseline memory), whereas
  Wails + a Go backend produces a single smaller binary and lets the app
  reuse one language across every backend surface.
- **Tauri (Rust backend + WebView).** Not chosen. No evidence in commit
  history that Tauri was evaluated. A Go backend was likely preferred
  because the rest of NIL's tooling (CLI, MCP server, HTTP API) is also
  Go — a Rust backend would have meant maintaining business logic in two
  languages or accepting a rewrite.
- **Pure web app (browser-hosted, no desktop shell).** Not chosen — NIL's
  product model (local SQLite files, cloud-sync-via-folder — see
  [ADR-0003](0003-vault-per-file-sqlite-single-kind-table.md)) assumes
  direct filesystem access a sandboxed browser tab doesn't have.
- **HTTP-only architecture for the GUI itself** (frontend talks to the Go
  backend over `localhost` HTTP instead of Wails' generated IPC bindings).
  Not chosen for the GUI's own traffic — same-process IPC avoids a port,
  a serialization boundary, and an auth story for calls that never leave
  the process. NIL did add an HTTP surface, but for a different problem
  (non-GUI processes reaching the same data): see
  [ADR-0004](0004-loopback-http-api.md).

## References

- [Architecture doc](../architecture/ARCHITECTURE.md) §1 "System overview", §3 "Generated bindings", §10 "Known hotspots / god-files" (WKWebView-specific note)
