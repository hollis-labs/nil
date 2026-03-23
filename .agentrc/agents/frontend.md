# Frontend Context — NANITE

> Project-specific frontend conventions. Loaded by the frontend agent role when working in this project.
> Lives at `nanite/.agentrc/frontend.md`.

## Stack

- **Framework:** Wails v2 (Go backend + WebView frontend via WKWebView on macOS)
- **UI:** React 18.2, TypeScript 4.6, Vite 3
- **Rich text:** TipTap v3.7.2 (ProseMirror-based) with Markdown plugin (`tiptap-markdown`)
- **Table:** @tanstack/react-table v8
- **Icons:** lucide-react v0.546
- **Scrollbars:** react-custom-scrollbars-2
- **Linter/Formatter:** Biome v2.4.7 (double quotes, semicolons, 2-space indent, 100-char line width)
- **Styling:** CSS custom properties (`--term-*` variables) driven by `ThemeProvider`. **No Tailwind or shadcn** — migration planned but not started.
- **State:** All app state lives in `App.tsx` (`Inner` component). No external state library. Settings use React Context (`SettingsProvider`). Theme uses React Context (`ThemeProvider`).
- **Forms:** No form library. Manual state management with `useState`.
- **Path alias:** `@` resolves to `frontend/src/`
- **TypeScript:** Extends `@tsconfig/strictest`, target ESNext, strict mode

## Project Structure

```
frontend/
├── biome.json              # Biome linter/formatter config
├── index.html              # HTML entry point
├── package.json            # Dependencies and scripts
├── tsconfig.json           # TS config (extends @tsconfig/strictest)
├── vite.config.ts          # Vite config with @ alias
├── wailsjs/                # AUTO-GENERATED Wails bindings — never edit
│   ├── go/main/App.d.ts    # TypeScript declarations for Go methods
│   ├── go/main/App.js      # JS bindings for Go methods
│   └── go/models.ts        # Generated Go struct types
└── src/
    ├── main.tsx             # React root render
    ├── style.css            # Global styles
    ├── vite-env.d.ts        # Vite type declarations
    ├── pages/
    │   └── App.tsx          # Root component — ALL app state, event handlers, layout (1395 lines)
    ├── components/
    │   ├── TerminalList.tsx       # Main item list with scope/date grouping (915 lines)
    │   ├── EditItemModal.tsx      # Create/edit modal with TipTap editor (1264 lines)
    │   ├── SettingsModal.tsx      # Settings UI + SettingsProvider context (1653 lines)
    │   ├── SessionContextModal.tsx # Session filter management (669 lines)
    │   ├── InboxView.tsx          # Inbox triage view (559 lines)
    │   ├── ChatPanel.tsx          # AI chat panel (463 lines)
    │   ├── QuickSearchModal.tsx   # Global search modal (385 lines)
    │   ├── RadialMenuWrapper.tsx  # Right-click radial action menu (352 lines)
    │   ├── SearchAutocomplete.tsx # Search bar with autocomplete (236 lines)
    │   ├── AutocompleteInput.tsx  # Generic multi-value autocomplete (242 lines)
    │   ├── TagsAutocomplete.tsx   # Tags autocomplete (wraps AutocompleteInput)
    │   ├── ProjectsAutocomplete.tsx # Projects autocomplete (wraps AutocompleteInput)
    │   ├── ContextsAutocomplete.tsx # Contexts autocomplete (wraps AutocompleteInput)
    │   ├── ItemTitleInput.tsx     # Title input with inline parsing (204 lines)
    │   ├── ConfirmDialog.tsx      # Reusable confirmation dialog (104 lines)
    │   ├── NotesModal.tsx         # Standalone notes viewer modal (190 lines)
    │   ├── MetaModal.tsx          # Metadata editor modal (245 lines)
    │   ├── HelpModal.tsx          # Help/keyboard shortcuts reference (228 lines)
    │   ├── PowerMenu.tsx          # Quit/restart power menu (107 lines)
    │   ├── WelcomeDialog.tsx      # First-run welcome flow (270 lines)
    │   ├── AlphaWarning.tsx       # Alpha/beta warning dialog (189 lines)
    │   ├── VaultSwitcher.tsx      # Vault selection modal (198 lines)
    │   ├── KeyboardScope.tsx      # Global keyboard shortcut handler (68 lines)
    │   ├── CustomScrollbar.tsx    # Custom scrollbar wrapper (71 lines)
    │   ├── CopyrightFooter.tsx    # Footer branding (136 lines)
    │   ├── ActionCard.tsx         # Onboarding action cards (258 lines)
    │   ├── WikilinkSuggestion.tsx # TipTap @ mention suggestion UI (120 lines)
    │   ├── ThemeSettingsModal.tsx  # Theme editor (hidden from UI) (208 lines)
    │   ├── TagsInput.tsx          # Tag chip input component (112 lines)
    │   ├── TemplatePickerCombobox.tsx # Template selection dropdown (161 lines)
    │   ├── TemplateSaveDialog.tsx # Save-as-template dialog (121 lines)
    │   ├── ChatMessage.tsx        # Individual chat message bubble (60 lines)
    │   └── QuickAddModal.tsx      # DEAD FILE — unused, never imported (115 lines)
    ├── lib/
    │   ├── query.ts              # Search query parser (todo.txt-style syntax)
    │   ├── sessionContext.ts     # Session profile localStorage helpers
    │   ├── templates.ts          # Task template localStorage CRUD
    │   ├── vaultStorage.ts       # Vault-scoped localStorage helpers
    │   ├── WikilinkExtension.tsx # TipTap custom extension for @mentions
    │   ├── main.js               # DEAD FILE — old radial menu demo code
    │   ├── main.css              # DEAD FILE — old radial menu styles
    │   ├── radial-menu.js        # DEAD FILE — stub (14 bytes, "404: Not Found")
    │   ├── radial-menu.css       # DEAD FILE — stub (14 bytes, "404: Not Found")
    │   ├── RadialMenu.js         # DEAD FILE — old radial menu library (576 lines)
    │   └── RadialMenu.css        # DEAD FILE — old radial menu styles
    └── theme/
        ├── theme.ts              # TermTheme type + preset definitions (157 lines)
        ├── theme.css             # CSS custom property definitions
        └── ThemeProvider.tsx     # Theme context provider (47 lines)
```

