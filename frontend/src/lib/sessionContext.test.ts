import { beforeEach, describe, expect, it } from "vitest";
import {
  clearActiveSession,
  deleteSessionProfile,
  getActiveSession,
  getSessionProfiles,
  saveSessionProfile,
  setActiveSession,
  updateSessionProfile,
} from "./sessionContext";

// Settings persistence critical path — localStorage round-trip for the
// `nil.sessionProfiles` / `nil.activeSession` keys (see CLAUDE.md's
// "localStorage Keys" list). These are plain read/write helpers with no
// Wails dependency, so they're unit-tested directly against jsdom's real
// localStorage rather than through EditItemModal/SessionContextModal's UI.

beforeEach(() => {
  localStorage.clear();
});

describe("session profiles (nil.sessionProfiles)", () => {
  it("returns an empty array when nothing is stored", () => {
    expect(getSessionProfiles()).toEqual([]);
  });

  it("saves a profile, assigning an id and createdAt, and persists it", () => {
    const saved = saveSessionProfile({
      name: "Deep Work",
      contexts: ["office"],
      projects: ["todoapp"],
      tags: ["focus"],
      priority: "A",
    });

    expect(saved.id).toBeTruthy();
    expect(saved.createdAt).toBeTruthy();

    const stored = JSON.parse(localStorage.getItem("nil.sessionProfiles")!);
    expect(stored).toHaveLength(1);
    expect(stored[0]).toMatchObject({ name: "Deep Work", contexts: ["office"] });

    // Round-trips back out through the reader.
    expect(getSessionProfiles()).toEqual([saved]);
  });

  it("updates an existing profile in place", () => {
    const saved = saveSessionProfile({ name: "Errands", contexts: [], projects: [], tags: [] });
    updateSessionProfile(saved.id, { name: "Errands v2", tags: ["urgent"] });

    const profiles = getSessionProfiles();
    expect(profiles).toHaveLength(1);
    expect(profiles[0]).toMatchObject({ id: saved.id, name: "Errands v2", tags: ["urgent"] });
  });

  it("deleting a profile removes it and clears the active session if it referenced that profile", () => {
    const saved = saveSessionProfile({ name: "Focus", contexts: [], projects: [], tags: [] });
    setActiveSession({ profileId: saved.id, contexts: [], projects: [], tags: [] });

    deleteSessionProfile(saved.id);

    expect(getSessionProfiles()).toEqual([]);
    expect(getActiveSession()).toBeNull();
  });

  it("deleting a profile leaves an unrelated active session untouched", () => {
    const saved = saveSessionProfile({ name: "Focus", contexts: [], projects: [], tags: [] });
    setActiveSession({
      profileId: "some-other-profile",
      contexts: ["home"],
      projects: [],
      tags: [],
    });

    deleteSessionProfile(saved.id);

    expect(getActiveSession()).toMatchObject({ profileId: "some-other-profile" });
  });

  it("recovers gracefully from corrupted JSON", () => {
    localStorage.setItem("nil.sessionProfiles", "{not valid json");
    expect(getSessionProfiles()).toEqual([]);
  });
});

describe("active session (nil.activeSession)", () => {
  it("returns null when nothing is stored", () => {
    expect(getActiveSession()).toBeNull();
  });

  it("round-trips a session through set/get", () => {
    const session = {
      contexts: ["home"],
      projects: ["todoapp"],
      tags: ["urgent"],
      priority: "B" as const,
      useAsFilterTab: true,
    };
    setActiveSession(session);
    expect(getActiveSession()).toEqual(session);
  });

  it("clears the stored session", () => {
    setActiveSession({ contexts: ["home"], projects: [], tags: [] });
    clearActiveSession();
    expect(getActiveSession()).toBeNull();
  });

  it("recovers gracefully from corrupted JSON", () => {
    localStorage.setItem("nil.activeSession", "{not valid json");
    expect(getActiveSession()).toBeNull();
  });
});
