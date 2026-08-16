// Vitest global test setup. Registered via `test.setupFiles` in vite.config.ts.
// Extends `expect` with the jest-dom DOM matchers (toBeInTheDocument, etc.)
// for every test file, with matching TypeScript types.
import "@testing-library/jest-dom/vitest";

// @testing-library/react normally auto-registers `afterEach(cleanup)` to
// unmount components and reset the DOM between tests, but it only does so
// when it detects a global `afterEach` — which requires `test.globals: true`
// in the Vitest config. This project's vite.config.ts does not set that (test
// files import `afterEach`/`describe`/`it` explicitly from "vitest" instead),
// so auto-cleanup never registers and every `render()` call across `it()`
// blocks in the same file accumulates onto `document.body` unmounted,
// breaking any test file with more than one render. Register it explicitly
// here instead.
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

afterEach(() => {
  cleanup();
});

// jsdom doesn't implement Element.scrollIntoView (a long-standing, deliberate
// jsdom gap — layout isn't simulated). Several components call it directly in
// effects that run on mount (VaultSwitcher, InboxView's keyboard-navigation
// scroll-into-view effects), which throws and fails the test with an
// unrelated "not a function" error otherwise. Stub it as a no-op, matching
// the standard workaround for this well-known jsdom limitation.
if (typeof Element.prototype.scrollIntoView !== "function") {
  Element.prototype.scrollIntoView = () => {};
}

// Work around a Node/Vitest-version interaction that leaves `localStorage`
// non-functional under jsdom in this environment: Node 22+ ships its own
// experimental global `localStorage` (inert — throws/returns undefined
// without the `--localstorage-file` flag). This Vitest version's jsdom
// environment setup only copies a jsdom window property onto the global
// scope when Node doesn't already define a global of the same name; since
// Node now pre-defines `localStorage`, jsdom's own working implementation
// never gets copied over, and every `localStorage.getItem/setItem` call
// silently fails. Several of this app's critical paths (settings, session
// context, task templates) persist through plain `localStorage` calls, so
// this must work for their tests to exercise real behavior rather than
// failing for an environment reason unrelated to the code under test.
// Detect the broken case and install a minimal, spec-compatible in-memory
// Storage polyfill so `localStorage` behaves normally for every test file.
function localStorageIsUsable(): boolean {
  try {
    const ls = (globalThis as { localStorage?: Storage }).localStorage;
    if (!ls) return false;
    const probeKey = "__nil_test_localstorage_probe__";
    ls.setItem(probeKey, "1");
    const ok = ls.getItem(probeKey) === "1";
    ls.removeItem(probeKey);
    return ok;
  } catch {
    return false;
  }
}

if (!localStorageIsUsable()) {
  class MemoryStorage implements Storage {
    private store = new Map<string, string>();

    get length(): number {
      return this.store.size;
    }

    clear(): void {
      this.store.clear();
    }

    getItem(key: string): string | null {
      return this.store.get(key) ?? null;
    }

    key(index: number): string | null {
      return Array.from(this.store.keys())[index] ?? null;
    }

    removeItem(key: string): void {
      this.store.delete(key);
    }

    setItem(key: string, value: string): void {
      this.store.set(key, String(value));
    }
  }

  Object.defineProperty(globalThis, "localStorage", {
    value: new MemoryStorage(),
    configurable: true,
    writable: true,
  });
}
