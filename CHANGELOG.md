# Changelog

All notable changes to this project will be documented in this file.

---

## [Unreleased]

## [1.3.0] — 2026-04

### Changed
- App renamed to **NIL** (Go module is now `github.com/hollis-labs/nil`, binary is `nil`, app bundle is `NIL.app`)
- localStorage keys now use the `nil.*` prefix — clean break, no migration; existing users will need to reconfigure tabs, sessions, templates, and saved settings on first launch
- Repository moved to `github.com/hollis-labs/nil`

### Fixed
- Resolved frontend strict-TypeScript build errors that were blocking production builds (`exactOptionalPropertyTypes`, `noUncheckedIndexedAccess`, unused vars across ~30 files)
- `internal/plugin/host.go` now defines its own app-specific types (`SlashCommandDef`, `UISlotEntry`, `KeybindingDef`) per the shared plugin package's design contract
- `go.mod` replace directive for `github.com/hollis-labs/otel` corrected to `../framework/libs/go-otel`

## [1.1.0] — 2026-01

### Added
- Escape closes EditItemModal with dirty detection; autosave preference (ask / always / never) configurable in Settings → General → Close & Save Behavior
- "→ Inbox" button in create mode for deliberate inbox routing regardless of whether the item has content
- `Shift+Shift` keyboard shortcut as alias for Quick Search
- Inline close prompt with "remember this choice" preference

### Fixed
- `UpdateItem` SQL was missing the `inbox` column — setting inbox via update was silently dropped

### Changed
- App renamed from PLANCK to NIL
- Core item type renamed from `Todo` to `Item` throughout the codebase (DB `type` column values `'todo'`/`'note'` unchanged)
- Default tabs now include an "All" tab for both Items and Notes modes
- localStorage keys migrated from `todo.*` / `planck.*` to `nil.*`
- Theme switcher hidden from Settings until Tailwind/shadcn migration is complete

---

## [1.0.0] — 2025-10 (initial alpha)

### Added
- Core item and note management with todo.txt-inspired input syntax
- Priority, projects (`+`), contexts (`@`), tags (`#`), due dates, threshold dates, recurrence
- SQLite storage with FTS5 full-text search
- Now / Soon / Anytime scope views
- Date view grouped by due date
- Custom tab filters (saved searches)
- Session context — temporary filter for focused work; new items auto-inherit session tags
- Inbox — fast capture with zero taxonomy friction
- `@reference` wikilinks with backlinks in the Notes editor
- Rich-text notes editor (TipTap / ProseMirror)
- Keyboard-driven UI with radial context menu
- Quick Search modal
- Theme system with 20+ presets
- macOS, Windows, and Linux builds via Wails v2
- Cloud sync support (iCloud Drive, Dropbox, Google Drive, OneDrive)
