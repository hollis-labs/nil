import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as Backend from "../../wailsjs/go/main/App";
import InboxView from "./InboxView";

// Inbox processing critical path — triage actions: process, archive,
// delete, batch selection.
//
// InboxView is a self-contained component (onClose/onEdit/onProcessed
// props only — no closure coupling to App.tsx's Inner()), so unlike the
// App.tsx-entangled paths this is tested at full component granularity: mount
// the real component and mock only its Wails boundary
// (wailsjs/go/main/App, which lib/backend.ts's getInboxItems() calls
// through to) and lib/backend's updateItem is not used here — InboxView
// calls Backend.ProcessInboxItem/Archive/DeleteItem directly.

vi.mock("../../wailsjs/go/main/App", () => ({
  GetInboxItems: vi.fn(),
  GetVaults: vi.fn(),
  GetActiveVault: vi.fn(),
  ProcessInboxItem: vi.fn(),
  Archive: vi.fn(),
  DeleteItem: vi.fn(),
}));

const mocked = {
  GetInboxItems: vi.mocked(Backend.GetInboxItems),
  GetVaults: vi.mocked(Backend.GetVaults),
  GetActiveVault: vi.mocked(Backend.GetActiveVault),
  ProcessInboxItem: vi.mocked(Backend.ProcessInboxItem),
  Archive: vi.mocked(Backend.Archive),
  DeleteItem: vi.mocked(Backend.DeleteItem),
};

function makeItem(overrides: Record<string, unknown>) {
  return {
    id: 1,
    title: "",
    completed: false,
    archived: false,
    section: "anytime",
    projects: [],
    contexts: [],
    tags: [],
    kind: "todo",
    created_at: "2026-08-01T00:00:00Z",
    ...overrides,
  };
}

const inboxItems = [
  makeItem({ id: 1, title: "Untitled capture" }),
  makeItem({ id: 2, title: "Buy milk" }),
  makeItem({ id: 3, title: "Call dentist" }),
];

beforeEach(() => {
  Object.values(mocked).forEach((fn) => {
    fn.mockReset();
  });
  mocked.GetInboxItems.mockResolvedValue(inboxItems as any);
  mocked.GetVaults.mockResolvedValue([]);
  mocked.GetActiveVault.mockResolvedValue({
    id: "active-vault",
    name: "Main",
    path: "/vaults/main",
    created_at: "",
  } as any);
  mocked.ProcessInboxItem.mockResolvedValue(undefined as any);
  mocked.Archive.mockResolvedValue(undefined as any);
  mocked.DeleteItem.mockResolvedValue(undefined as any);
});

async function renderInbox() {
  const onClose = vi.fn();
  const onEdit = vi.fn();
  const onProcessed = vi.fn();
  render(<InboxView onClose={onClose} onEdit={onEdit} onProcessed={onProcessed} />);
  await waitFor(() => expect(screen.getByText("Buy milk")).toBeInTheDocument());
  return { onClose, onEdit, onProcessed };
}

// Each item row is a per-item <div> whose direct children are the header
// row (checkbox/badge/title/date) and the actions row (Archive/Delete/
// Process buttons) as siblings. The title text sits inside the header row,
// so its closest styled-div ancestor is the header row itself — one more
// `parentElement` reaches the shared row container both are children of.
function rowFor(title: string): HTMLElement {
  return screen.getByText(title).closest("div[style]")!.parentElement as HTMLElement;
}

