import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { config } from "../../wailsjs/go/models";
import VaultSwitcher from "./VaultSwitcher";

// Vault switching critical path — UI behavior.
//
// VaultSwitcher receives the resolved vault list and active vault id as
// props (App.tsx's `loadVaults` fetches them via Backend.GetVaults /
// Backend.GetActiveVault and stores them in Inner()'s state) and reports
// user intent back through onSwitch/onClose callbacks. That data-in,
// callback-out shape makes it testable without mocking any Wails binding —
// this test renders it directly with fixture vaults and asserts the
// filter/selection/keyboard-navigation behavior a real vault switch
// depends on.
//
// What this does NOT cover: App.tsx's handleVaultSwitch, which is what
// actually calls Backend.SwitchVault, updates `activeVault` state, and
// re-runs the search after onSwitch fires. That orchestration lives inside
// Inner()'s closures (per docs/architecture/ARCHITECTURE.md §10's
// assessment of App.tsx's weak seams) and isn't exercised here — this test
// covers the switcher UI's contract with its caller, not App.tsx's response
// to it.

function makeVault(overrides: Partial<config.Vault>): config.Vault {
  return new config.Vault({
    id: "default-id",
    name: "Default",
    path: "/vaults/default",
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  });
}

const vaults = [
  makeVault({ id: "v1", name: "Personal", path: "/vaults/personal" }),
  makeVault({ id: "v2", name: "Work", path: "/vaults/work" }),
  makeVault({ id: "v3", name: "Archive", path: "/vaults/archive" }),
];

describe("VaultSwitcher", () => {
  it("renders nothing when closed", () => {
    render(
      <VaultSwitcher
        open={false}
        vaults={vaults}
        activeVaultId="v1"
        onSwitch={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(screen.queryByText("Switch Vault")).not.toBeInTheDocument();
  });

  it("lists all vaults and marks the active one when open", () => {
    render(
      <VaultSwitcher
        open={true}
        vaults={vaults}
        activeVaultId="v2"
        onSwitch={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByText("Personal")).toBeInTheDocument();
    expect(screen.getByText("Work")).toBeInTheDocument();
    expect(screen.getByText("Archive")).toBeInTheDocument();
    expect(screen.getByText("active")).toBeInTheDocument();
  });

  it("filters the vault list by name as the user types", () => {
    render(
      <VaultSwitcher
        open={true}
        vaults={vaults}
        activeVaultId="v1"
        onSwitch={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    const filterInput = screen.getByPlaceholderText("Filter vaults…");

    fireEvent.change(filterInput, { target: { value: "wo" } });

    expect(screen.getByText("Work")).toBeInTheDocument();
    expect(screen.queryByText("Personal")).not.toBeInTheDocument();
    expect(screen.queryByText("Archive")).not.toBeInTheDocument();
  });

  it("shows a no-match message when the filter matches nothing", () => {
    render(
      <VaultSwitcher
        open={true}
        vaults={vaults}
        activeVaultId="v1"
        onSwitch={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    fireEvent.change(screen.getByPlaceholderText("Filter vaults…"), {
      target: { value: "nonexistent" },
    });
    expect(screen.getByText(/No vaults match/)).toBeInTheDocument();
  });

  it("clicking a vault row calls onSwitch with that vault", () => {
    const onSwitch = vi.fn();
    render(
      <VaultSwitcher
        open={true}
        vaults={vaults}
        activeVaultId="v1"
        onSwitch={onSwitch}
        onClose={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByText("Work"));

    expect(onSwitch).toHaveBeenCalledTimes(1);
    expect(onSwitch.mock.calls[0]![0]).toMatchObject({ id: "v2", name: "Work" });
  });

  it("clicking the backdrop calls onClose", () => {
    const onClose = vi.fn();
    const { container } = render(
      <VaultSwitcher
        open={true}
        vaults={vaults}
        activeVaultId="v1"
        onSwitch={vi.fn()}
        onClose={onClose}
      />,
    );
    fireEvent.click(container.firstChild as HTMLElement);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("Escape calls onClose via the document-level key handler", () => {
    const onClose = vi.fn();
    render(
      <VaultSwitcher
        open={true}
        vaults={vaults}
        activeVaultId="v1"
        onSwitch={vi.fn()}
        onClose={onClose}
      />,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("ArrowDown then Enter switches to the next vault in the list", () => {
    const onSwitch = vi.fn();
    render(
      <VaultSwitcher
        open={true}
        vaults={vaults}
        activeVaultId="v1"
        onSwitch={onSwitch}
        onClose={vi.fn()}
      />,
    );

    // Focus starts on the first row (Personal); arrow down moves to Work.
    fireEvent.keyDown(document, { key: "ArrowDown" });
    fireEvent.keyDown(document, { key: "Enter" });

    expect(onSwitch).toHaveBeenCalledTimes(1);
    expect(onSwitch.mock.calls[0]![0]).toMatchObject({ id: "v2", name: "Work" });
  });

  it("does not attach the document key handler while closed", () => {
    const onClose = vi.fn();
    render(
      <VaultSwitcher
        open={false}
        vaults={vaults}
        activeVaultId="v1"
        onSwitch={vi.fn()}
        onClose={onClose}
      />,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
  });
});
