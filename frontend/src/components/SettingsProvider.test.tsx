import { act, renderHook } from "@testing-library/react";
import type * as React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsProvider, useSettings } from "./SettingsContext";

// Settings persistence critical path — localStorage round-trip for
// `nil.settings` (see CLAUDE.md's "localStorage Keys" list), including the
// one-time migrations SettingsProvider runs on first load.
//
// SettingsModal itself (the 1644-line tab-partitioned settings UI) is out of
// scope here — mounting it pulls in GetDatabasePath/GetAPIConfig/GetVaults/
// GetChatConfig Wails calls and theme/vault/chat state that have nothing to
// do with localStorage persistence. SettingsProvider/useSettings are
// exported specifically as the settings-state layer independent of that UI,
// so this test exercises them directly via renderHook rather than mounting
// the modal.

function wrapper({ children }: { children: React.ReactNode }) {
  return <SettingsProvider>{children}</SettingsProvider>;
}

beforeEach(() => {
  localStorage.clear();
});

describe("SettingsProvider / useSettings (nil.settings)", () => {
  it("provides default settings on a fresh install and stamps the notes-tab migration flag", () => {
    const { result } = renderHook(() => useSettings(), { wrapper });

    expect(result.current.settings.showCompleted).toBe(false);
    expect(result.current.settings.tabs.some((t) => t.appMode === "notes")).toBe(true);
    expect(localStorage.getItem("nil.settings.migrated.notesTab")).toBe("1");
  });

  it("setSettings updates state and persists to nil.settings", () => {
    const { result } = renderHook(() => useSettings(), { wrapper });

    act(() => {
      result.current.setSettings({
        ...result.current.settings,
        showCompleted: true,
        defaultInputMode: "search",
      });
    });

    expect(result.current.settings.showCompleted).toBe(true);
    const stored = JSON.parse(localStorage.getItem("nil.settings")!);
    expect(stored.showCompleted).toBe(true);
    expect(stored.defaultInputMode).toBe("search");
  });

  it("migrates the legacy todo.settings key to nil.settings on first load", () => {
    localStorage.setItem(
      "todo.settings",
      JSON.stringify({
        showCompleted: true,
        tabs: [
          { id: "1", label: "All", query: "", appMode: "todos" },
          { id: "2", label: "All", query: "", appMode: "notes" },
        ],
      }),
    );

    const { result } = renderHook(() => useSettings(), { wrapper });

    expect(result.current.settings.showCompleted).toBe(true);
    expect(localStorage.getItem("todo.settings")).toBeNull();
    expect(JSON.parse(localStorage.getItem("nil.settings")!).showCompleted).toBe(true);
  });

  it("does not clobber an existing nil.settings with the legacy key", () => {
    localStorage.setItem("todo.settings", JSON.stringify({ showCompleted: true, tabs: [] }));
    localStorage.setItem(
      "nil.settings",
      JSON.stringify({
        showCompleted: false,
        tabs: [{ id: "1", label: "All", query: "", appMode: "notes" }],
      }),
    );

    const { result } = renderHook(() => useSettings(), { wrapper });

    // nil.settings (the current key) wins; the stale legacy key is left alone.
    expect(result.current.settings.showCompleted).toBe(false);
    expect(localStorage.getItem("todo.settings")).not.toBeNull();
  });

  it("injects a notes-mode All tab exactly once for settings that predate it", () => {
    localStorage.setItem(
      "nil.settings",
      JSON.stringify({
        showCompleted: false,
        tabs: [{ id: "1", label: "All", query: "", appMode: "todos" }],
      }),
    );

    const { result } = renderHook(() => useSettings(), { wrapper });

    const notesTabs = result.current.settings.tabs.filter((t) => t.appMode === "notes");
    expect(notesTabs).toHaveLength(1);
    expect(localStorage.getItem("nil.settings.migrated.notesTab")).toBe("1");
  });

  it("does not inject a duplicate notes tab once the migration has already run", () => {
    localStorage.setItem(
      "nil.settings",
      JSON.stringify({
        showCompleted: false,
        tabs: [
          { id: "1", label: "All", query: "", appMode: "todos" },
          { id: "notes-all", label: "All", query: "", appMode: "notes" },
        ],
      }),
    );
    localStorage.setItem("nil.settings.migrated.notesTab", "1");

    const { result } = renderHook(() => useSettings(), { wrapper });

    const notesTabs = result.current.settings.tabs.filter((t) => t.appMode === "notes");
    expect(notesTabs).toHaveLength(1);
  });

  it("throws when useSettings is called outside a SettingsProvider", () => {
    // React logs the thrown render error via console.error even though the
    // test asserts on it directly — silence that expected noise. (jsdom
    // separately logs an "Uncaught" line for the same error; that one comes
    // from jsdom's own exception reporter rather than console.error/window
    // listeners, so it isn't practical to suppress from here — it's benign
    // stderr noise for an intentionally-thrown, assert-on error.)
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      expect(() => renderHook(() => useSettings())).toThrow(/must be used within SettingsProvider/);
    } finally {
      consoleSpy.mockRestore();
    }
  });
});
