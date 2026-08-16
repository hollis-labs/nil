import { describe, expect, it } from "vitest";
import { parseQuery } from "./query";

// Search critical path — query + filter parsing.
//
// `parseQuery` is the pure function App.tsx's runSearch() calls to turn the
// raw search-bar string into structured filters (contexts/projects/tags/
// priority/due/status, plus their negated forms) before building the
// store.SearchRequest sent to the backend. It's also reused to seed a new
// item's default taxonomy from the active search/tab query (App.tsx's
// `defaultNewTodoFilters`). Being a pure function with no Wails/DOM
// dependency, this is unit-tested directly rather than through the UI.

describe("parseQuery", () => {
  it("extracts positive tags, contexts, and projects", () => {
    const result = parseQuery("#work @home +todoapp");
    expect(result.tags).toEqual(["work"]);
    expect(result.contexts).toEqual(["home"]);
    expect(result.projects).toEqual(["todoapp"]);
    expect(result.keywords).toEqual([]);
  });

  it("extracts negative tags, contexts, and projects", () => {
    const result = parseQuery("-#work -@home -+todoapp");
    expect(result.negativeTags).toEqual(["work"]);
    expect(result.negativeContexts).toEqual(["home"]);
    expect(result.negativeProjects).toEqual(["todoapp"]);
  });

  it("treats a bare leading dash as a negative keyword, except -completed", () => {
    const result = parseQuery("-urgent -completed");
    expect(result.negativeKeywords).toEqual(["urgent"]);
    expect(result.flags.completed).toBe(false);
  });

  it("parses priority filters with pri:/priority: aliases, case-insensitively", () => {
    expect(parseQuery("pri:A").priority).toEqual(["A"]);
    expect(parseQuery("priority:b").priority).toEqual(["B"]);
    expect(parseQuery("PRI:c").priority).toEqual(["C"]);
  });

  it("does not treat a malformed priority token as a filter", () => {
    const result = parseQuery("pri:AA");
    expect(result.priority).toBeUndefined();
    expect(result.keywords).toEqual(["pri:AA"]);
  });

  it("parses due/threshold date filters with :, <=, and >= operators", () => {
    const result = parseQuery("due:2026-01-01 due<=2026-02-01 t>=2026-03-01");
    expect(result.due).toEqual([
      { op: ":", date: "2026-01-01" },
      { op: "<=", date: "2026-02-01" },
    ]);
    expect(result.t).toEqual([{ op: ">=", date: "2026-03-01" }]);
  });

  it("parses status: flags and is:archived", () => {
    const result = parseQuery("status:open status:blocked is:archived");
    expect(result.flags.status).toEqual(["open", "blocked"]);
    expect(result.flags.archived).toBe(true);
  });

  it("parses the completed flag", () => {
    expect(parseQuery("completed").flags.completed).toBe(true);
  });

  it("keeps unrecognized tokens as plain keywords", () => {
    const result = parseQuery("review pull request");
    expect(result.keywords).toEqual(["review", "pull", "request"]);
  });

  it("honors quoted phrases as a single token", () => {
    const result = parseQuery('"buy milk" #groceries');
    expect(result.keywords).toEqual(["buy milk"]);
    expect(result.tags).toEqual(["groceries"]);
  });

  it("mixes keywords with taxonomy and priority filters in one query", () => {
    const result = parseQuery("review #work +todoapp @home pri:A -urgent");
    expect(result).toMatchObject({
      keywords: ["review"],
      tags: ["work"],
      projects: ["todoapp"],
      contexts: ["home"],
      priority: ["A"],
      negativeKeywords: ["urgent"],
    });
  });

  it("returns an empty parse for blank input", () => {
    const result = parseQuery("");
    expect(result.keywords).toEqual([]);
    expect(result.tags).toEqual([]);
    expect(result.contexts).toEqual([]);
    expect(result.projects).toEqual([]);
  });
});
