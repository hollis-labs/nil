# NIL Roadmap

This document tracks the public-facing development direction for NIL.

---

## Near Term

- **Rename & cleanup** — directory names, stale files, and the app module cleaned up
- **Default tabs** — Items and Notes views each have an "All" tab by default
- **User-facing docs** — keyboard shortcut reference, Quick Add syntax guide, Scope and Session documentation, cloud sync setup guide
- **Fix wikilink click** — clicking `@reference` chips in the TipTap editor should open the referenced item (currently suppressed by WKWebView)
- **Frontend layout migration** — migrate from hand-rolled CSS custom properties to Tailwind v4 + shadcn/ui while preserving the terminal aesthetic and `--term-*` CSS variable system

---

## F1 — Inbox (Fast Capture + Triage)

**Status**: MVP shipped.

**Philosophy**: Never lose an idea to taxonomy friction. Empty or minimally-filled items are automatically flagged as inbox items and surfaced in a dedicated Inbox view rather than cluttering main views.

**Remaining**:
- Saved named inbox views (filter presets)
- Deterministic router / keyword rules → auto-assign taxonomy on capture
- AI-assisted routing and batch triage mode
- Advanced filter syntax (taxonomy, date ranges, negative terms)

---

## F2 — Advanced Search & Recall

**Goal**: Make retrieval fast and precise.

- Negative search terms, field-scoped queries, date range filters, type filters, taxonomy filters
- User-defined result templates (Markdown)
- Speed vs. depth trade-off control (fast FTS5 vs. ranked fuzzy/semantic search)

---

## F3 — Addon System *(low priority)*

A plugin architecture that allows first- and third-party addons to introduce new item types and behaviors without bloating the core app.

**Planned first-party addons**:
- **Journal** — time-stamped daily entries with prompt support
- **Scheduler** — recurring reminders and time-based triggers
- **Broadcast** — send messages/updates to a list of recipients or channels
- **Contacts** — lightweight contact records linkable to items/notes
- **Calendar** — event and deadline visualization

---

## F4 — Automations & Flows *(low priority)*

AI-assisted automation builder for creating recurring workflows: daily/weekly digests, custom newsletters, report generation, and any repeatable process that can be templated.

- Addon/plugin-based so users install only what they need
- AI helps construct and refine flows from natural language descriptions
- Pairs naturally with F3 addons (e.g., Scheduler addon triggering a Broadcast flow)
