# NIL Architecture

Canonical system model for NIL: a Wails v2 desktop app (Go backend + React/TypeScript
frontend) that manages tasks and notes as one data model. This document exists to make
the system's structure, data flow, and trust boundaries explicit, so future work
(especially the god-file split wave — CW-20260424-0026 phase tasks targeting `app.go`,
`store/store.go`, `cli/cli.go`, `chat/bridge.go`, `cmd/nil-mcp/mcp.go`,
`frontend/src/pages/App.tsx`, and the modal components) has a grounded map to work from
instead of re-deriving it.

Every claim below was checked against the code as of this writing (module
`github.com/hollis-labs/nil`, schema version 13 as of this writing — this repo is under
active concurrent development and that number changed mid-session; see §8). File:line references point at the
current commit; they will drift as the code changes — treat them as "look here first,"
not as permanently pinned coordinates.

For narrative orientation ("why does this exist," domain vocabulary) see
[`AGENTS.md`](../../AGENTS.md). For code-level conventions (how to add a Wails method,
how to add a migration) see [`CLAUDE.md`](../../CLAUDE.md). This document is the system
model those two lean on.

---

## 1. System overview

NIL ships as a single macOS/Windows/Linux desktop binary built with
[Wails v2](https://wails.io): a Go process that owns application logic and persistence,
paired with a WebView (WKWebView on macOS) rendering a React 18 + TypeScript + Vite
frontend. `main.go` is the process entry point — it either dispatches to the CLI
(`cli.Run`, see §2.3) when invoked with a bare subcommand, or calls `wails.Run(...)` to
launch the GUI (`main.go:37-90`).

In GUI mode, the frontend does **not** talk to the Go backend over HTTP. Wails embeds a
native bridge: methods on `*App` listed in `wails.Run`'s `Bind: []interface{}{app}`
(`main.go:67-69`) are introspected at build time by `wails generate module`, which emits
`frontend/wailsjs/go/main/App.js` — thin wrappers that call `window['go']['main']['App']['<Method>'](...)`
(e.g. `frontend/wailsjs/go/main/App.js:5-7`). The WebView's native side (part of the
Wails runtime, not code in this repo) intercepts those calls, marshals arguments to Go,
invokes the bound method on `*App`, and returns the (possibly async) result as a
resolved Promise on the JS side. This is same-process IPC, not a network call — no port,
no serialization format the app controls, no auth (the WebView and the Go process are
one OS process; see §7 for why this matters for trust boundaries).

A second surface — the local HTTP API (`apiserver/apiserver.go`) — exists deliberately
*outside* this Wails bridge, so that non-GUI processes (CLI, `nil-mcp`, external
scripts, a future sync client) can reach the same vault data without going through
Wails bindings at all. It is optionally embedded inside the GUI process (bound to
`127.0.0.1` only) and can also run completely standalone via `nil serve-api`
(`cli/cli.go:216-262`) with no Wails/GUI dependency whatsoever. See §2.2.

```
                     ┌─────────────────────────────────────────────┐
                     │              Desktop process (GUI)           │
                     │                                                │
  WebView (React) ───┼─→ wailsjs bindings ─→ *App (app.go) ─┐        │
  window.go.main.App │   (generated, in-process IPC)         │        │
                     │                                        ▼        │
                     │                          service/items.Service  │
                     │                          vault.Manager           │
                     │                          store.Store (SQLite)    │
                     │                                        ▲        │
                     │   apiserver.New(cfg, vaultMgr) ─────────┘        │
                     │   (optional, cfg.APIEnabled, 127.0.0.1:port)     │
                     └───────────────┬───────────────────────────────┘
                                      │ HTTP + X-API-Key
                       ┌──────────────┼───────────────┐
                       │              │               │
                 nil-mcp (stdio)   CLI (`nil serve-api`  scripts / future
                 MCP server         is a headless copy    sync clients
                                     of the same server)
```

---

## 2. Go backend surfaces

There are five distinct entry points into the same underlying data layer. Each has its
own transport and payload shape; all funnel item create/update/search logic through
`service/items.Service` (§2.6) rather than each reimplementing it.

### 2.1 Wails-bound GUI surface — `app.go`

`app.go` (923 lines) defines `type App struct` and every method Wails binds to the
frontend. `NewApp()` returns a zero-value `*App`; `startup(ctx)` (`app.go:45-78`) does
the real initialization: loads config, ensures API defaults, opens the vault manager,
optionally starts the embedded API server, and opens the chat subsystem
(`chat.Open` → `chat.db`, separate from vault SQLite files).

The struct's fields (`app.go:26-39`) are the union of everything the GUI needs live:
`vaultMgr *vault.Manager`, an optional `apiServer *http.Server` (+ mutex), and the chat
addon's `chatStore`/`chatRunner`/`chatBridge`/per-session tool caches. Every
Wails-bound method is a method on `*App`; there is no separate router — the method set
*is* the API surface, and `wails generate module` (§3) walks it directly.

Rough groupings inside `app.go` (already visually separated by `// --- Section ---`
comments, though not by file):
- Setup/config (`NeedsSetup`, `GetDatabasePath`, `SetDatabasePath`) — `app.go:120-171`
- Vault management (`GetVaults`, `CreateVault`, `RenameVault`, `DeleteVault`,
  `SwitchVault`) — `app.go:173-229`
- API config (`GetAPIConfig`, `SetAPIEnabled`) — `app.go:230-270`
- Item CRUD (`CreateItemFromLine`/`CreateNoteFromLine`/`CreateScratchFromLine` →
  shared `createFromLine`, `GetItem`, `UpdateItem`, `ToggleComplete`, `Archive`,
  `DeleteItem`, `Search`, `GetFilters`, `GetBackrefs`) — `app.go:289-433`
- Inbox (`GetInboxCount`, `GetInboxItems`, `ProcessInboxItem`) — `app.go:435-452`
- Demo data + import/export (`SeedDemoData`, `RemoveDemoData`, `ExportTodoTxt`,
  `ImportTodoTxt`, `ListKinds`) — `app.go:454-635`
- Chat addon (`StartChatSession`, `SendChatMessage`, `ApproveChatAction`,
  `DenyChatAction`, `GetChatHistory`, `GetActionAudit`, `Get/SetChatConfig`,
  chat template CRUD) — `app.go:637-923`

