import { beforeEach, describe, expect, it, vi } from "vitest";
import { deleteTemplate, getTemplates, saveTemplate, updateTemplate } from "./templates";

// Settings persistence critical path — localStorage round-trip for the
// `nil.taskTemplates` key (see CLAUDE.md's "localStorage Keys" list).
// EditItemModal's TemplatePickerCombobox / TemplateSaveDialog are thin UI
// wrappers over these read/write helpers; the persistence logic itself has
// no Wails dependency, so it's unit-tested directly.

beforeEach(() => {
  localStorage.clear();
});

describe("task templates (nil.taskTemplates)", () => {
  it("returns an empty array when nothing is stored", () => {
    expect(getTemplates()).toEqual([]);
  });

  it("saves a template, assigning an id and createdAt, and persists it", () => {
    const saved = saveTemplate({
      name: "Bug triage",
      contexts: ["office"],
      projects: ["nil"],
      tags: ["bug"],
      priority: "A",
    });

    expect(saved.id).toBeTruthy();
    expect(saved.createdAt).toBeTruthy();

    const stored = JSON.parse(localStorage.getItem("nil.taskTemplates")!);
    expect(stored).toHaveLength(1);
    expect(stored[0]).toMatchObject({ name: "Bug triage", tags: ["bug"] });
    expect(getTemplates()).toEqual([saved]);
  });

  it("updates an existing template in place", () => {
    const saved = saveTemplate({ name: "Weekly review", contexts: [], projects: [], tags: [] });
    updateTemplate(saved.id, { name: "Weekly review v2", projects: ["nil"] });

    const templates = getTemplates();
    expect(templates).toHaveLength(1);
    expect(templates[0]).toMatchObject({
      id: saved.id,
      name: "Weekly review v2",
      projects: ["nil"],
    });
  });

  it("leaves templates untouched when updating an id that doesn't exist", () => {
    saveTemplate({ name: "Existing", contexts: [], projects: [], tags: [] });
    updateTemplate("nonexistent-id", { name: "Should not apply" });

    const templates = getTemplates();
    expect(templates).toHaveLength(1);
    expect(templates[0]!.name).toBe("Existing");
  });

  it("deletes a template", () => {
    // saveTemplate ids are Date.now().toString() — force distinct timestamps
    // so the two templates in this test don't collide onto the same id.
    const nowSpy = vi.spyOn(Date, "now");
    nowSpy.mockReturnValueOnce(1000);
    const saved = saveTemplate({ name: "Temp", contexts: [], projects: [], tags: [] });
    nowSpy.mockReturnValueOnce(2000);
    saveTemplate({ name: "Keep", contexts: [], projects: [], tags: [] });
    nowSpy.mockRestore();

    deleteTemplate(saved.id);

    const templates = getTemplates();
    expect(templates).toHaveLength(1);
    expect(templates[0]!.name).toBe("Keep");
  });

  it("recovers gracefully from corrupted JSON", () => {
    localStorage.setItem("nil.taskTemplates", "{not valid json");
    expect(getTemplates()).toEqual([]);
  });

  it("recovers gracefully when stored value isn't an array", () => {
    localStorage.setItem("nil.taskTemplates", JSON.stringify({ not: "an array" }));
    expect(getTemplates()).toEqual([]);
  });
});