## Component Inventory

| Component | Location | Use for |
|-----------|----------|---------|
| `TerminalList` | `components/TerminalList.tsx` | Main list view with scope sections (Now/Soon/Anytime) and date grouping |
| `EditItemModal` | `components/EditItemModal.tsx` | Creating and editing items/notes with TipTap rich text editor |
| `SettingsModal` | `components/SettingsModal.tsx` | App settings + exports `SettingsProvider` and `useSettings` hook |
| `InboxView` | `components/InboxView.tsx` | Inbox triage view with batch operations |
| `SessionContextModal` | `components/SessionContextModal.tsx` | Session filter profiles management |
| `ConfirmDialog` | `components/ConfirmDialog.tsx` | Generic confirmation dialog |
| `AutocompleteInput` | `components/AutocompleteInput.tsx` | Multi-value autocomplete input — base component |
| `TagsAutocomplete` | `components/TagsAutocomplete.tsx` | Thin wrapper loading tag suggestions from backend |
| `ProjectsAutocomplete` | `components/ProjectsAutocomplete.tsx` | Thin wrapper loading project suggestions from backend |
| `ContextsAutocomplete` | `components/ContextsAutocomplete.tsx` | Thin wrapper loading context suggestions from backend |
| `SearchAutocomplete` | `components/SearchAutocomplete.tsx` | Search bar with live autocomplete |
| `CustomScrollbar` | `components/CustomScrollbar.tsx` | Theme-aware scrollbar wrapper |
| `KeyboardScope` | `components/KeyboardScope.tsx` | Global keyboard shortcut listener |
| `RadialMenuWrapper` | `components/RadialMenuWrapper.tsx` | Right-click context menu for items |
| `QuickSearchModal` | `components/QuickSearchModal.tsx` | Global quick search (Cmd+K style) |
| `ChatPanel` | `components/ChatPanel.tsx` | AI chat sidebar |
| `VaultSwitcher` | `components/VaultSwitcher.tsx` | Vault switching modal |
| `ThemeProvider` | `theme/ThemeProvider.tsx` | Theme context + CSS variable injection |

## Patterns to Follow

### Data Flow
- Go backend exposes methods on `*App` struct via Wails bindings
- Frontend calls `Backend.MethodName()` (async) from auto-generated `wailsjs/go/main/App`
- All app state is held in `App.tsx` (`Inner` function component)
- State changes trigger `runSearch()` which re-fetches from backend
- Child components receive data and callbacks via props

