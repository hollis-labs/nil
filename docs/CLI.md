# NIL CLI — Current Reality & Roadmap

> Updated 2026-02-22

This document captures how the current `nil` command-line experience works today and outlines a plan to make it first-class for both humans and autonomous agents. Source of truth for the implementation is `cli/cli.go`, which intercepts `nil <subcommand>` invocations before the Wails GUI boots (`main.go`).

## Current Architecture

### Command framework options
- **Keep `flag.FlagSet` (status quo)** — zero dependencies, small binary, but requires hand-built help/usage strings and manual parsing for nested commands. Good for minimal CLIs but tedious as surface grows.
- **`spf13/cobra`** — de facto Go standard for complex CLIs. Pros: automatic help/usage trees, persistent/global flags, tab completion generators, active community. Cons: heavier dependency, more boilerplate structs per command. Best when we expect dozens of commands and need discoverability.
- **`urfave/cli/v3`** — lightweight alternative with context objects, before/after hooks, and built-in subcommand parsing. Slightly simpler than Cobra but less batteries included (no auto docs/completions).
- **Custom hybrid** — keep `flag` but build a thin dispatcher + help generator that reads metadata per command (name, description, flags). Lowest churn vs rewriting everything, but we still own UX concerns.

**Decision:** start with the custom hybrid so we can stub new verbs immediately without pulling in Cobra. Once the command tree/UX matures, migrate to Cobra for auto-generated help/completions.

### Entry flow
- `main.go:16-38` checks `os.Args[1]` for a bare word; if present, it loads config (`config.LoadOrDefault()`), opens vaults via `vault.NewManager`, and calls `cli.Run`.
- `cli.Run` dispatches on the first argument and enforces `nil <push|search|inbox|get|vaults>` usage. Errors call `die()` which prints to `stderr` and exits non-zero.
- Every subcommand writes a JSON envelope (`{"ok": true|false, "data": ..., "error": ...}`) via `printJSON`. There is no text/table formatter.

### Data dependencies
- `vault.Manager` exposes three store handles: `ActiveStore()` (current vault), `InboxStore()` (shared inbox DB), and `StoreForID()` (lazy-open any vault directory from `config.Config.Vaults`).
- `store.Store` encapsulates SQLite CRUD, search, inbox filters, refs, taxonomy, and migrations (current schema version: 6). CLI commands operate directly on this layer rather than going through HTTP APIs.
- Config (`config/config.go`) lists vault metadata plus API/chat toggles. CLI trusts whatever is on disk; there is no CLI-side auth or session concept.

### CLI-only utilities
- `parseInterspersed` lets users mix flags and positional tokens freely (e.g., `nil push "Title" --tags focus`).
- `splitCSV` splits comma-separated taxonomies; blank strings return `nil`, so JSON omits empty arrays.

## Current Commands & Usage

| Command | Purpose | Key Flags / Notes | Output |
|---------|---------|-------------------|--------|
| `nil add [title]` | Create todos/notes from inline text or `--file` input. | `--file`, `--body`, `--type`, `--vault`, `--inbox`, `--tags`, `--contexts`, `--projects`, `--due`, `--priority`, `--meta`. Files must be UTF-8 text/Markdown. | JSON envelope with created item. |
| `nil import --dir PATH` | Bulk-ingest text/Markdown files from a directory (recursive). | `--dry-run`, `--type`, `--vault`, `--inbox`, `--tags`, `--contexts`, `--projects`, `--meta`. | JSON summary of processed paths and errors. |
| `nil list [view]` | Built-in views (`latest`, `inbox`, `today`, `overdue`, `high`, `notes`, `todos`). | `--view`, `--vault`, `--limit`, `--query`, `--include-inbox`. | JSON envelope containing items array. |
| `nil show <id>` | Pretty-print a single item. | `--vault id|inbox`, `--format detail|json`. | Detail envelope or raw JSON. |
| `nil update --ids ...` | Batch mutate items (complete/archive/delete/retag). | `--ids`, `--complete`, `--uncomplete`, `--archive`, `--unarchive`, `--delete`, `--priority`, `--tags-add`, `--tags-remove`, `--vault`, `--inbox`. | JSON summary of updated vs failed IDs. |
| `nil context [topic]` | Emit agent-friendly context (app, schema, stats, cache). | `--topic app|schema|stats|cache|refresh`, `--refresh`. | JSON snapshot data; `cache --refresh` rebuilds `${config}/context-cache/snapshot.json`. |
| `nil search <keywords>` | Full-text/filtered search over the active (or specified) vault. | `--vault`, `--type`, `--tags`, `--contexts`, `--projects`, `--page`, `--page-size`. | JSON array of matching items. |
| `nil inbox [keywords]` | Keyword-filtered listing of inbox items housed in the shared inbox store. | `--page`, `--page-size`. | JSON array of inbox items. |
| `nil vaults` | Dump config metadata so agents can learn vault IDs and the active vault. | _None_. | JSON object: `{"active":"<id>","vaults":[...]}`. |
| `nil push <title>` | **Legacy alias** for add that expects TipTap HTML via `--notes`. Prefer `nil add`. | Same flags as before; no new behavior planned. | JSON document of the created item. |