Notable internal detail: `UpdateItem` (`app.go:356-374`) has real business logic beyond
a thin call-through — if the incoming item has `Inbox: true` but currently lives in the
active vault, it *moves* it (create in inbox store, delete from active store) rather
than just updating in place. This is the "→ Inbox" button's server-side half; it's GUI
routing logic, not something `service/items.Service` owns (the service's `ProcessInbox`
only clears the flag, deliberately leaving vault-to-vault movement to the caller — see
`service/items/items.go:375-381`).

### 2.2 Local HTTP API — `apiserver/apiserver.go`

`apiserver.go` (870 lines) is a **standalone package with no Wails dependency** — its
own doc comment (`apiserver/apiserver.go:1-10`) states this explicitly: it depends only
on `config`, `service/items`, `store`, and `vault`. `apiserver.New(cfg *config.Config,
vm *vault.Manager) *http.Server` builds an `*http.Server` bound to `127.0.0.1:<port>`
(`apiserver.go:39-78`) using Go 1.22+'s `http.ServeMux` pattern routing
(`METHOD /path/{param}`).

Two call sites share this exact same constructor and therefore the exact same
route/handler/auth logic:
- **Embedded in the GUI**: `App.startAPIServer` (`app.go:90-106`), started from
  `startup()` only if `cfg.APIEnabled` is true.
- **Headless**: `cli/cli.go`'s `cmdServeAPI` (`cli/cli.go:216-262`), invoked via
  `nil serve-api [--port N]`. No Wails, no GUI process — just
  `apiserver.New(&cfg, env.mgr)` plus a `ListenAndServe`/signal-handling loop.

Route surface (`apiserver.go:44-69`): inbox CRUD, item CRUD (including
`POST /api/v1/items/batch` for all-or-nothing batch create), `GET /api/v1/items/ids`
(the cheap id+updated_at deletion-signal endpoint), item actions
(complete/archive), `GET /api/v1/items/{id}/backrefs`, `GET /api/v1/search`,
`GET /api/v1/taxonomy`, `GET /api/v1/vaults`. Vault selection is per-request via the
`X-Vault-ID` header (`storeForRequest`, `apiserver.go:84-98`) — absent means active
vault, `"inbox"` means the shared inbox store, anything else is looked up by ID. Auth is
a single shared-secret header check (`h.auth`, `apiserver.go:115-123`) — see §7.

### 2.3 CLI — `cli/cli.go`

`cli/cli.go` (1399 lines) implements `nil <command> [args]`. `main.go` intercepts any
bare (non-flag) first argument before `wails.Run` ever executes (`main.go:37-48`) and
calls `cli.Run(os.Args[1:], cfg, mgr)`.

The file currently has **two coexisting command-dispatch idioms** worth knowing about
for a future split:
1. A `commandRegistry []command` table built in `init()` (`cli.go:51-162`) —
   `{name, aliases, short, usage, run}` entries. `run` is
   `func(context.Context, []string, *commandEnv)`, where `commandEnv{cfg, mgr}` bundles
   the two things every command needs.
2. Older bare functions with a different signature —
   `cmdPush(ctx, args, mgr *vault.Manager)`, `cmdSearch`, `cmdInbox`, `cmdGet`,
   `cmdVaults(cfg, mgr)` (`cli.go:1120` onward) — that don't take `*commandEnv`
   directly. These are wired into the registry via small closures
   (`cli.go:113-148`) that unpack `env.mgr`/`env.cfg` and forward. `push`/`search`/
   `inbox`/`get`/`vaults` are the *legacy* command implementations (note `push`'s
   alias is literally `legacy-add`, `cli.go:109-116`); `add`, `add-batch`, `list`,
   `show`, `backrefs`, `ids`, `context`, `import`, `update`, `version`, `serve-api` are
   the newer `commandEnv`-native implementations.

`Run(args, cfg, mgr)` (`cli.go:1081-1102`) is the actual dispatcher: handles
`help`/`--version` specially, otherwise looks up the command via `findCommand` and
invokes `cmd.run`. This split (two signatures reconciled via closures) is a real seam
for a future split — the two groups could become two files without behavior change; the
registry itself and `Run`/`findCommand`/shared helpers (`parseInterspersed`, `die`,
`printJSON`, `parseIDsArg`, `chooseCreateDestination`, `selectVaultStore`, `findItem`)
would need to stay together or move to a shared internal helper file.

`nil serve-api` (§2.2) lives in this file (`cmdServeAPI`, `cli.go:216-262`) even though
it's conceptually "run the HTTP API surface" rather than "do a CLI item operation" —
worth flagging for the split since it pulls in `apiserver` as a CLI-package dependency.

### 2.4 MCP stdio server — `cmd/nil-mcp`