### Styling
- Use CSS custom properties: `var(--term-bg)`, `var(--term-fg)`, `var(--term-accent)`, `var(--term-border)`, `var(--term-panel)`, `var(--term-dim)`, `var(--term-success)`, `var(--term-warn)`, `var(--term-info)`
- All styling is inline `style={{}}` objects — no CSS modules, no Tailwind
- Theme presets defined in `theme/theme.ts`, applied via `ThemeProvider`
- Badge classes: `badge`, `badge warn`, `badge success`, `badge info`, `badge fg` for pill buttons

### Component Design
- Modal pattern: `open` boolean + `onOpenChange` callback props
- All modals render their own full-screen overlay (`position: fixed, inset: 0`)
- Components call backend directly when they need data (e.g., autocomplete components each call `Backend.GetFilters()`)
- Default exports for components
- Props type defined at top of file, named `Props` (not `{Component}Props`)

### Backend Communication
- Import as `import * as Backend from "../../wailsjs/go/main/App"`
- After mutations, call `runSearch()` to refresh the list
- Backend returns Go structs serialized as JSON; frontend uses `as any` for flexible handling

### localStorage
- All keys use `nanite.` prefix
- Vault-scoped keys use `nanite.vault.{vaultId}.{key}` format
- Key modules: `sessionContext.ts`, `templates.ts`, `vaultStorage.ts`

## Anti-Patterns Found

### AP1 — God Component: `App.tsx` (1395 lines)
- **File:** `src/pages/App.tsx`
- **Issue:** Single `Inner()` function holds 30+ `useState` calls, all event handlers, all layout JSX, search logic, animation logic, and inline modal orchestration. This is the single source of truth for state by design, but the file mixes concerns: search/filter logic, CRUD handlers, animation state, keyboard shortcuts, onboarding flow, and 700+ lines of JSX.
- **Impact:** Any change to any feature requires reading/modifying this file. High merge conflict risk.

### AP2 — God Component: `SettingsModal.tsx` (1653 lines)
- **File:** `src/components/SettingsModal.tsx`
- **Issue:** Contains the `SettingsProvider`, `useSettings` hook, `Settings` type definition, AND the entire settings UI across 5 tabs (general, tabs, data, vaults, chat). Exports both the modal component and the context provider from the same file.
- **Impact:** Importing `useSettings` forces loading the entire 1653-line modal component.

### AP3 — God Component: `EditItemModal.tsx` (1264 lines)
- **File:** `src/components/EditItemModal.tsx`
- **Issue:** Handles both create and edit modes, TipTap editor setup, template management, session context merging, backlinks, formatting toolbar, and all form state. 19 `useState` calls.

### AP4 — Excessive `any` Usage (37 occurrences across 11 files)
- **Files:** `App.tsx` (18), `SettingsModal.tsx` (4), `EditItemModal.tsx` (3), `query.ts` (2), others
- **Issue:** Backend return types are cast with `as any` instead of using proper types from `wailsjs/go/models`. `req: any` objects are hand-constructed instead of using generated types. This defeats TypeScript's type safety.
- **Example:** `App.tsx:309` — `const req: any = { query: merged.keywords.join(" "), ... }`

### AP5 — Duplicated Modal Overlay Pattern (13 files)
- **Files:** Every modal component independently renders `position: fixed; inset: 0; background: rgba(0,0,0,0.75)` overlay divs with nearly identical inline styles.
- **Impact:** No shared `Modal` or `Overlay` abstraction. Each modal reimplements overlay, centering, backdrop click handling, and z-index management independently.

### AP6 — Duplicated Autocomplete Backend Calls
- **Files:** `TagsAutocomplete.tsx`, `ProjectsAutocomplete.tsx`, `ContextsAutocomplete.tsx`
- **Issue:** All three components independently call `Backend.GetFilters()` on mount, each making a separate backend call to get the same data, then extracting their respective field. Also, `EditItemModal.tsx` makes its own `Backend.GetFilters()` call.
- **Impact:** 4 redundant backend calls when the edit modal opens with all three autocomplete components.

### AP7 — Dead Files (6 files)
- **Files:**
  - `src/lib/main.js` — old radial menu demo (121 lines)
  - `src/lib/main.css` — associated styles
  - `src/lib/radial-menu.js` — stub file (14 bytes, contains "404: Not Found")
  - `src/lib/radial-menu.css` — stub file (14 bytes, contains "404: Not Found")
  - `src/lib/RadialMenu.js` — old radial menu library (576 lines)
  - `src/lib/RadialMenu.css` — old radial menu styles
  - `src/components/QuickAddModal.tsx` — component never imported anywhere (115 lines)
