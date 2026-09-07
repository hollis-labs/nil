# NIL frontend and data conventions

Working details that are too specific for `AGENTS.md` and not part of the system
model in [`architecture/ARCHITECTURE.md`](architecture/ARCHITECTURE.md). Every
claim below was checked against the source on 2026-09-07; file paths drift, so
treat them as "look here first."

## Path alias

`@` resolves to `frontend/src/` (`frontend/vite.config.ts:15`).

## Wails bindings

Import generated bindings as `import * as Backend from "../../wailsjs/go/main/App"`.

Prefer the typed wrappers in `frontend/src/lib/backend.ts` (`search`,
`updateItem`, `getInboxItems`) over calling `Backend.*` with an `as any` cast.
The generated bindings demand full `store.SearchRequest` / `store.Item` shapes;
the wrappers take `Partial<>` and merge defaults, so the compiler catches
field-name typos at the boundary. Passing a cast partial is how `type:` slipped
through where `kind:` was meant — the v1.3.2 silent-search bug.

## Theme system

Themes are `TermTheme` objects in `frontend/src/theme/theme.ts`;
`ThemeProvider.tsx` injects them as CSS custom properties, and `theme.css`
declares the defaults. All UI reads `var(--term-*)`. A new theme variable must be
added to the `TermTheme` type and to every preset in the same change.

The theme switcher is hidden from Settings (`SettingsModal` renders `ThemeTab`
behind `{false && ...}`) pending the Tailwind v4 + shadcn/ui migration. The theme
system itself is live and is intentional product design, not tech debt.

## localStorage keys

All keys are `nil.`-prefixed:

| Key | Holds |
|---|---|
| `nil.settings` | app settings (tabs, showCompleted, closeBehavior, …) |
| `nil.viewMode` | `scope` \| `date` |
| `nil.appMode` | `todos` \| `notes` |
| `nil.inputMode` | `search` \| `add` |
| `nil.sessionProfiles` | saved session profiles |
| `nil.activeSession` | current active session |
| `nil.taskTemplates` | saved task templates |

## Modal close flow

Modals receive open/close state and callbacks as props; they do not own their
visibility in app-level state.

`EditItemModal`'s Escape / Close / Cancel paths all call `requestClose()`. The
dirty-check and the `closeBehavior` branching (`ask` | `always` | `never`, from
`nil.settings`) live in the extracted `useDirtyClose()` hook, which drives
`ClosePromptDialog`. The dirty comparison is a deep-equal on the TipTap JSON
document, re-baselined after each save.

The "→ Inbox" button is create-mode only. It sets `forceInbox`, which becomes
`extras.inbox = true` on submit; `App.tsx`'s quick-add handlers must merge that
into the item before calling `UpdateItem`, or the flag is dropped.

## Known issues

**Wikilink click navigation (WKWebView).** `@` mention insertion, chip
rendering, `refs` sync and the backlinks section all work
(`frontend/src/lib/WikilinkExtension.tsx`,
`frontend/src/components/WikilinkSuggestion.tsx`). Clicking a chip inside the
TipTap editor does not open the referenced item. WKWebView suppresses the
`click` event once ProseMirror has handled `mousedown`; React `onClick`,
container delegation, `handleDOMEvents` and a native capture-phase listener were
all tried, and in each the `preventDefault` fires but the state-update callback
does not. Next step is attaching Safari devtools to the live WKWebView to find
where the chain breaks; the fallback is an "open" affordance rendered outside
ProseMirror's DOM. Also tracked in `ROADMAP.md`.

**Import/Export.** A todo.txt import/export surface exists — `App.ExportTodoTxt`
(`app_demo.go:161`) behind `frontend/src/components/DataTab.tsx`. It carries a
long-standing "not working" backlog note whose specifics were never recorded;
that report is unverified. Reproduce before trusting either the note or the
feature.

## TipTap in WKWebView

The editors run inside WKWebView on macOS, where DOM event handling has quirks
that do not reproduce in a browser. Test click and keyboard interactions in the
built app. `DEBUG=1 open build/bin/NIL.app` opens the WebKit inspector.