describe("InboxView", () => {
  it("loads and renders inbox items on mount", async () => {
    await renderInbox();
    expect(screen.getByText("Untitled capture")).toBeInTheDocument();
    expect(screen.getByText("Call dentist")).toBeInTheDocument();
    expect(mocked.GetInboxItems).toHaveBeenCalledTimes(1);
  });

  it("processing a single item calls ProcessInboxItem with the active vault and notifies the parent", async () => {
    const { onProcessed } = await renderInbox();
    const row = rowFor("Buy milk");

    fireEvent.click(within(row).getByText("Process"));

    await waitFor(() => expect(mocked.ProcessInboxItem).toHaveBeenCalledWith(2, "active-vault"));
    expect(onProcessed).toHaveBeenCalled();
  });

  it("archiving a single item calls Archive(id, true) and notifies the parent", async () => {
    const { onProcessed } = await renderInbox();
    const row = rowFor("Call dentist");

    fireEvent.click(within(row).getByText("Archive"));

    await waitFor(() => expect(mocked.Archive).toHaveBeenCalledWith(3, true));
    expect(onProcessed).toHaveBeenCalled();
  });

  it("deleting a single item requires a confirm click before calling DeleteItem", async () => {
    const { onProcessed } = await renderInbox();
    const row = rowFor("Buy milk");

    fireEvent.click(within(row).getByText("Delete"));
    // First click only arms the confirmation — no delete call yet.
    expect(mocked.DeleteItem).not.toHaveBeenCalled();
    expect(within(row).getByText("Confirm?")).toBeInTheDocument();

    fireEvent.click(within(row).getByText("Yes, Delete"));

    await waitFor(() => expect(mocked.DeleteItem).toHaveBeenCalledWith(2));
    expect(onProcessed).toHaveBeenCalled();
  });

  it("supports selecting individual items and batch-processing them", async () => {
    const { onProcessed } = await renderInbox();

    // Select "Buy milk" and "Call dentist" via their row checkboxes.
    fireEvent.click(rowFor("Buy milk").querySelector("svg")!);
    fireEvent.click(rowFor("Call dentist").querySelector("svg")!);

    expect(screen.getByText("2 selected")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Process All"));

    await waitFor(() => expect(mocked.ProcessInboxItem).toHaveBeenCalledWith(2, "active-vault"));
    expect(mocked.ProcessInboxItem).toHaveBeenCalledWith(3, "active-vault");
    expect(mocked.ProcessInboxItem).toHaveBeenCalledTimes(2);
    expect(onProcessed).toHaveBeenCalled();
  });

  it("supports select-all, batch archive, and clearing the selection", async () => {
    const { onProcessed } = await renderInbox();

    // The header checkbox is the only one preceding the search input.
    const checkboxes = screen.getAllByRole("checkbox");
    fireEvent.click(checkboxes[0]!); // select all

    expect(screen.getByText("3 selected")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Archive All"));

    await waitFor(() => expect(mocked.Archive).toHaveBeenCalledTimes(3));
    for (const id of [1, 2, 3]) {
      expect(mocked.Archive).toHaveBeenCalledWith(id, true);
    }
    expect(onProcessed).toHaveBeenCalled();
  });

  it("batch delete requires a confirm step before calling DeleteItem for each selected id", async () => {
    await renderInbox();

    fireEvent.click(rowFor("Buy milk").querySelector("svg")!);
    fireEvent.click(rowFor("Call dentist").querySelector("svg")!);

    fireEvent.click(screen.getByText("Delete All"));
    expect(mocked.DeleteItem).not.toHaveBeenCalled();

    fireEvent.click(screen.getByText(/Confirm Delete 2/));

    await waitFor(() => expect(mocked.DeleteItem).toHaveBeenCalledTimes(2));
    expect(mocked.DeleteItem).toHaveBeenCalledWith(2);
    expect(mocked.DeleteItem).toHaveBeenCalledWith(3);
  });

  it("clicking a row (outside its action buttons) calls onEdit with that item", async () => {
    const { onEdit } = await renderInbox();
    fireEvent.click(screen.getByText("Buy milk"));
    expect(onEdit).toHaveBeenCalledTimes(1);
    expect((onEdit.mock.calls[0]![0] as { id: number }).id).toBe(2);
  });

  it("filters items via the search box (debounced) by re-querying the backend", async () => {
    await renderInbox();
    mocked.GetInboxItems.mockResolvedValueOnce([inboxItems[1]] as any);

    fireEvent.change(screen.getByPlaceholderText("Search inbox…"), { target: { value: "milk" } });

    await waitFor(() => expect(mocked.GetInboxItems).toHaveBeenCalledTimes(2), { timeout: 1000 });
  });
});
