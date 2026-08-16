# ADR-0003: Vault/data model — per-vault SQLite files, one kind-discriminated table

## Status

Accepted

## Date

2026-08-16 (documenting decisions already in place by v1.0.0, 2025-10, and
extended by the vault registry added 2026-02-22, commit `55f6d73`)

## Context

This ADR covers two related data-model decisions that both predate any
recorded alternatives-evaluation in this repo's history, so both are
grounded in what shipped plus the product goals documented elsewhere
(README.md, AGENTS.md, CHANGELOG.md) rather than a design discussion.

NIL's v1.0.0 feature set already included "Cloud sync support (iCloud
Drive, Dropbox, Google Drive, OneDrive)" (CHANGELOG.md), and README.md
describes it as "store your database in iCloud Drive, Dropbox, or any
folder." Separately, NIL's product thesis from the outset is that "tasks
and notes are the *same object* with different affordances" (AGENTS.md,
"(a) What is this, and why") — the closest competitive reference named is
Logseq, task-first, not two separate task-app/note-app data models bolted
together.

## Decision

1. **Per-vault SQLite files.** Each vault is an independent SQLite
   file/connection, opened in WAL mode and owned by `vault.Manager`
   (`vault/manager.go`). A vault is just a directory path a user points
   NIL at — including a cloud-sync folder. There is no cross-vault query
   anywhere in the stack; every search/list operation targets exactly one
   `*store.Store` (confirmed as a deliberate choice, not an oversight, by
   `apiserver.go`'s own doc comment on `handleSearch`). See the
   architecture doc's [§4 Vault model](../architecture/ARCHITECTURE.md).
2. **One kind-discriminated table.** Todos, notes, and any other
   registered kind (e.g. `scratch`) live in a single `todos` table (name
   preserved from pre-kind-registry history), discriminated by a `kind`
   column validated against a `kinds` registry table (schema v9–v10) —
   rather than separate `todos`/`notes` tables. See the architecture doc's
   [§8 Data model](../architecture/ARCHITECTURE.md).

## Consequences

**Per-vault files:**
- Each vault is trivially portable and backupable (one file/folder) and
  works directly with folder-based cloud-sync tools (iCloud Drive,
  Dropbox, etc.) with no sync protocol of NIL's own to build or maintain.
- Multiple vaults let a user segment contexts (work/personal) with actual
  SQLite-level isolation, not just an app-level filter.
- SQLite's WAL mode lets the GUI process, a standalone `nil serve-api`
  process, and CLI invocations hold live, concurrent connections to the
  same vault file.
- The cost: no cross-vault search. A bulk-sync consumer must loop over
  `GET /api/v1/vaults` and issue one search per vault (architecture doc
  §4).
- Concurrent-write correctness across multiple processes touching one
  vault file has real sharp edges — the migration-v13 `busy_timeout`
  DSN-parameter bug (architecture doc §8) only surfaced under genuine
  multi-process concurrent writes to a single vault.

**Single kind table:**
- One taxonomy system (projects/contexts/tags), one FTS5 index, one
  CRUD/search code path serves todos/notes/scratch/future kinds — the
  shared `service/items.Service` layer only has to reason about one
  `Item` shape, and adding a new kind is a registry entry, not a new
  table plus duplicated CRUD.
- The cost: the table carries columns that are semantically irrelevant to
  some kinds (e.g. `due_at`/`recurrence_rule` don't really apply to a
  `note`), and the schema can't enforce kind-specific constraints at the
  DB level — that's left to application logic.
- The table name itself (`todos`) is now permanently misleading, since it
  holds notes and other kinds too. Renaming it is explicitly deferred
  (AGENTS.md, CLAUDE.md) — it would require a data migration with no
  user-visible benefit.

## Alternatives Considered

- **Single shared database, vaults as rows tagged with a `vault_id`
  column.** Not chosen. Would centralize storage but break the
  "point NIL at a cloud-synced folder" portability model, and would make
  per-vault operations like `DeleteVault` (registry removal without
  touching other vaults' data) less clean.
- **Separate `todos`/`notes` (and future `scratch`) tables.** Not chosen.
  Would require the shared service layer, search, and taxonomy joins to
  be either duplicated per table or generalized across them — directly
  against the "same object, different affordances" product thesis
  (AGENTS.md).

**Honest gap**: no commit or design doc in this repo explicitly discusses
*why not* a single shared database (with vault_id) was rejected — that
reasoning is inferred from the cloud-sync feature (README.md/CHANGELOG.md)
and the vault registry's per-vault `Path` field (`config.Vault`,
`vault/manager.go`'s `openVaultByID`), not quoted from a recorded
decision.

## References

- [Architecture doc](../architecture/ARCHITECTURE.md) §4 "Vault model", §8 "Data model"
