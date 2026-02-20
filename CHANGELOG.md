# Changelog

All notable changes to this project will be documented in this file.

---

## [Unreleased]

## [1.1.0] — 2026-01

### Added
- Escape closes EditItemModal with dirty detection; autosave preference (ask / always / never) configurable in Settings → General → Close & Save Behavior
- "→ Inbox" button in create mode for deliberate inbox routing regardless of whether the item has content
- `Shift+Shift` keyboard shortcut as alias for Quick Search
- Inline close prompt with "remember this choice" preference

### Fixed
- `UpdateItem` SQL was missing the `inbox` column — setting inbox via update was silently dropped

### Changed
- App renamed from PLANCK to NANITE
- Core item type renamed from `Todo` to `Item` throughout the codebase (DB `type` column values `'todo'`/`'note'` unchanged)
- Default tabs now include an "All" tab for both Items and Notes modes
- localStorage keys migrated from `todo.*` / `planck.*` to `nanite.*`
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
