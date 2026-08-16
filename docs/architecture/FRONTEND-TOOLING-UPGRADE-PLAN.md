# Frontend Tooling Upgrade Plan

Status: **plan only — not executed.** This document was produced for Torque task
`CW-20260424-0036` ("Refresh dated frontend tooling after verification is in place").
No dependency versions, config files, or lockfiles were changed while writing it. It
exists so that whoever picks up the actual upgrade later has a concrete, sequenced,
low-risk path to execute instead of re-deriving the compatibility research from
scratch.

**Re-verify before executing.** Every version number below was checked against the
live npm registry on 2026-08-16 (`npm view <pkg> version`, `npm view <pkg> dependencies`,
`npm view <pkg> peerDependencies`, `npm view <pkg> engines`). This ecosystem moves fast
(TypeScript 7.0 GA'd four weeks before this was written). Re-run the same `npm view`
commands immediately before starting the work — do not trust these numbers as
still-current.

Why this is a plan and not a PR: this task landed at the end of a large, 17-commit
portfolio branch (test coverage, an architecture doc, ADRs, 8 file-splits) that hadn't
been through review yet. Compounding that with a risky multi-major-version tooling jump
was judged worse than shipping the plan and executing separately once the baseline
branch has landed.

---

## 1. Current vs. latest (as of 2026-08-16)

| Package | Current (`frontend/package.json`) | Latest on npm | Gap |
|---|---|---|---|
| `vite` | `^3.0.7` (installed: 3.2.11) | `8.2.1` | 5 majors |
| `typescript` | `^4.6.4` (installed: 4.9.5) | `7.0.2` | 3 "majors" (5, 6, 7 — see §5 on why 6/7 aren't really adjacent hops) |
| `@vitejs/plugin-react` | `^2.0.1` | `6.0.5` | 4 majors |
| `vitest` | `^0.34.6` | `4.1.10` | effectively 4+ majors (0.x → 1 → 2 → 3 → 4) |
| `@testing-library/react` | `^14.3.1` | `16.3.2` | 2 majors |
| `@testing-library/jest-dom` | `^6.9.1` | `7.0.1` | 1 major |
| `jsdom` | `^24.1.3` | `30.0.1` | 6 majors |
| `@biomejs/biome` | `^2.4.7` | `2.5.8` | same major, patch/minor behind |
| `@tsconfig/strictest` | `^2.0.8` | `2.0.8` | **already current, no action needed** |
| `@tiptap/*` (react/starter-kit/mention/suggestion) | `^3.7.2` | `3.30.1` | same major, 23 minors behind |
| `react` / `react-dom` | `^18.2.0` | `19.2.8` | 1 major — **explicitly out of scope**, see §9 |
| `@types/react` / `@types/react-dom` | `^18.0.17` / `^18.0.6` | `19.2.18` / `19.2.4` | tracks React itself; stay on 18.x typings until React is upgraded |
| `@tanstack/react-table` | `^8.21.3` | `9.1.2` | already latest 8.x; 9.x is a separate, out-of-scope call |
| `lucide-react` | `^0.546.0` | `1.31.0` | out of scope, icon lib, no coupling to Vite/TS |
| `react-custom-scrollbars-2`, `tippy.js`, `tiptap-markdown` | pinned versions | same | already latest, no action |

The task's explicit focus is Vite, TypeScript, and "adjacent tooling" — read as: everything
in the Vite/Vitest/TypeScript build-and-test chain, not the whole dependency tree. TipTap,
Biome, lucide-react, and the TanStack majors are noted for completeness but are
independent, lower-risk, separately-schedulable bumps (see §9).

---

## 2. The coupled-dependency graph

Four packages move in lockstep and must be reasoned about together, not upgraded
independently: **Vite → `@vitejs/plugin-react` → Vitest → jsdom/Testing Library**. Node.js
version is the floor underneath all of them.

| Vite major | Node floor (`engines.node`) | Compatible `@vitejs/plugin-react` | Compatible Vitest majors (checked via `dependencies.vite`, not `peerDependencies` — older Vitest majors don't declare Vite as a peer) | Notable breaking changes |
|---|---|---|---|---|
| 3.x (current) | `^14.18.0 \|\| >=16.0.0` | 2.x (`peer: vite ^3.0.0`) | 0.34.x (`^3.1.0 \|\| ^4.0.0 \|\| ^5.0.0-0`) | — (current baseline) |
| 4.x | `^14.18.0 \|\| >=16.0.0` | 3.x (`peer: vite ^4.0.0`) or 4.x (`peer: vite ^4.2.0`) | 0.34.x still valid (same range as above covers `^4.0.0`) | Mostly mechanical; internal Rollup/esbuild bumps, few user-facing config breaks |
| 5.x | `^18.0.0 \|\| >=20.0.0` (**Node 14/16/17/19 dropped**) | 4.x (needs `>=4.2.0`) or 5.x (`peer: vite ^4.2.0 \|\| ^5 \|\| ^6 \|\| ^7`, wide range) | 1.x, 2.x require exactly `^5.0.0`; 0.34.x also still covers `^5.0.0-0` (beta only, not final 5.x) | Manifest.json: CSS no longer a top-level manifest entry; Rollup 4 renames import assertions → import attributes; CLI shortcuts need extra Enter keypress |
| 6.x | `^18.0.0 \|\| ^20.0.0 \|\| >=22.0.0` | 5.x | 3.0.x (`^5.0.0 \|\| ^6.0.0`) | Vite team explicitly minimized breaks; new (initially experimental) Environment API groundwork; CSS lib-mode output naming change |
| 7.x | `^20.19.0 \|\| >=22.12.0` (**Node 18 dropped**) | 5.x (wide range already covers it) | 3.2.4+ (`^5.0.0 \|\| ^6.0.0 \|\| ^7.0.0-0`); 4.0.x (`^6.0.0 \|\| ^7.0.0`, **drops Vite 5 support**) | Several long-deprecated internals removed (`legacy.proxySsrExternalModules`, `ModuleRunnerOptions.root`, etc.); **see §4 for a live Wails-specific regression in 7.1.8** |
| 8.x | `^20.19.0 \|\| >=22.12.0`, package is now **ESM-only** | 6.x (`peer: vite ^8.0.0` only — no longer supports older Vite majors) | 4.1.x (`^6.0.0 \|\| ^7.0.0 \|\| ^8.0.0`) | **Biggest change since Vite 2**: default bundler swapped to Rolldown (Rust-based, unifies the previous esbuild+Rollup dual-bundler setup); `build.rollupOptions` renamed `build.rolldownOptions`; CJS interop changes; Yarn PnP incompatible |

Practical reading of this table: `@vitejs/plugin-react@5.x` has a peer range wide enough
(`^4.2.0 || ^5.0.0 || ^6.0.0 || ^7.0.0`) to ride through the Vite 5→6→7 hops without a
second plugin bump — the plugin only needs to move again for the eventual Vite 8 hop
(where it becomes Rolldown-specific and hard-pins to `vite ^8.0.0`).

Testing Library and jest-dom are less tightly coupled to Vite itself, but do track React
and Node:

- `@testing-library/react` 14.x/15.x peer only `react: ^18.0.0`; 16.x widens to
  `react: ^18.0.0 || ^19.0.0` — no reason to move past 14.x/15.x until this project
  actually upgrades React.
- `@testing-library/jest-dom` 6.x has a low `engines.node: >=14` floor; 7.x jumps to
  `engines.node: >=22` and adds a `peerDependencies.vitest: >=0.32` constraint. Since the
  Vite 7/8 hops already require Node ≥20.19/22.12, this isn't an extra constraint in
  practice by the time you get there — just don't bump jest-dom to 7.x while still on an
  earlier Vite/Node floor without checking the Node version on the actual dev/CI machine.
- `jsdom` 24.x–26.x all declare `engines.node: >=18`; **jsdom 30.x jumps to
  `engines.node: ^22.22.2 || ^24.15.0 || >=26.0.0`** — a much narrower floor than Vite
  itself requires. This is the tightest Node constraint in the whole chain; check it
  explicitly before bumping jsdom past the 26–29 range.

---

## 3. TypeScript: 4.6 → 5.x → 6.0 → 7.0

npm's `dist-tags` for `typescript` as of 2026-08-16:

```
beta: 6.0.0-beta   rc: 7.0.1-rc   latest: 7.0.2   next: 7.1.0-dev...
```

TypeScript 6.0 (GA March 2026) is described by the TS team as "the last JavaScript-based
release before the Go-powered TypeScript 7." TypeScript 7.0 (GA July 2026) is a full
**native port of the compiler to Go** — not an incremental release. That reframes the
"4.6 → latest" jump: it isn't one long ladder of comparable rungs, it's 4.6→5.x (a normal
multi-year TS upgrade) followed by a *architecture change* (5.x/6.x→7.x) that the
ecosystem is still catching up to.

**Recommended target for this pass: TypeScript 5.9.x (latest 5.x), not 6.0 or 7.0.**
Reasons:

1. **TypeScript 7.0 has no stable programmatic compiler API yet.** Tools that use the TS
   API directly (`typescript-eslint`, `ts-jest`, `ts-morph`, etc.) can't run against it —
   `typescript-eslint`'s published peer range only allows `typescript <6.1.0`, and forcing
   it crashes inside `typescript-estree`. The stable API is targeted for TypeScript 7.1
   (expected ~October 2026, per Microsoft's own messaging as of this writing).
   **This project uses Biome for linting, not ESLint/`typescript-eslint`**, so the
   headline ecosystem blocker doesn't directly apply here — but re-verify before jumping
   to 7.x that no other devDependency (test reporters, any future tooling) touches the TS
   Program/LanguageService API programmatically. Plain `tsc` CLI invocation (`"build": "tsc
   && vite build"`) is the safer, more isolated use pattern and is less exposed to this
   gap.
2. **Both 6.0 (5 months old) and 7.0 (1 month old) are very young** relative to this
   project's risk tolerance for a build-critical dependency. 5.9.x has had a full TS 5.x
   release cycle to shake out.
3. **A concrete, codebase-specific blocker exists for 7.0 regardless of the API question**:
   `frontend/tsconfig.json` currently sets `"moduleResolution": "Node"` (the legacy
   `node10` resolution strategy). TypeScript 7.0 **removes `node10` support entirely** —
   the option has been emitting a deprecation warning since TS 5.x (deprecated flags "had
   no effect from 5.5, became errors in 6.0" per the TS 6.0 release notes) and is a hard
   error by 7.0. **Action needed before any TS 6/7 hop**: change `moduleResolution` from
   `"Node"` to `"bundler"` (the recommended value for Vite-bundled projects — matches how
   Vite itself resolves modules, and needs no `.js` extension rewriting in relative
   imports, unlike `"nodenext"`). Do this as **its own isolated step**, verified with
   `tsc --noEmit` and the frontend test suite, before touching the TypeScript version at
   all — it's independent of which TS version you land on, and derisks the eventual 6/7
   hop preemptively.

`@tsconfig/strictest` is already at its latest published version (2.0.8) — no bump
needed there. But its assumptions should be re-checked at each TS target, since strictest
configs are exactly the kind of thing that surfaces newly-enforced strictness:

- TS 5.9 turns on `--strictInference` under `--strict` by default (a real behavior
  change, not just a lint-level tweak) — this can surface newly-inferred `any`/type
  errors in code that compiled cleanly under 4.6-generation inference.
- TS 6.0 removes `--keyofStringsOnly` (deprecated, not used here — grep confirms no
  occurrence in `frontend/tsconfig*.json`) and assumes JS strict mode unconditionally.

**Concrete verification step for the TS hop**: after bumping to 5.9.x, run `npx tsc
--noEmit` standalone (not just as part of `vite build`) and read every new diagnostic
before assuming "the build passed" is sufficient — a newly-stricter inference rule can
pass silently through `tsc && vite build` if `vite build`'s own esbuild-based
transpilation doesn't re-run full type-checking (it doesn't; Vite's dev/build pipeline
strips types without checking them, so `tsc`'s own exit code is the only signal that
matters here).

---

## 4. Wails-specific risk

This frontend is built via `wails build`, which — per `wails.json` — shells out to
`npm run build` (`"frontend:build": "npm run build"`, itself `tsc && vite build`), and
`wails dev` shells out to `npm run dev` (`"frontend:dev:watcher": "npm run dev"`, i.e.
`vite`) while Wails' own process proxies/watches the dev server URL
(`"frontend:dev:serverUrl": "auto"`). Wails does not pin or vendor a Vite version itself
— it just orchestrates whatever `npm run build`/`npm run dev` in this repo happen to
invoke. That means Wails is exposed to Vite's dev-server protocol and CLI behavior
indirectly, and a Vite change that's invisible to `vite build`/`tsc` alone can still break
`wails build`/`wails dev`.

**Confirmed, specific regression** (via `gh issue view 4620 --repo wailsapp/wails`):

> Wails can't connect to the dev server if you use Vite 7.1.8 — [wailsapp/wails#4620](https://github.com/wailsapp/wails/issues/4620)

The dev server would start and report itself online, but Wails' embedded WebView could
not reach it at all — not even by manually navigating to the reported URL. One commenter
confirmed **"Same problem with wails v2"** (the original report used `wails3 dev`, but
the environment details show it was tested against Wails v2.10.2 — the same version this
repo is on per `go.mod`). It was traced to a Vite PR
([vitejs/vite#20885](https://github.com/vitejs/vite/pull/20885)) that got reverted, and
was **confirmed fixed in Vite 7.1.9**. This is not a hypothetical compatibility note —
it's a real, recent, patch-level regression in exactly the version line this plan
recommends landing on (see §6). Two takeaways:

1. **Never target a specific Vite patch by "latest is always safest" reasoning** — pin to
   a specific, verified-working patch version and re-check for open Wails-integration
   issues against that exact version before landing it, not just the major/minor.
2. **`vite build` and `tsc` succeeding is not sufficient evidence that a Vite bump is
   safe for this app.** The dev-server proxy behavior Wails depends on (`wails dev`) is a
   distinct code path from the production build Vite pipeline. Every Vite version bump in
   the sequencing plan below includes an explicit `wails dev` smoke test, not just
   `make verify`.

**Secondary risk — no Node version floor is enforced anywhere in this repo.** There is no
`.nvmrc`, no `engines` field in `frontend/package.json`, and no CI workflow (`.github/`
doesn't exist in this repo) pinning a Node version. Vite 5/6/7/8 each raise the Node
floor (see §2's table); without an enforced floor, a contributor or a future CI runner on
an older Node version will get a cryptic `npm install`/`vite` failure rather than a clear
"upgrade Node" message. **Recommend adding an `engines.node` field to
`frontend/package.json`** (and ideally a `.nvmrc`) as part of the first Vite-major hop in
this plan, set to match whatever Vite major is landed. This wasn't asked for as a config
change to execute now, but should be the first concrete diff in the actual upgrade PR.

---

## 5. Codebase-specific risk areas to re-verify after any Vite/Vitest bump

These were found by reading the current `frontend/vite.config.ts` and
`frontend/src/test/setup.ts` — both contain environment-interaction workarounds that
were discovered as **real bugs this session**, which means they are exactly the kind of
code most likely to silently break (or silently become dead code) on a Vite/Vitest major
bump:

1. **`vite.config.ts` — Fast Refresh mode guard.**
   ```ts
   // Fast Refresh's runtime preamble is only injected by Vite's dev-server
   // HTML transform, so it must stay off under Vitest (mode: 'test') or
   // @vitejs/plugin-react throws "can't detect preamble" on first render.
   plugins: [react({ fastRefresh: mode !== "test" })],
   ```
   This encodes a specific interaction between `@vitejs/plugin-react`'s HMR preamble
   injection and how Vitest invokes Vite's transform pipeline outside a real dev server.
   `@vitejs/plugin-react` changed its internal preamble-detection mechanism across majors
   (2.x → 6.x); re-verify this guard is still necessary (not a no-op, not newly
   insufficient) every time `@vitejs/plugin-react` moves to a new major — the failure
   mode if this silently stops working is every component test throwing on first render,
   which `make verify`'s frontend-test step will catch loudly (good), but debugging why
   would cost real time without this note.

2. **`vite.config.ts` — path alias (`@` → `./src`).** Matches `tsconfig.json`'s
   `"paths": {"@/*": ["./src/*"]}`. Vite's `resolve.alias` config shape hasn't changed
   across the 3→8 range in any breaking way found in this research, but re-confirm after
   the TypeScript `moduleResolution` change (§3) — switching to `"bundler"` resolution
   changes how TS itself resolves the alias for type-checking purposes even though Vite's
   runtime resolution is unaffected; a mismatch here would show up as `tsc` errors on
   `@/...` imports that `vite build` doesn't reproduce.

3. **`vite.config.ts` — `test.setupFiles` / `test` block.** Currently minimal
   (`environment: "jsdom"`, one setup file). Vitest 3.x renamed `workspace` →
   `projects`; not used here, so no direct impact, but if a future contributor adds a
   workspace/projects config as part of the Vitest bump, use the new key from the start
   rather than the deprecated one.

4. **`src/test/setup.ts` — manual `afterEach(cleanup)` registration.** The comment in
   this file explains that `@testing-library/react`'s auto-cleanup only self-registers
   when it detects Vitest's `test.globals: true`, which this project deliberately does
   not set (tests import `afterEach`/`describe`/`it` explicitly). Re-verify this
   detection mechanism is unchanged after bumping `@testing-library/react` past 14.x —
   if a future major changes how auto-cleanup detects the test globals, this explicit
   registration could become redundant (harmless) or could interact badly with a new
   auto-registration (double-cleanup, usually harmless but worth a quick look at test
   output for warnings).

5. **`src/test/setup.ts` — `localStorage` polyfill workaround.** This is the highest-risk
   item on this list. The comment documents a live version-specific bug: Node 22+ ships
   its own inert global `localStorage`, and **this Vitest version's** jsdom-environment
   setup only copies jsdom's working `localStorage` implementation onto the global scope
   when Node doesn't already define one of the same name — so on Node 22+, jsdom's real
   implementation never gets installed, silently breaking every `localStorage` call in
   the app under test. The current Node in this environment is v26, meaning **this
   workaround is live and load-bearing today**, not theoretical. Two things can change
   this on a bump:
   - `jsdom` 30.x (latest) raises its own Node floor to `^22.22.2 || ^24.15.0 ||
     >=26.0.0` — i.e., it now *only* supports the exact Node line where this collision
     happens, which raises real hope the jsdom maintainers fixed the underlying
     global-detection order as part of that floor-raise. **Explicitly test this**: after
     bumping `vitest`/`jsdom`, try removing the polyfill block in a scratch branch and
     rerun the suite — if `localStorage`-dependent tests (settings, session context, task
     templates persistence) still pass, the workaround is now dead code and should be
     deleted with a comment explaining why, rather than left as silent legacy code no one
     understands the continued necessity of.
   - Vitest itself may change how it wires jsdom's globals (this is described as "this
     Vitest version's" environment setup specifically, implying it's known to be
     version-sensitive) — the same re-test applies after any Vitest major bump even if
     jsdom doesn't move.

6. **`src/test/setup.ts` — `Element.prototype.scrollIntoView` stub.** Documented as a
   permanent, well-known jsdom gap (jsdom deliberately doesn't implement layout), not a
   version-transient bug — low risk of needing changes, but re-confirm jsdom hasn't added
   a real (or intentionally-throwing) implementation that would make the `typeof
   ... !== "function"` guard behave differently.

---

## 6. Recommended sequencing plan

General principles used below: **lower-coupling-risk first**, **one major-version hop
per commit** (never a single "bump everything" commit — see §7 for why), and a
verification gate after every single step, not just at the end.

### Phase 0 — Prep (no version bumps)

1. Add `engines.node` to `frontend/package.json` reflecting the *current* Vite 3
   requirement (`"^14.18.0 || >=16.0.0"`) as a baseline, to be tightened at each later
   phase. Add a `.nvmrc` if the team wants Node-version managers to pick it up
   automatically.
2. Change `frontend/tsconfig.json`'s `"moduleResolution": "Node"` → `"bundler"`. This is
   independent of the TypeScript version bump (works fine under TS 4.6–5.x already) and
   derisks the TS 6/7 hop preemptively (§3). Verify with `npx tsc --noEmit` and
   `make frontend-test`.
3. Commit each of the above separately. Run `make verify` after each.

### Phase 1 — TypeScript 4.6 → 5.9.x (isolated)

1. Bump `typescript` to `^5.9.x` (check the actual current 5.x latest at execution time —
   5.9.3 as of this writing). Nothing else changes in this commit.
2. Verify: `npx tsc --noEmit` standalone first (read every new diagnostic — see §3's note
   on why `vite build` alone won't catch new strictness errors), then `make verify`
   end-to-end (lint + test + frontend-build + codegen-check + build), then a real `wails
   build` and manual app smoke test (open the built app, exercise a few core flows).
3. This step should NOT need `@vitejs/plugin-react`, `vite`, or `vitest` to move — verify
   that's actually true (Vite 3's own bundled TS-stripping via esbuild is
   version-tolerant of newer `typescript` in `devDependencies` since Vite doesn't invoke
   `tsc` itself; the project's `"build": "tsc && vite build"` script is what actually
   invokes the new compiler).
4. Commit in isolation. If something breaks here, it's unambiguously a TypeScript-version
   issue, not a Vite one.

### Phase 2 — Vite 3 → 5, plugin-react, and Vitest, as one coupled bump group (still isolated per major hop)

This is the core coupled group from §2. Do it as **two commits**, not one:

**2a. Vite 3 → 4.**
1. Bump `vite` to latest 4.x. Bump `@vitejs/plugin-react` to 3.x or 4.x (whichever pairs
   with the landed Vite 4.x patch — re-check `peerDependencies` at execution time; this
   plan found 3.x needs `vite ^4.0.0`, 4.x needs `vite ^4.2.0`, so prefer landing on a
   Vite 4.2+ patch and taking `@vitejs/plugin-react@4.x` to set up for the wide-range 5.x
   plugin later).
2. Vitest stays at `0.34.6` through this hop (its range already covers `^4.0.0`) —
   **do not bump Vitest yet**.
3. Verify: `make verify`, `wails dev` smoke test (start it, confirm the WebView connects,
   confirm HMR works by editing a component), `wails build` + manual app smoke test.
4. Commit.

**2b. Vite 4 → 5, `@vitejs/plugin-react` → 5.x, Vitest 0.34 → latest 2.x (or newest available at execution time), Testing Library harness re-pinned.**
1. Bump `vite` to latest 5.x.
2. Bump `@vitejs/plugin-react` to 5.x (wide peer range covers this and the next two Vite
   hops — no need to touch it again until Vite 8).
3. Bump `vitest` to a version whose `dependencies.vite` range includes the landed Vite
   5.x (this plan found `vitest@1.x`/`2.x` require exactly `^5.0.0` — re-check current
   latest 2.x at execution time). **This is the point where the test harness's original
   "generation contemporaneous with Vite 3.2.11" pinning rationale (this session's
   deliberate `vitest@^0.34.6` choice) needs to be explicitly retired** — update any
   comment/doc referencing that rationale so it doesn't mislead a future reader into
   thinking Vitest is still intentionally pinned old.
4. `@testing-library/jest-dom` can likely stay on 6.x here (Node floor for 7.x is `>=22`,
   which may not yet be enforced — check the Node version actually in use). `jsdom` and
   `@testing-library/react` can stay put unless a specific Vitest 1.x/2.x compatibility
   issue forces otherwise.
5. Verify: full `make verify`, all 78 tests passing (not just "suite exits 0" — check the
   actual test count in the output matches expectations, since a config regression can
   cause silent test-skip rather than failure), the §5 risk-area re-checks (especially
   the `localStorage` polyfill — try removing it in scratch to see if it's now dead),
   `wails dev` + `wails build` smoke tests.
6. Commit.

### Phase 3 — Vite 5 → 6 → 7 (each its own commit)

1. **Vite 5 → 6**: bump `vite` only. `@vitejs/plugin-react@5.x` already covers this. Bump
   `vitest` to a 3.0.x release if the Vitest 1.x/2.x line doesn't support Vite 6 (this
   plan found Vitest 3.0.x is the first line whose range includes `^6.0.0`). Verify per
   the same gate as above. Commit.
2. **Vite 6 → 7**: bump `vite` to a **specific verified-good patch, explicitly avoiding
   7.1.8** (confirmed broken for `wails dev`; 7.1.9+ confirmed fixed — re-check for any
   newer regressions against the actual target patch before landing it). Bump `vitest` to
   a 3.2.4+ or 4.0.x release per its Vite-range support at execution time. **This step's
   verification gate must include the `wails dev` smoke test as a hard requirement, not
   optional** — this is the exact version line with the confirmed regression. Commit.

Stop here for this pass unless there's a specific reason to chase Vite 8 (Rolldown) —
see below.

### Phase 4 (separate, later effort) — Vite 8 / Rolldown

Vite 8's Rolldown-based bundler is the single largest architectural change in this whole
plan — bigger than any TypeScript hop. Recommend treating it as its own follow-up task,
not bundled into "refresh dated tooling," for these reasons:

- It requires `@vitejs/plugin-react` 6.x, which **only** supports `vite ^8.0.0` — no
  gradual overlap window like the 5.x plugin had.
- `build.rollupOptions` → `build.rolldownOptions` is a config rename that could affect
  any custom Rollup options if this project ever adds them (currently `vite.config.ts`
  has none, which makes this a lower-risk hop than it would be for a project with a
  complex build config — worth re-confirming at execution time that's still true).
- The Vite team's own guidance is to first trial the `rolldown-vite` package **on top of
  Vite 7** to isolate Rolldown-specific issues before the full Vite 8 jump — a two-step
  process in itself.
- Vitest only added `^8.0.0` support in the 4.1.x line — check whether that's matured
  (patch-stable) by the time this phase is picked up.

### Phase 5 (separate, deferred, not this task) — TypeScript 6.0 / 7.0

Revisit once: (a) TypeScript 7.1 ships with a stable programmatic API (removes the
ecosystem-tooling risk even though Biome sidesteps the acute `typescript-eslint`
blocker), and (b) enough time has passed for 6.0/7.0-specific bug reports to surface and
get fixed. The `moduleResolution: "bundler"` prerequisite from Phase 0 will already be
in place by then, removing one blocker ahead of time.

---

## 7. Rollback / bisection strategy

- **One version-hop per commit, always.** A regression discovered later (e.g., "the app
  hangs on cold start" reported days after a merge) should be bisectable with `git
  bisect` to a single specific dependency-version change, not "somewhere in this batch of
  12 packages that moved 4 majors each." This is the single most important process rule
  in this plan — a giant `npm-check-updates --upgrade`-style commit would make any
  regression here nearly undebuggable given how many moving, coupled pieces are involved
  (§2).
- Each commit's verification gate (§6) is also its rollback trigger: if `wails dev`,
  `wails build`, or `make verify` fails and the fix isn't obviously forward-compatible
  within ~30 minutes, revert that single commit rather than attempting to patch forward —
  the next attempt can re-research the specific failure with a narrower diff to explain.
- Keep the PR/branch for this work small and reviewable in the same spirit as the
  reasoning that produced this plan-only task in the first place: land Phase 0–3 as a
  reviewable unit (or even split further, one PR per phase) rather than one mega-PR,
  so a regression found post-merge in production usage still has a clean bisection
  target at the commit level *and* a clean revert target at the PR level.
- Tag or note the commit SHA before each phase begins (e.g., in the PR description) so a
  human reviewer doing a post-hoc bisection doesn't have to re-derive "which commit was
  the Vite 6→7 hop" from commit messages alone.

---

## 8. Explicitly out of scope for this upgrade (with rationale)

- **React 18 → 19.** Not "adjacent tooling" — it's a product-code-affecting framework
  major with its own breaking-change surface (new JSX transform assumptions already
  satisfied here since `jsx: "react-jsx"` is set, but ref-as-prop changes, removed
  APIs, etc. would need their own audit). `@testing-library/react` 16.x already supports
  React 19 whenever this is picked up, so the test harness isn't a blocker when that
  decision is made — but it's a separate task with its own risk profile, not a tooling
  refresh.
- **TipTap 3.7.2 → 3.30.1.** Same major, so lower risk than anything above, but also not
  Vite/TS-coupled — a good candidate for its own small, independent bump whenever
  convenient, decoupled from this plan's sequencing.
- **`@biomejs/biome` 2.4.7 → 2.5.8.** Same major, low risk, independent of the Vite/TS
  chain — bump opportunistically.
- **`@tanstack/react-table` 8.x → 9.x, `lucide-react` 0.x → 1.x.** Both majors, both
  unrelated to the build/test tooling chain this task is scoped to. Separate calls.
- **TypeScript 6.0 / 7.0.** See §6 Phase 5 — deferred pending ecosystem/API maturity, not
  because it's out of scope permanently.
- **Vite 8 / Rolldown.** See §6 Phase 4 — deferred as its own follow-up given the scale
  of the bundler-architecture change.

---

## 9. Appendix — verification commands used for this research

Re-run these (with current package names/versions) before executing any phase above; do
not trust the numbers in this document without re-checking, especially for `typescript`
and `vite`, both of which shipped major versions in the months immediately preceding this
writeup.

```bash
# Current vs. latest
npm view <package> version                       # latest on npm
npm view <package>@<current-range> version        # currently-resolvable version

# Peer/runtime coupling
npm view <package>@<version> peerDependencies
npm view <package>@<version> dependencies.vite    # vitest majors before ~1.x don't use peerDependencies for this
npm view <package>@<version> engines

# Wails-specific
gh issue view <n> --repo wailsapp/wails --json title,body,state,comments

# Local verification gates (already defined in this repo)
make verify        # lint + test + frontend-build + codegen-check + build
make frontend-test  # vitest run — 78 tests across 10 files as of this writing
wails dev           # manual smoke test: confirm WebView connects, HMR works
wails build         # manual smoke test: build succeeds, app launches, core flows work
```