### Dispatcher & helper commands

- `nil help [command]` shows usage derived from the registry; `nil version` prints Go/platform info.
- Commands emit `{"ok": bool, "data": ..., "error": ...}` envelopes so agents can parse results consistently.
- For automation, add a step like `nil context cache --refresh >/dev/null` to bootstrap scripts so every workspace starts with a fresh snapshot.

### Notable behaviors
- `nil add` stores body text verbatim (inline or file). Rich TipTap conversion remains a backlog item, so no markdown-to-HTML conversion occurs yet.
- Imports/add only accept UTF-8 text or Markdown; binary files or other encodings throw explicit errors.
- Legacy verbs (`push`, `get`) still exist, but new flows (`add`, `list`, `show`, `update`, `context`, `import`) should be used by default.

## Gaps Relative to First-Class CLI Goals

1. **User ergonomics** — Commands remain JSON-first. Table/plaintext formats and nicer error presentation are still backlog work.
2. **Rich conversion** — add/import still store raw text; the shared TipTap converter (BACKLOG-20260222-008) must land before auto-converting files or inline markdown.
3. **Historical/per-entity context** — the cache captures only the latest snapshot. Historical and per-entity caches are tracked in BACKLOG-20260222-009/010.

## Proposed CLI Roadmap

The following plan turns the CLI into a first-class interface for both humans and agents while meeting the specific goals listed in the task description.

### 1. Foundations & Experience (Goal #1)
1. **Command framework cleanup:** Introduce a proper subcommand/flag UX (either keep the lightweight `flag.FlagSet` with a shared `--help` renderer or migrate to `spf13/cobra`). Ensure `nil help`, `nil <command> --help`, and `nil version` exist.
2. **Output modes:** Keep the JSON envelope as the default for agents but add `--format table|json|ids` (per command or global) so humans can read tabular output. Provide colorized stderr for errors when running interactively.
3. **Consistent naming:** Rename `push` → `add`, `get` → `view`, and group list commands under `nil list <view>` for clarity. Keep legacy aliases temporarily with deprecation warnings.
4. **Authentication for future remote use:** Accept an optional `--api-key`/`--api-url` pair (defaulting to local store) so agents can talk to remote Nil instances with the same CLI syntax.

