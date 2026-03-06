---
intent: pcc_global
project: nanite
updated_at: "2026-03-04"
---

# Backlog & Priorities

## Task tracking

Nanite uses `forge.yaml` (legacy; not yet migrated to Volon). No `.volon/tasks/` directory. Priorities tracked in CLAUDE.md and ROADMAP.md.

## Immediate priorities (from CLAUDE.md)

1. **Move repo and GitHub setup** -- move to `~/Projects-apps/nanite`, set up GitHub under `hollis-labs` org
2. **User-facing docs** -- keyboard shortcuts, quick-add syntax, scope/session docs, cloud sync guide
3. **Code standardization** -- audit inline styles vs CSS variables; standardize prop patterns
4. **Stack evaluation** -- evaluate Wails v2 vs v3 vs alternative frameworks
5. **Frontend layout migration** -- Tailwind v4 + shadcn/ui while preserving terminal aesthetic
6. **Fix wikilink click** -- `@reference` chip clicks suppressed by WKWebView

## Feature roadmap

- **F1 Inbox** -- MVP shipped; remaining: saved inbox views, deterministic router, AI-assisted triage
- **F2 Advanced Search** -- negative terms, field-scoped queries, date range filters, result templates
- **F3 Addon System** (low priority) -- journal, scheduler, broadcast, contacts, calendar
- **F4 Automations & Flows** (low priority) -- AI-assisted automation builder

## Known issues

- Wikilink click navigation broken in WKWebView (ProseMirror event suppression)
- Import/Export not working (needs revisit)

## Completed recently

- Dead file cleanup, app rename (PLANCK to NANITE), item rename (Todo to Item)
- Hidden theme switcher pending Tailwind migration
- Default tab migration, escape-closes-modal behavior

## Evidence
- Last refreshed: 2026-03-04 (mentat PCC bootstrap)
- Sources: CLAUDE.md, ROADMAP.md, README.md