As of CW-20260918-0016 (2026-09-18), `cmd/nil-mcp` speaks MCP via
[`github.com/hollis-labs/go-mcp`](https://github.com/hollis-labs/go-mcp) — a thin
wrapper the portfolio shares around the official `modelcontextprotocol/go-sdk`,
targeting the 2026-07-28 spec — rather than a hand-rolled JSON-RPC 2.0
implementation. This is a wire-transport change only; ADR-0005's actual decision
("thin HTTP-proxying stdio server, zero direct DB dependency") is unchanged. The file
split is now:

- `cmd/nil-mcp/main.go` (50 lines) — bootstrap: loads config, does a best-effort
  health-check GET against the local API, builds an `apiClient` and a
  `gomcp.NewServer("nil-mcp", ...)` (`main.go:38`), registers all tools
  (`main.go:44`), and serves stdio via `srv.Run(context.Background())` (`main.go:46`).
  Builds to a separate `nil-mcp` binary (own `package main`), distinct from the `nil`
  binary.
- `cmd/nil-mcp/client.go` (122 lines) — `apiClient`, the HTTP client every tool proxies
  through: `do(ctx, method, path, body, vaultID)` (`client.go:37`) sets `X-API-Key` and
  `X-Agent-Source: nil-mcp` on every request (`client.go:52-53`). `nil-mcp` has zero
  direct dependency on `store`/`vault`/`service/items`; it only knows HTTP and JSON.
  Also holds `decodeArgs`/`decodeResult`, the untyped-map ↔ typed-struct bridge between
  go-mcp's `ToolHandler` signature and this file's argument/result shapes.
- `cmd/nil-mcp/tools.go` (605 lines) — per-tool argument structs and handler
  implementations (`toolXxx` methods on `*apiClient`), each independent, all going
  through `apiClient.do`. A handler returns the decoded API response value directly
  (not a pre-formatted string); go-mcp JSON-marshals it into both
  `CallToolResult.StructuredContent` (SEP-2106) and mirrored text content.
- `cmd/nil-mcp/tool_schemas.go` (264 lines) — pure data and registration: JSON-Schema
  property-builder helpers, the `toolAnnotations` hint table (go-mcp requires
  `ReadOnlyHint`/`DestructiveHint`/`IdempotentHint`/`OpenWorldHint` explicit on every
  tool — never inferred from a name), and `registerTools(s, c)`, which wires all 15
  tools into the `*gomcp.Server`.

**Every tool is still a thin proxy that calls the local HTTP API** — that hasn't
changed. Currently registered **15** tools: `nil_list_vaults`, `nil_search`,
`nil_get_item`, `nil_get_backrefs`, `nil_list_item_ids`, `nil_create_item`,
`nil_create_items`, `nil_update_item`, `nil_delete_item`, `nil_toggle_complete`,
`nil_archive`, `nil_list_inbox`, `nil_create_inbox`, `nil_process_inbox`,
`nil_get_taxonomy`. (AGENTS.md and `.agent-ops/project.yaml` both currently say "12" —
see §11, this is stale.)

The three-way seam this section used to recommend splitting toward — transport
framing / tool implementations / tool schema data — is now the actual file layout
above, since adopting go-mcp removed the hand-rolled framing code entirely rather than
just relocating it.

### 2.5 AI chat bridge — `chat/`

`chat/bridge.go` (914 lines) is **not** a CLI subprocess wrapper — it calls the
Anthropic Messages API directly over HTTP (`claudeAPIURL =
"https://api.anthropic.com/v1/messages"`, `bridge.go:23`) using `net/http`, no
Anthropic SDK. `Bridge.Send` (`bridge.go:335-431`) runs an agentic tool-use loop
(`maxToolIterations = 5`, `bridge.go:27`): call the model, if it emits `tool_use`
content blocks execute them locally via `executeTool` (`bridge.go:491-543`, dispatching
to `executeSearchVault`, `executeGetItem`, `executeListTaxonomy`,
`executeGetVaultStats`, `executeCreateItem`, `executeListTemplates`,
`executeUseTemplate`), feed results back, repeat. `buildTools()` (`bridge.go:163-334`)
declares the tool schema sent to the model, and `buildSystemPrompt`
(`bridge.go:840-874`) injects vault name/ID and the caller's `VaultCaps` into
`systemPromptTemplate`.

`chat` is a **separate package tree** with its own SQLite database — `chat.db`,
opened by `chat.Open(ctx, configDir)` (`chat/store.go:93-115`), independent of any
vault's `todo.db`. Its schema (`chat/store.go:15-86`) has `chat_sessions`,
`chat_messages`, `action_proposals`, `action_audit`, `chat_tool_calls`, `templates` —
none of these live in a vault's `todos` table. `chat/actions.go`'s `ActionRunner`
(`Propose`/`Approve`/`Deny`) is the human-in-the-loop gate: the model can *propose* a
mutating action, but nothing lands in a vault until `ActionRunner.Approve` runs it
through capability checks (`checkCaps`, `chat/actions.go:186`) against the caller's
`VaultCaps` — unless `DirectCreate` capability is granted, in which case `create_item`
calls execute immediately (`executeCreateItem`, `bridge.go:716-768`, gated on
`caps.DirectCreate` vs. going through the propose/approve flow). This propose/approve
split is the chat surface's own business logic — it does **not** route through
`service/items.Service.Create`/`UpdatePatch` the way the other four surfaces do; it
calls `st.CreateItem` more directly inside `executeCreateItem` and the approval path.

`chat/bridge.go`'s only Go-level entry point is `Bridge.Send`, called exclusively from
`App.SendChatMessage` (`app.go:697-792`) — the chat surface has no CLI or MCP exposure
today; it is GUI-only.

### 2.6 The shared layer — `service/items.Service`

`service/items/items.go` (456 lines) is the one place create/update/search defaulting
and notes-input resolution live, specifically so GUI, HTTP API, CLI, and (partially —
see §2.5) chat don't each reimplement it. Its own doc comment
(`service/items/items.go:1-16`) names the incident this fixed: "three independent notes
input resolvers, four kind defaulters" pre-existed and caused the v1.3.0 silent-backfill
bug (see CHANGELOG). The service is stateless (`type Service struct{}`, `New()` returns
`&Service{}`) and store-agnostic — every method takes the already-resolved
`*store.Store` the caller chose; vault routing itself stays in each consumer.

Key methods and who calls them:
- `Create` / `CreateBatch` (`items.go:130-157`) — used by `app.go` (`createFromLine`,
  `SeedDemoData`), `apiserver.go` (`handleCreateInbox`, `handleCreateItem`,
  `handleCreateItemsBatch`), `cli/cli.go` (multiple `cmdAdd*`/`cmdPush` paths).
- `Update` (`items.go:207-212`) — full-replace, used specifically by `app.go`'s
  `UpdateItem` (the GUI round-trips a complete `store.Item` through Wails bindings, so
  no patch semantics needed).
- `UpdatePatch` (`items.go:217-284`) — partial update (nil = unchanged), used by
  `apiserver.go`'s `handleUpdateItem` and CLI's `cmdUpdate`. HTTP/CLI callers don't hold
  a full `store.Item`, so this is their path.
- `Search` / `ListInbox` / `InboxCount` / `ProcessInbox` — defaulting wrappers around
  the equivalent `store.Store` methods (`applySearchDefaults`, `items.go:442-456`:
  `Kind` defaults to `"all"` — deliberately overriding the store's own `"todo"` default
  — `PageSize` to 50, `SortBy`/`SortDir` to `created_at`/`desc`).
- `ResolveNotesInput` (`items.go:293-320`) — the notes_doc > notes_md > notes_html
  precedence resolver, backed by the `ingest` package (`ingest.MarkdownToDoc`,
  `ingest.HTMLToDoc`, `ingest.DocToHTML`). Single source of truth for every external
  writer (HTTP, CLI, MCP-via-HTTP) that submits raw markdown/HTML.
- `WithText` / `WithTextSlice` / `PlainText` (`items.go:390-440`) — wrap a `store.Item`
  in `ItemView{store.Item; NotesText string}` for API/CLI/MCP responses, computing
  `notes_text` on the fly via `ingest.DocToPlainText`. This is **not** a stored column —
  it's response-time-only.

**What does *not* go through this layer**: `app.go`'s `UpdateItem` inbox-move logic
(§2.1) is GUI-specific routing that wraps `svc.Update`/`svc.Create` rather than being
inside the service itself. `chat/bridge.go`'s `executeCreateItem` mostly bypasses the
service too (see §2.5) — it's the one surface with meaningfully divergent create logic.
Everything else (search defaults, notes resolution, batch create) is shared.

---

## 3. Generated bindings (`frontend/wailsjs/`)

`frontend/wailsjs/` is produced entirely by `wails generate module` (wrapped as
`make codegen`), which statically analyzes the Go types bound in `main.go`'s
`Bind: []interface{}{app}` and everything reachable from `*App`'s method signatures.
Output:
- `frontend/wailsjs/go/main/App.d.ts` / `App.js` — one exported function per
  Wails-bound `*App` method, each a thin wrapper over `window['go']['main']['App'][...]`.
- `frontend/wailsjs/go/models.ts` (476 lines) — TypeScript class mirrors of every Go
  struct reachable from a bound method's parameters/return types (`store.Item`,
  `store.SearchRequest`, `config.Vault`, `chat.ChatSession`, `chat.ActionProposal`,
  etc., namespaced by Go package).