### 2. File/Dir Imports (Goals #2 & #5)
1. **`nil add --file` / `--dir`:** Allow specifying `--file path/to.md` or `--dir path/to/folder`. Single files become individual notes or todos; directories expand recursively and import every supported file as separate notes. MVP accepts plain text + markdown only; if the file is not UTF-8 text, the command should fail with a clear error (current implementation).
2. **Metadata flags:** Reuse existing taxonomy flags plus new `--meta key=value` (repeatable), `--vault <id>`, `--inbox`, `--type todo|note`, and `--title <string>` to override filename-derived titles.
3. **Processing rules:** Treat file contents as raw text/markdown and store exactly what was supplied. Once the shared HTML conversion/intelligence pipeline ships elsewhere in Nil, revisit and auto-convert in the CLI (tracked as a backlog item).
4. **Bulk import UX:** Add `nil import --dir ... [--dry-run] [--batch-size N]` that emits a summary (counts, errors, preview of generated metadata) before committing. MVP could simply be a wrapper around repeated `add --file` calls but records a single envelope containing successes/failures.
5. **Context caches:** Agent-centric snapshots now live under `${XDG_CONFIG_HOME}/nil/context-cache/snapshot.json`. Run `nil context cache --refresh` to regenerate after data changes; the CLI will refresh automatically if the cache is missing. A future backlog item still tracks the shared text→TipTap converter so imports can gain richer processing.

### 3. View & Action Commands (Goal #3)
1. **`nil list <view>`:** Built-in views like `latest`, `inbox`, `today`, `overdue`, `high`, `note`, `todo`, `focus` (session), each mapping to a `store.SearchRequest`. Provide optional `--vault`, `--limit`, `--json`.
2. **`nil show <id>`:** Pretty-print a single item, including metadata, wikilinks, and raw JSON via `--raw`.
3. **`nil add` ergonomics:** Support inline quick-add syntax (e.g., `(A) Call mom +family due:2026-02-23`) in addition to structured flags, matching the GUI parser.
4. **Batch operations:** `nil update --ids 1,2,3 --complete`, `nil move --ids ... --vault <id>`, `nil delete --ids ...`, `nil tag --ids ... --add #focus --remove #someday`. Ensure every batch command has a `--json` summary that agents can parse.
5. **Saved queries:** `nil views` (list) plus `nil view save <name> --query ...` so agents can orchestrate workflows by referencing canonical filters.

### 4. Agent Context Commands (Goal #4)
1. **`nil context app`:** Emits a structured JSON summary describing what Nil is, active features, CLI/API entry points, and references to documentation (drawn from existing README + PCC data).
2. **`nil context schema`:** Returns the current schema version, table definitions, migrations applied, and enumerations like allowed item types or sections.
3. **`nil context stats`:** Summaries such as item counts per type, inbox size, most-used tags, and last sync time so agents can understand the data landscape.
4. **`nil context cache`:** Streams the cached snapshot (generating one automatically when absent). Use `nil context cache --refresh` to rebuild the snapshot before returning data so agents always work from a consistent view.
5. **Documentation sync:** Generate markdown (possibly `docs/CLI.md` excerpts) via `nil docs cli` command so that both humans and agents can request up-to-date instructions through the CLI itself.

### 5. Implementation & Rollout Plan
1. **Phase 1 (Docs + Refactor):** Publish CLI docs (this file), wire up `nil help`, rename commands with aliases, and add JSON schema tests to lock down current behavior.
2. **Phase 2 (Add/List Enhancements):** Build the quick-add parser integration, add view shortcuts, and support `--format`.
3. **Phase 3 (Imports & Batch Ops):** Implement file/dir ingestion plus batch update/move/delete commands. Begin exposing `nil import` (dir flag MVP) with incremental improvements.
4. **Phase 4 (Agent Context):** Implement `nil context <topic>` commands, design the context-cache storage, and provide refresh/generate verbs. Document how GUI and chat agent will call into these commands internally.
5. **Phase 5 (Polish & Internal Adoption):** Ensure GUI/Chat pipelines call into the CLI, add integration tests around agent flows, and deprecate old command names once GUI migration is complete.

## Backlog / Follow-Ups
- Build the text/HTML conversion utility shared by GUI + CLI to support file imports (currently missing).
- Design a richer "agent context cache" schema (beyond the current snapshot) if future workflows require separate files or historical rollups.
- Evaluate whether commands should default to the inbox or active vault when neither `--inbox` nor `--vault` is specified for imports.
- Determine how to authenticate when CLI commands are invoked remotely by agents (vs local store access).

This roadmap keeps CLI parity with future GUI/agent tooling and ensures that every GUI action can be represented 1:1 through stable commands and flags.