- **Impact:** 7 dead files totaling ~1000 lines of unused code.

### AP8 — Inconsistent localStorage Key Prefixes
- **File:** `src/theme/ThemeProvider.tsx:10,39`
- **Issue:** `ThemeProvider` uses the old `todo.term.theme` localStorage key while the rest of the app uses the `nanite.` prefix. The `SettingsProvider` has migration logic for the old `todo.settings` key but `ThemeProvider` was never migrated.

### AP9 — Excessive Console Logging (78 occurrences across 17 files)
- **Files:** `App.tsx` (29 occurrences), `SettingsModal.tsx` (13), `EditItemModal.tsx` (7), `InboxView.tsx` (7), `ThemeProvider.tsx` (5)
- **Issue:** Debug `console.log` statements throughout production code, including verbose search debugging (`[runSearch]`, `[Search]`, `[InputSubmit]`).

### AP10 — Inline Styles Everywhere
- **Files:** All component files
- **Issue:** All styling is done via inline `style={{}}` objects, often with 10-20 properties per element. No CSS classes, no utility framework, no design token abstraction beyond theme variables. Makes responsive design and consistent spacing impossible to enforce.
- **Example:** `App.tsx:728-738` — drag region div with 9 inline style properties plus `@ts-ignore` for Wails-specific CSS properties.

### AP11 — Duplicated Search Logic
- **File:** `src/pages/App.tsx` — `runSearch()` at line 260 and `handleInputSubmit()` at line 611
- **Issue:** `handleInputSubmit` (quick add mode) rebuilds the entire search request and result merging logic instead of calling `runSearch()`. Both functions construct identical `req` objects with the same structure, perform the same completed-item merge, and call the same backend methods.

### AP12 — Code Duplication: `handleQuickAdd` and `handleQuickAddNote`
- **File:** `src/pages/App.tsx:502-533`
- **Issue:** These two functions are nearly identical — they differ only in calling `Backend.CreateItemFromLine` vs `Backend.CreateNoteFromLine`. The merge logic is copy-pasted.

### AP13 — `@ts-ignore` Suppression (4 occurrences)
- **File:** `src/pages/App.tsx`
- **Issue:** Used to suppress errors for Wails-specific CSS properties (`--wails-draggable`, `WebkitAppRegion`). Should be typed properly or use a type-safe wrapper.

## Reference Implementations

| Pattern | Reference File | Why it's good |
|---------|---------------|---------------|
| localStorage helper module | `src/lib/vaultStorage.ts` | Clean, focused module with proper generic typing, error handling, and namespaced key management. 39 lines doing one thing well. |
| Reusable dialog component | `src/components/ConfirmDialog.tsx` | Clean Props type, minimal logic, proper composition (overlay + content), early return for `!open`. 104 lines. |
| Query parser | `src/lib/query.ts` | Pure function with well-defined input/output types, no side effects, comprehensive token parsing. Would benefit from removing the 2 `as any` casts. |

## Design Tokens / Visual Rules

- **Theme variables:** All colors come from `var(--term-*)` — see `src/theme/theme.ts` for the `TermTheme` type
- **Presets:** `default` (dark blue/terminal), `light`, and potentially others in `themePresets`
- **Font:** System monospace stack (`ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas`)
- **Badge classes:** `badge` base + `warn`/`success`/`info`/`fg` modifiers for pill-style buttons
- **Spacing:** No consistent spacing scale — hardcoded pixel values throughout (`4px`, `6px`, `8px`, `10px`, `12px`, `16px`, `24px`, `32px`)
- **Border radius:** Inconsistent — `3px`, `4px`, `6px`, `8px`, `12px` used interchangeably

## Notes

- **WKWebView quirks:** On macOS, the frontend runs inside WKWebView. Standard DOM event handling has quirks — `click` events can be suppressed by ProseMirror's `mousedown` handling. Always test interactions in the actual app.
- **Wails bindings are auto-generated:** Never edit files in `frontend/wailsjs/`. Run `wails generate module` after Go method changes.
- **Tailwind migration planned:** The `CLAUDE.md` mentions a planned Tailwind v4 + shadcn/ui migration. Until then, continue using inline styles and `var(--term-*)` variables.
- **Theme switcher hidden:** The theme tab is commented out of the Settings UI but the system is fully functional in code.
- **DB table still named `todos`:** Despite the rename to "items" throughout the codebase, the SQLite table remains `todos`.