**Why this is fragile** (directly relevant to CW-20260424-0034, the stale-check task):
nothing at compile time ties the TypeScript shape to the current Go struct shape. If a
Go method signature or a bound struct's fields change and nobody re-runs `wails generate
module`, `go build`/`go test` pass cleanly (they only see the Go side) while the
frontend silently compiles against a stale field name or missing method — a runtime
failure, not a build failure. This has broken `make build` before without `go
build`/`go test` catching it (see CLAUDE.md's Backend conventions section). `make
codegen-check` (`Makefile:38-52`) exists specifically to close this gap in CI/local
verification: it re-runs codegen and fails if `git diff` shows any change under
`frontend/wailsjs/`.

**Boundary between hand-written and generated frontend code**: everything under
`frontend/wailsjs/` is generated — the file headers say so explicitly ("Cynhyrchwyd y
ffeil hon yn awtomatig. PEIDIWCH Â MODIWL" / "This file is automatically generated. DO
NOT EDIT", e.g. `frontend/wailsjs/go/main/App.d.ts:1-2`) and CLAUDE.md repeats the
warning. All hand-written frontend code lives under `frontend/src/`. The one
hand-written adapter layer sitting directly on top of the generated bindings is
`frontend/src/lib/backend.ts` — typed `Partial<>` wrappers around `Backend.Search` /
`Backend.GetInboxItems` / `Backend.UpdateItem` specifically so TypeScript catches
field-name typos that `as any` casts previously hid (the v1.3.2 "silent search bug"
class of bug, per its own doc comment, `backend.ts:4-10`). Most components import the
generated module directly (`import * as Backend from "../../wailsjs/go/main/App"`), so
`backend.ts` is a convenience wrapper for a subset of calls, not a mandatory
indirection layer everyone goes through.

---

## 4. Vault model

`vault.Manager` (`vault/manager.go`, 251 lines) owns every open `*store.Store`
connection for the process, plus one always-open "shared inbox" store that is not part
of the vault registry. `NewManager(ctx, cfg)` (`manager.go:25-52`) opens the inbox store
(if `cfg.InboxPath` is set) and the currently-active vault eagerly; every other
registered vault is opened lazily on first access via `StoreForID`
(`manager.go:90-110`, double-checked locking over a `map[string]*store.Store`).

Key methods:
- `ActiveStore()` / `InboxStore()` — the two stores almost every consumer needs.
- `StoreForID(vaultID)` — lazy-open-or-return-cached, used whenever a caller names a
  vault explicitly (HTTP's `X-Vault-ID`, CLI's `--vault` flag, MCP's `vault_id` tool arg
  — all three ultimately resolve through this or `ActiveStore`).
- `SwitchVault(id)` — changes `activeID` and persists `cfg.ActiveVaultID` via
  `config.Save`.
- `CreateVault` / `RenameVault` / `DeleteVault` — registry mutation; `DeleteVault`
  refuses to delete the active vault or the last remaining vault
  (`manager.go:165-187`).
- `MoveItemToVault(ctx, id, targetVaultID)` (`manager.go:191-217`) — the inbox-to-vault
  triage operation: fetch from inbox store, clear `Inbox`/default `Section`, create in
  target store, delete from inbox store. Used by `App.ProcessInboxItem`
  (`app.go:447-452`).

Each vault is an independent SQLite file/connection (`store.Open`, opened in WAL mode —
inside `store.Open`, `store/store.go:124-` — specifically so the GUI process, a standalone `nil
serve-api` process, and CLI invocations can hold live connections to the same vault
concurrently). There is **no cross-vault query** anywhere in the stack — every
search/list operation targets exactly one `*store.Store`. `apiserver.go`'s own doc
comment on `handleSearch` (`apiserver.go:747-757`) states this as a deliberate design
choice, not an oversight: a bulk-sync consumer is expected to loop over
`GET /api/v1/vaults` and issue one `/api/v1/search` call per vault.

**How vault selection threads through each surface**:
| Surface | Selector | Resolves via |
|---|---|---|
| GUI (Wails) | Implicit — `App.vaultMgr.ActiveStore()`/`InboxStore()` per call | `vault.Manager` methods called directly in `app.go` |
| HTTP API | `X-Vault-ID` header (absent = active, `"inbox"` = inbox store, else vault ID) | `apiHandler.storeForRequest` (`apiserver.go:84-98`) |
| CLI | `--vault <id>` flag (commands vary; some default to inbox, some to active) | `selectVaultStore`/`chooseCreateDestination` (`cli.go:957-994`) |
| MCP | `vault_id` tool argument, forwarded as `X-Vault-ID` | `apiDo`'s `vaultID` param (`mcp.go:410-412`) — MCP never talks to `vault.Manager` directly, only through the HTTP API |

---

## 5. Local API — embedded vs. headless duality

`apiserver.New(cfg *config.Config, vm *vault.Manager) *http.Server` (§2.2) is
constructed identically in two call sites and the resulting `*http.Server` is used
identically (`ListenAndServe`/`Shutdown`) in both — the only difference is what
process hosts it and how its lifecycle is triggered:

- **Embedded in GUI**: `App.startAPIServer(cfg)` (`app.go:90-106`) is called from
  `startup()` when `cfg.APIEnabled`, and from `SetAPIEnabled` when the user flips the
  Settings toggle at runtime (`app.go:253-270`). `App.stopAPIServer()`
  (`app.go:108-118`) is called on `shutdown()` and when the toggle is turned off. The
  server's lifetime is tied to the GUI process; if NIL.app isn't running (or the toggle
  is off), the API is unreachable.
- **Headless**: `nil serve-api` (`cmdServeAPI`, `cli.go:216-262`) constructs its own
  `vault.Manager` (via `main.go`'s CLI dispatch path, `main.go:37-48`) and runs the
  server directly, blocking on SIGINT/SIGTERM for graceful shutdown. No GUI process
  involved at all — useful for CI, scripted access, or a headless server deployment of
  the same vault data.

Both paths read/write the same `config.json` (port, API key, vault registry) and the
same vault SQLite files — `config.EnsureDefaults()` (`config/config.go:59-72`) seeds a
random port (`7765`) and a random 32-byte hex API key on first run from *either* path,
so a machine that has only ever used the CLI still gets a working API key.

---

## 6. MCP/chat surfaces

Already detailed in §2.4/§2.5; summarized here for the "surfaces" mental model:

- **`cmd/nil-mcp`**: a stdio MCP server (via `github.com/hollis-labs/go-mcp`, §2.4),
  separate binary, zero direct DB/store dependency — purely an HTTP client of the local
  API (§2.2), translating each `tools/call` into an `apiClient.do` HTTP request and
  returning the decoded response as structured tool content. It **trusts the local API
  endpoint and API key it read from `config.json` unconditionally** — no independent
  authentication of its own (see §7).
- **`chat/bridge.go`**: GUI-only, no CLI/MCP exposure. Talks directly to Anthropic's
  Messages API over HTTPS, executes a bounded (5-iteration) local tool-use loop against
  the active vault's `*store.Store`, and gates mutating actions behind a
  propose/approve flow with per-vault capability flags (`config.VaultCap`:
  `Read`/`Write`/`Delete`/`DirectCreate`). Chat conversation state lives in its own
  `chat.db`, entirely separate from any vault's `todo.db`.

---

## 7. Security / trust boundaries

**Network exposure**: the only network-facing surface is the local HTTP API
(§2.2/§5), and it is deliberately loopback-only —
`net.JoinHostPort("127.0.0.1", ...)` (`apiserver.go:72`), never `0.0.0.0`. It is not
reachable from another machine on the LAN under normal operation. Everything else (GUI
↔ Go IPC, chat’s outbound call to Anthropic, `nil-mcp`’s stdio transport) is either
in-process or an outbound-only connection initiated by this app.

**Auth model for the local API**: a single shared secret, `cfg.APIKey` (32 random
bytes, hex-encoded, generated once by `EnsureDefaults`, `config/config.go:59-72`),
checked via `X-API-Key` header equality in `apiHandler.auth`
(`apiserver.go:115-123`). This is:
- A plain `!=` string comparison, not constant-time — a timing side-channel exists in
  principle. Low real-world severity given the API is loopback-only and the attacker
  would need local code execution already, but worth naming since a future "expose
  this beyond loopback" change would need to fix this first.
- The same key for every caller — there is no per-agent/per-consumer credential.
  `X-Agent-Source` (set by `nil-mcp` to `"nil-mcp"`, `cmd/nil-mcp/client.go:53`) is an
  attribution/audit label recorded on created items (`api_source` column) — it is
  **not** an authorization mechanism. Anything holding the one API key can act as any
  "agent source" it likes by setting this header to anything.
- Stored in cleartext in `config.json` (`config.Save`, `config/config.go:207-221`,
  written `0644`) and also handed back to the frontend verbatim via
  `App.GetAPIConfig` (`app.go:239-251`) for display/copy in Settings
  (`SettingsModal.tsx`'s `apiKeyCopied` state). Anyone who can read that file, or
  screen-share/read the Settings UI, has full read/write/delete access to every vault
  via the API.
- CORS is wide open (`Access-Control-Allow-Origin: "*"`, `apiserver.go:104`). Combined
  with loopback-only binding this mostly matters for DNS-rebinding-style attacks from a
  malicious webpage open in a browser on the same machine — such a page still needs the
  API key to do anything, but a wildcard origin means the browser's Same-Origin Policy
  provides zero additional defense-in-depth here.

**What trusts what**:
- `nil-mcp` trusts `config.json` (reads the API key straight off local disk,
  `cmd/nil-mcp/main.go:18-22`) and, transitively, trusts the API endpoint it connects
  to completely — it does not verify it's talking to the genuine NIL API (e.g. no TLS,
  no pinned identity beyond "whatever is listening on 127.0.0.1:<configured port>").
  Whoever can run `nil-mcp` (i.e., has local filesystem + process-spawn access) can
  fully control any vault this machine's NIL app manages.
- The HTTP API trusts any local caller equally once it presents the one API key — there
  is no per-vault, per-operation, or per-caller authorization tier beyond the
  `X-Vault-ID` routing header (which is caller-supplied and unauthenticated beyond the
  vault existing in the registry).
- The chat bridge is the one surface with an internal authorization tier
  (`config.VaultCap`: Read/Write/Delete/DirectCreate per vault) — but that gate exists
  to protect the *user* from the *LLM* taking unwanted action, not to protect the vault
  from an untrusted process. `cfg.Chat.APIKey` (the Anthropic key) has the identical
  cleartext-in-`config.json` storage/display exposure as the local API key above.
- The Wails GUI↔backend bridge has no auth at all — by design, since it's
  intra-process IPC within a single OS process the user already launched; there is no
  meaningful boundary to authenticate across.

**Implicit assumption worth naming explicitly**: the entire trust model assumes the
local machine and local user account are the trust boundary. Nothing in this stack
defends against another local process (run by the same OS user) reading `config.json`
or a malicious webpage attempting requests against `127.0.0.1:<port>`. That's a
reasonable posture for a single-user desktop app today, but it should be treated as a
known, accepted boundary — not rediscovered as a surprise — if NIL's exposure model
ever changes (e.g. multi-user, containerized, or LAN-exposed deployment).

---

## 8. Data model

Core table: `todos` (SQLite, name deliberately unchanged from pre-kind-registry
history — see AGENTS.md). Go struct: `store.Item` (`store/models.go:3-37`). Columns of
note (full DDL in `store/schema.sql:6-28`):
- `id`, `title`, `priority` (single-char CHECK constraint), `completed`, `archived`,
  `created_at`/`updated_at` (auto-touched by the `todos_update_ts` trigger,
  `schema.sql:52-56`).
- `due_at`, `threshold_at`, `recurrence_rule` — scheduling fields.
- `kind` — discriminator (`todo`/`note`/`scratch`/registered plugin kinds), validated
  against the `kinds` registry table (`schema.sql:100-115`, `IsValidKind`/
  `validateKindOrDefaultTx`, `store/store.go:1690-`). Core kinds are seeded via
  `INSERT OR IGNORE` on every `Open()` (`schema.sql:112-115`).
- `notes_doc` — canonical TipTap/ProseMirror document JSON. `notes_html` — write-time
  render cache (`notes_html_version` invalidates it). `notes_md` — retired/deadweight
  column from the pre-v7 storage shape; not read or written by current code
  (`schema.sql`'s own comment says it will be dropped in a follow-up; CLAUDE.md's "Data
  Model" section agrees). `notes_text` is **not a column** — it's computed at response time by
  `service/items.Service.PlainText`/`WithText` (§2.6) via `ingest.DocToPlainText`, and
  separately by the store's own `derivePlainText` (`store/store.go:1676`) for FTS5
  indexing (`updateFTSTx`, `store/store.go:1657-`  — the FTS5 table
  `todos_fts(title, notes_text)` is a standalone virtual table maintained by
  application-layer calls, not SQL triggers, per `schema.sql:117-121`).
- `section` (`now`/`soon`/`anytime`), `pinned`, `inbox` (bool, auto-set on blank-title
  create or explicit "→ Inbox" routing), `api_source` (free-text writer label),
  `external_ref` (writer-supplied idempotency key; unique-per-vault via a partial
  index created post-migration in `Open()`, `store/store.go:187-191` — see that
  function's comment for why the index can't live in `schema.sql`'s unconditional
  exec).

Taxonomy: `projects`/`contexts`/`tags` tables plus join tables `todo_projects`/
`todo_contexts`/`todo_tags` (`schema.sql:58-90`), all `ON DELETE CASCADE` from
`todos`.

Inter-item references: `refs(source_id, target_id)` (`schema.sql:93-97`, `ON DELETE
CASCADE` from `todos` on both `source_id`/`target_id`), synced on every save
(`UpdateRefs`/`updateRefsTx`, `store/store.go:1109-1128`), read back via
`GetBackrefs` (`store/store.go:1129-`) — exposed identically on the GUI
(`App.GetBackrefs`), HTTP API (`GET /api/v1/items/{id}/backrefs`), and MCP
(`nil_get_backrefs`).

**Schema versioning**: `currentSchemaVersion = 13` as of this writing
(`store/store.go:54`) — this bumped from 12 to 13 *during this same work session* via a
concurrent fix (see below), which is a live illustration of how fast this number drifts;
treat it as a pointer to check, not a fact to cache. A `migration
struct{version int; sql string}` slice (`store/store.go:63-99`) plus a `schema_version`
tracking table (`runMigrations`, `store/store.go:196-`) drive upgrades. Versions 7–13
have empty `sql` fields and are instead dispatched as inline, idempotency-aware
multi-statement blocks inside `runMigrations` (`migrateV7`...`migrateV13...`,
`store/store.go:447-` through the `72x`s — note `migrateV10` is defined *after*
`migrateV12`/`migrateV13CleanupOrphanedRefs` in file order, `store.go:699`, purely a
source-ordering quirk, not a version-ordering one). There's also a fresh-install fast
path (`store/store.go:` shortly after `runMigrations` begins): if `schema_version` is
empty but the latest columns/tables already exist (i.e., `schema.sql` alone produced
the final shape), every historical migration is skipped and the version is just
stamped — avoiding redundant FTS5 rebuild churn on brand-new databases.

**Migration v13 is a live example of the exact kind of bug this document exists to make
easier to avoid**: `store.Open`'s SQLite DSN used to include `_fk=1`
(`store/store.go:151`, pre-fix) intending to turn on foreign-key enforcement, but
`modernc.org/sqlite`'s driver has never recognized that parameter name — the real
syntax is `_pragma=foreign_keys(1)`. Unrecognized DSN params are silently ignored
rather than erroring, so foreign-key enforcement (which `refs`' `ON DELETE CASCADE`
depends on) was **never actually active** in any released version. `DeleteItem`'s plain
`DELETE FROM todos` therefore never cascaded to matching `refs` rows, leaving orphans
behind (confirmed against a real vault: 11 of 15 `refs` rows were orphaned). Migration
`migrateV13CleanupOrphanedRefs` (`store/store.go:683-`) is a one-time, idempotent data
cleanup for the damage; `Open`'s DSN now correctly reads
`todo.db?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)` (`store/store.go:151`,
current). Tracked as CW-20260816-0059. If you are reading this doc later and
`currentSchemaVersion` is now higher than 13, that's expected — re-check
`store/store.go` directly rather than trusting this paragraph's number.

**Chat's separate schema**: `chat.db` (§2.5, `chat/store.go:15-86`) is entirely
independent of `store/schema.sql`'s versioning — it has its own `CREATE TABLE IF NOT
EXISTS` set with no version-tracking table at all currently.

---

## 9. Current verification entrypoints

Canonical local path: **`make verify`** = `lint test frontend-build codegen-check
build` (`Makefile:33`). Individual targets and what each actually covers:

| Target | What it runs | What it does *not* cover |
|---|---|---|
| `make test` | `go test ./...`, **and** (as of a change that landed on this branch during this same investigation — see the note below) now depends on `frontend-test`, which runs `npm --prefix frontend run test` → `vitest run` | Frontend coverage exists but is minimal: exactly one test file so far, `frontend/src/components/CopyrightFooter.test.tsx` (Vitest + Testing Library + jsdom; config in `frontend/vite.config.ts`'s `test` block, setup in `frontend/src/test/setup.ts`). None of the three hotspot components (`App.tsx`, `SettingsModal.tsx`, `EditItemModal.tsx`) have any test coverage yet. |
| `make lint` | `format-check` (gofmt/goimports) + `go-lint` (`golangci-lint run --new`) + `frontend-lint` (Biome, **only on files changed vs. `HEAD`**, `Makefile:71-77`) | `golangci-lint --new` only flags issues on changed lines — pre-existing lint debt elsewhere is not surfaced. Frontend lint is diff-scoped too, so an unrelated pre-existing frontend file is never checked by `make lint` alone. |
| `make frontend-build` | `npm run build` (Vite production build) | Type-checks and bundles, but is not a test suite — a build succeeding says nothing about runtime correctness. |
| `make codegen-check` | Re-runs `wails generate module`, fails on any diff under `frontend/wailsjs/` | Only catches drift between Go-bound signatures and generated TS; does not verify the *frontend code that consumes* those bindings still type-checks against behavior changes (that's `frontend-build`'s job, and only insofar as TS strictness catches it). |
| `make build` | `wails build` (full app + frontend bundle) | Slowest step; effectively a superset smoke-check of `frontend-build` + Go compilation together. |

> **Live-drift note**: mid-way through writing this document, a concurrent commit on
> this same branch (a) added `frontend-test`/`vitest` wiring and made `make test` depend
> on it, and (b) landed migration v13 (§8) with three new store-layer tests. This is
> called out explicitly rather than silently absorbed, both as an honest timestamp on
> this section and as a concrete demonstration of how fast "current" facts about this
> repo can go stale — re-run `find . -name '*_test.go'` and `cat Makefile` rather than
> trusting the counts below if meaningful time has passed since this was written.

**Go test coverage that exists** (`find . -name '*_test.go'`, current line counts):
`config/config_test.go` (318 lines), `ingest/ingest_test.go` (181),
`apiserver/apiserver_test.go` (637), `vault/manager_test.go` (438),
`store/store_test.go` (1069, including the three new `TestForeignKeysPragmaEnabled`/
`TestDeleteItemCascadesRefs`/`TestMigrationV13CleanupOrphanedRefs` tests locking in the
migration-v13 fix from §8), `service/items/items_test.go` (498), plus the frontend's
first test, `frontend/src/components/CopyrightFooter.test.tsx`. **No test files exist**
for `app.go` (package `main`, Wails-bound methods), `cli/cli.go`, any file in `chat/`,
`cmd/nil-mcp/` (any of its four files — §2.4), or `parse/line.go` — these are exactly the surfaces with the most
branching/business logic outside the well-tested `store`/`apiserver`/`service/items`
core, and on the frontend, coverage of the three hotspot components (`App.tsx`,
`SettingsModal.tsx`, `EditItemModal.tsx`) remains at zero. This gap is itself one of the
outstanding items in the modernization portfolio (see AGENTS.md's "Immediate
Priorities").

---

## 10. Known hotspots / god-files

Line counts below are current (`wc -l`, this session) — treat any other number
elsewhere (including in a prompt or an older doc) as unverified.

| File | Lines | Seam assessment |
|---|---|---|
| `app.go` | 923 | **Good seams exist.** Already visually grouped by `// --- Section ---` comments into: setup/config, vault management, API config, item CRUD, inbox, demo/import/export, chat addon (§2.1 lists exact line ranges). The chat addon block alone is ~290 lines (`app.go:637-923`) and has almost no coupling to the item-CRUD block beyond sharing `*App`'s fields — a strong candidate to become its own file (e.g. `app_chat.go`) with `*App` methods split across files (valid Go — one type, multiple files). Demo-data seeding (`SeedDemoData`/`RemoveDemoData`/`ExportTodoTxt`/`ImportTodoTxt`, ~170 lines including the large inline demo-content literal) is also cleanly separable. |
| `store/store.go` | 1757 (grew from 1709 to 1757 lines *during this same investigation*, via a concurrent migration-v13 commit — see §8; re-run `wc -l` before trusting this number) | **Seams exist but are entangled by the `dbtx` shared-transaction pattern.** Public API surface (`CreateItem`, `UpdateItem`, `GetItem`, `Search`, `GetBackrefs`, inbox methods, stats, taxonomy, kinds) is one clear group (`store.go:827` onward); migrations (`migrateV7`–`migrateV13...`, `runMigrations`) are a second, self-contained group (`store.go:196-` through the mid-700s) that only needs `*sql.DB` and could move to `store/migrations.go` with minimal churn. The tricky part: many `*Tx` helper functions (`createItemTx`, `updateItemTx`, `hydrateTx`, `setLinksTx`, `updateRefsTx`, `updateFTSTx`) are shared between the single-item path (via `*sql.DB` satisfying `dbtx`) and `CreateItemsBatch`'s explicit `*sql.Tx` — splitting "CRUD" from "batch" would either duplicate these helpers or require a shared internal file both import from. `stripHTML`/`derivePlainText`/FTS helpers are a third, genuinely standalone group. |
| `cli/cli.go` | 1399 | **Seam already exists, just not yet split**: the two command-dispatch idioms described in §2.3 (registry-native `commandEnv` commands vs. legacy `mgr`-only commands wired via closures) are almost mechanically separable into two files, provided the shared helpers (`parseInterspersed`, `die`, `printJSON`, `findCommand`, `parseIDsArg`, `chooseCreateDestination`/`selectVaultStore`/`findItem`) land in a third shared file or stay in whichever file `init()`/`Run` end up in. `cmdServeAPI` (§2.3) is arguably better relocated near `apiserver` usage than kept in the CLI's item-command file. |
| `chat/bridge.go` | 914 | **Reasonable internal seams, currently one file.** `buildTools()` (schema declarations, ~170 lines, pure data) vs. `Send`/`callAPI` (the agentic loop + HTTP transport to Anthropic, ~100 lines) vs. the `executeXxx` tool-implementation methods (~230 lines) vs. `buildSystemPrompt`/`extractAction` (prompt templating + action-proposal extraction from model output, ~120 lines) are four fairly distinct concerns with narrow interfaces between them (`BridgeRequest` in, `ChatResponse` out). `chat/` as a package already has `actions.go`/`models.go`/`profile.go`/`store.go` alongside `bridge.go`, so this would be more "further split bridge.go along its existing internal boundaries" than "restructure the package." |
| ~~`cmd/nil-mcp/mcp.go`~~ | — | **Resolved, not a current hotspot.** This row previously tracked a single 1159-line file with a clean three-way seam (framing / tool implementations / schema data). CW-20260918-0016 (2026-09-18) adopted `github.com/hollis-labs/go-mcp` for the transport, which removed the hand-rolled JSON-RPC framing entirely (not relocated — deleted) and left the remaining tool logic split across `client.go` (122 lines), `tools.go` (605 lines), and `tool_schemas.go` (264 lines) — see §2.4. |
| `frontend/src/pages/App.tsx` | 1406 | **Weak seams — this is the hardest split.** Almost the entire file is one function, `Inner()` (`App.tsx:34` through just before `App.tsx:1398`), holding ~30 `useState` hooks (`App.tsx:35-63`) and a dozen-plus `useEffect`s, wrapping a large `Inner`-local set of `async function handleXxx` handlers (`handleToggle`, `handleMoveSection`, `handleArchive`, `handleRefClick`, `handleSaveNotes`, `handleQuickAdd`, `handleQuickAddNote`, `handleUpdateItem`, `handleUpdateItemStay`, `handleCloneItem`, `handleConvertType`, `handleMetaSave`, `handlePin`, `handleInputSubmit`, `handleDeleteTodo` — `App.tsx:437-690+`) that close over most of that state directly. `AppPage` (`App.tsx:1398`) is just a thin wrapper providing context/providers around `Inner`. Any split has to either (a) introduce a reducer/context to break the closure coupling first, or (b) extract along handler *groups* that touch disjoint state slices (e.g. modal-open-state handlers vs. item-mutation handlers vs. session/vault-switch handlers) and accept prop-drilling the touched state back in. There is no cheap, behavior-preserving seam here the way there is in the Go files — CLAUDE.md's own warning ("This file is large; look before adding new state or handlers") is accurate and this file needs a real design pass, not a mechanical split. |
| `frontend/src/components/SettingsModal.tsx` | 1644 | **Good seam: already tab-partitioned.** `activeTab` (`type SettingsTab = 'general' \| 'tabs' \| 'data' \| 'vaults' \| 'chat'`, `SettingsModal.tsx:27`) gates five large, largely-disjoint JSX blocks: `general` (~405–692), `tabs` (~692–803), `vaults` (~803–997), `chat` (~997–1433), `data` (~1433–end). Each block is a strong candidate for its own component (`GeneralTab`, `TabsTab`, `VaultsTab`, `ChatTab`, `DataTab`), taking `local`/`setLocal` (settings draft state) and tab-specific state slices as props. The file also exports a `SettingsProvider`/`useSettings` context (`SettingsModal.tsx:56-107`) that is logically a separate concern (global settings context) from the modal UI itself and could move to its own module independent of the tab split. |
| `frontend/src/components/EditItemModal.tsx` | 1368 | **Moderate seam, but tightly closure-coupled like App.tsx.** ~20 `useState` hooks (`EditItemModal.tsx:43-63`) plus named handlers (`isDirty`, `doSave`, `doSaveStay`, `requestClose`, `handleSubmit`, `handleClear`, `handleTemplateSelect`, `handleContextProfileSelect`, `EditItemModal.tsx:273-430`) occupy the first third of the file; the remaining ~900 lines (`EditItemModal.tsx:484` to EOF) are a single `return (...)` JSX tree with inline conditional sections (fullscreen toggle header, title/priority/due inputs, taxonomy autocompletes, session-profile picker, TipTap `EditorContent`, inline delete-confirm bar, inline close-prompt dialog, and the external `<TemplateSaveDialog>`). The inline delete-confirm and close-prompt blocks are the cleanest extraction candidates (self-contained conditionals with narrow prop needs); the taxonomy/editor core is more entangled with `line`/`priority`/`tags`/`contexts`/`projects` state and the TipTap `editor` instance, so extracting it means threading that state through props rather than a free lift. |

