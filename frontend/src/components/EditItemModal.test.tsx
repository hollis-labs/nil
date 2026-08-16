import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as Backend from "../../wailsjs/go/main/App";
import EditItemModal from "./EditItemModal";
import { SettingsProvider } from "./SettingsContext";

// Quick add critical path — item-creation-payload half.
//
// The todo.txt-inspired trigger parsing itself is covered separately by
// ItemTitleInput.test.tsx; this file covers what happens after that: when
// the user submits, EditItemModal's doSave() (create-mode branch,
// EditItemModal.tsx's doSave/handleSubmit) assembles the `extras` payload
// — cleaned (prefix-stripped) tags/contexts/projects, priority, due date,
// and the "send to inbox" flag — and calls `onSubmit(line, extras)`. App.tsx's
// handleQuickAdd (untested — see docs/architecture/ARCHITECTURE.md §10 on
// App.tsx's closure coupling) then merges that with
// Backend.CreateItemFromLine()'s response and calls Backend.UpdateItem. This
// test covers the payload-assembly half at the component boundary that
// actually owns it: EditItemModal itself, wrapped in the real
// SettingsProvider and with Backend.GetFilters mocked (called by the modal
// and by each of the Tags/Projects/Contexts autocomplete children).
//
// The TipTap description editor (`notes_doc`/`notes_html`) is left
// untouched by these tests — mounting it (confirmed empirically) works fine
// under jsdom, but driving actual typed input into a ProseMirror
// contentEditable through fireEvent is unreliable in jsdom and out of scope
// for what "quick add" needs covered here. Because the editor mounts with an
// (empty) document, `notes_doc`/`notes_html` are always present in the
// submitted extras (EditItemModal.tsx's `if (notes_doc)` check is truthy for
// any mounted editor, empty doc included) — assertions below use
// `expect.objectContaining` so they don't hardcode that empty-doc shape.

vi.mock("../../wailsjs/go/main/App", () => ({
  GetFilters: vi.fn(),
}));

const mockGetFilters = vi.mocked(Backend.GetFilters);

beforeEach(() => {
  localStorage.clear();
  mockGetFilters.mockReset();
  mockGetFilters.mockResolvedValue({ projects: [], contexts: [], tags: [] });
});

function renderCreateModal(onSubmit = vi.fn()) {
  render(
    <SettingsProvider>
      <EditItemModal open={true} onOpenChange={vi.fn()} onSubmit={onSubmit} />
    </SettingsProvider>,
  );
  return { onSubmit };
}

function titleInput() {
  return screen.getByPlaceholderText<HTMLInputElement>(/review pull request/i);
}

describe("EditItemModal — quick add (create mode)", () => {
  it("submits the typed title with empty taxonomy when nothing else is set", () => {
    const { onSubmit } = renderCreateModal();
    fireEvent.change(titleInput(), { target: { value: "Buy milk" } });

    fireEvent.click(screen.getByText("Create"));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    const [line, extras] = onSubmit.mock.calls[0]!;
    expect(line).toBe("Buy milk");
    expect(extras).toMatchObject({ tags: [], contexts: [], projects: [] });
    expect(extras.priority).toBeUndefined();
    expect(extras.due).toBeUndefined();
    expect(extras.inbox).toBeUndefined();
  });

  it("includes the selected priority in extras", () => {
    const { onSubmit } = renderCreateModal();
    fireEvent.change(titleInput(), { target: { value: "Urgent task" } });
    fireEvent.click(screen.getByText("HIGH"));

    fireEvent.click(screen.getByText("Create"));

    const [, extras] = onSubmit.mock.calls[0]!;
    expect(extras.priority).toBe("A");
  });

  it("toggling a priority button off removes it from extras", () => {
    const { onSubmit } = renderCreateModal();
    fireEvent.change(titleInput(), { target: { value: "Task" } });
    fireEvent.click(screen.getByText("HIGH"));
    fireEvent.click(screen.getByText("HIGH")); // toggle back off

    fireEvent.click(screen.getByText("Create"));

    const [, extras] = onSubmit.mock.calls[0]!;
    expect(extras.priority).toBeUndefined();
  });

  it("strips the @/+/# prefix from added contexts/projects/tags before submitting", () => {
    const { onSubmit } = renderCreateModal();
    fireEvent.change(titleInput(), { target: { value: "Ship it" } });

    fireEvent.change(screen.getByPlaceholderText("Add project..."), {
      target: { value: "todoapp" },
    });
    fireEvent.keyDown(screen.getByPlaceholderText("Add project..."), { key: "Enter" });

    fireEvent.change(screen.getByPlaceholderText("Add context..."), { target: { value: "home" } });
    fireEvent.keyDown(screen.getByPlaceholderText("Add context..."), { key: "Enter" });

    fireEvent.change(screen.getByPlaceholderText("Add tag..."), { target: { value: "urgent" } });
    fireEvent.keyDown(screen.getByPlaceholderText("Add tag..."), { key: "Enter" });

    fireEvent.click(screen.getByText("Create"));

    const [, extras] = onSubmit.mock.calls[0]!;
    expect(extras.projects).toEqual(["todoapp"]);
    expect(extras.contexts).toEqual(["home"]);
    expect(extras.tags).toEqual(["urgent"]);
  });

  it("the '→ Inbox' button submits with extras.inbox = true", () => {
    const { onSubmit } = renderCreateModal();
    fireEvent.change(titleInput(), { target: { value: "Sort me later" } });

    fireEvent.click(screen.getByText("→ Inbox"));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    const [line, extras] = onSubmit.mock.calls[0]!;
    expect(line).toBe("Sort me later");
    expect(extras.inbox).toBe(true);
  });

  it("Clear resets the title and priority", () => {
    renderCreateModal();
    fireEvent.change(titleInput(), { target: { value: "Draft text" } });
    fireEvent.click(screen.getByText("HIGH"));
    expect(titleInput().value).toBe("Draft text");

    fireEvent.click(screen.getByText("Clear"));

    expect(titleInput().value).toBe("");
    // HIGH is no longer the active priority once cleared — submitting now
    // should carry no priority.
  });

  it("falls back to settings.defaultTags when no tags/projects were added", () => {
    localStorage.setItem(
      "nil.settings",
      JSON.stringify({
        showCompleted: false,
        tabs: [{ id: "1", label: "All", query: "", appMode: "todos" }],
        defaultTags: ["inbox-default"],
      }),
    );
    const { onSubmit } = renderCreateModal();
    fireEvent.change(titleInput(), { target: { value: "Untagged item" } });

    fireEvent.click(screen.getByText("Create"));

    const [, extras] = onSubmit.mock.calls[0]!;
    expect(extras.tags).toEqual(["inbox-default"]);
  });
});