---

## 11. Known doc drift vs. AGENTS.md / project metadata

Two discrepancies were found between AGENTS.md/`.agent-ops/project.yaml` and the actual
code while writing this document. Neither was fixed here — this document reports them;
fixing AGENTS.md/`.agent-ops/project.yaml` is a separate decision for whoever owns those
files.

1. **`AGENTS.md:32`** lists `api.go` as the HTTP API surface's entry point. No file
   named `api.go` exists anywhere in this repository — the HTTP API lives at
   `apiserver/apiserver.go` (package `apiserver`), and has since before this document
   was written. `CLAUDE.md`'s own "Project Layout" section already correctly names
   `apiserver/apiserver.go` — only `AGENTS.md`'s "Where to start" section has the stale
   `api.go` reference.
2. **`AGENTS.md`'s "(c) Key domain concepts"** and **`.agent-ops/project.yaml`'s**
   `usage_examples` both state `cmd/nil-mcp` exposes **12** tools. The actual count, per
   `registerTools` in `cmd/nil-mcp/tool_schemas.go` (§2.4), is **15**:
   `nil_list_vaults`, `nil_search`, `nil_get_item`, `nil_get_backrefs`,
   `nil_list_item_ids`, `nil_create_item`, `nil_create_items`, `nil_update_item`,
   `nil_delete_item`, `nil_toggle_complete`, `nil_archive`, `nil_list_inbox`,
   `nil_create_inbox`, `nil_process_inbox`, `nil_get_taxonomy`.

A third apparent discrepancy self-corrected mid-session and is *not* live drift as of
this writing: `CLAUDE.md`'s "Current schema version" line said **12** when this document
was started, and a concurrent commit on this same branch bumped both the actual schema
(`currentSchemaVersion = 13`, §8) and `CLAUDE.md`'s claim to match, within the same
working session. It's called out here only as evidence of how quickly single-number
claims like this go stale in an actively-developed repo — worth remembering when
trusting *any* specific version/count/line-number claim, including the ones in this
document, without a fresh check.

---

## 12. Architectural Decision Records

The *why* behind five of this document's major structural choices —
desktop-shell direction, the vault/data model, the local API model, the
MCP surface, and the chat permission/safety model — plus the chat
bridge's direct-API transport choice, is recorded as ADRs rather than
re-derived here. See [`docs/adr/README.md`](../adr/README.md) for the
full index:

- [ADR-0001](../adr/0001-chat-bridge-direct-api.md) — Chat bridge calls the Anthropic API directly, not a CLI subprocess
- [ADR-0002](../adr/0002-wails-desktop-shell.md) — Desktop shell: Wails v2, same-process IPC over generated bindings
- [ADR-0003](../adr/0003-vault-per-file-sqlite-single-kind-table.md) — Vault/data model: per-vault SQLite files, one kind-discriminated table
- [ADR-0004](../adr/0004-loopback-http-api.md) — Local HTTP API: loopback-only, shared-secret auth, alongside the Wails IPC bridge
- [ADR-0005](../adr/0005-mcp-thin-http-proxy.md) — MCP surface: `nil-mcp` is a thin HTTP-proxying stdio server
- [ADR-0006](../adr/0006-chat-propose-approve-capability-model.md) — Chat permission/safety model: propose/approve gated by per-vault capabilities

---

## Cross-references

- Narrative/domain orientation: [`AGENTS.md`](../../AGENTS.md)
- Code conventions, schema-change checklist, roadmap: [`CLAUDE.md`](../../CLAUDE.md)
- CLI reference: [`docs/CLI.md`](../CLI.md)
- Distribution/build specifics: [`docs/DISTRIBUTION.md`](../DISTRIBUTION.md)
- Architectural Decision Records: [`docs/adr/README.md`](../adr/README.md)
- Live task tracking: Torque project `PRJ-20260417-0005`
