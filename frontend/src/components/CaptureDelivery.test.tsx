import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import CaptureDelivery from "./CaptureDelivery";

const backend = vi.hoisted(() => ({
  GetCaptureAccounts: vi.fn(),
  ListCaptureDeliveries: vi.fn(),
  SelectCaptureAccount: vi.fn(),
  PairCaptureBroker: vi.fn(),
  RenewCaptureAccount: vi.fn(),
  GetCaptureEvidence: vi.fn(),
  RunCaptureDelivery: vi.fn(),
  ListRemoteCaptureStatuses: vi.fn(),
}));
vi.mock("../../wailsjs/go/main/App", () => backend);
const account = {
  id: "account",
  origin: "https://fixture.example",
  deviceID: "device",
  label: "Nil",
  expiresAt: "2026-10-20",
};
const item = {
  captureID: "fixture-capture-0001",
  accountID: "account",
  state: "queued",
  confirmed: false,
  code: "",
  context: { state: "unavailable", code: "not_sampled_at_invocation" },
  status: { state: "queued", mayHaveSent: true },
};
describe("native capture delivery", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    backend.GetCaptureAccounts.mockResolvedValue({ accounts: [account], selectedID: "account" });
    backend.ListCaptureDeliveries.mockResolvedValue([item]);
  });
  it("reads local custody without probing and warns before stopping possible FE data", async () => {
    render(<CaptureDelivery />);
    await screen.findByText(/awaiting FE/);
    expect(backend.GetCaptureEvidence).not.toHaveBeenCalled();
    expect(backend.RunCaptureDelivery).not.toHaveBeenCalled();
    const stop = screen.getByRole("button", { name: "Stop delivery" });
    expect(stop).toBeDisabled();
    fireEvent.click(screen.getByLabelText(/An FE copy may already exist/));
    fireEvent.click(stop);
    await waitFor(() =>
      expect(backend.RunCaptureDelivery).toHaveBeenCalledWith(
        item.captureID,
        "stop",
        true,
        "account",
      ),
    );
  });
  it("clears one-use code before awaiting native exchange and preserves native errors", async () => {
    backend.PairCaptureBroker.mockRejectedValue(new Error("fresh grant required"));
    render(<CaptureDelivery />);
    await screen.findByText(/awaiting FE/);
    fireEvent.change(screen.getByLabelText("Basalisk origin"), {
      target: { value: account.origin },
    });
    const code = screen.getByLabelText("One-use Capture app code");
    fireEvent.change(code, { target: { value: "disposable-code" } });
    fireEvent.click(screen.getByRole("button", { name: "Pair Nil" }));
    expect(code).toHaveValue("");
    await screen.findByText(/fresh grant required/);
    expect(backend.PairCaptureBroker).toHaveBeenCalledWith(account.origin, "disposable-code");
  });
  it("never labels bare saved status confirmed or lets another pairing adopt it", async () => {
    backend.ListCaptureDeliveries.mockResolvedValue([
      {
        ...item,
        accountID: "original",
        state: "saved",
        status: { state: "saved", fragmentID: "", fragmentRevisionID: "" },
      },
    ]);
    render(<CaptureDelivery />);
    await screen.findByText(/Saved status without confirmed FE identifiers/);
    expect(screen.queryByText(/^Confirmed in Fragments Engine/)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send / reconcile original" })).toBeDisabled();
  });
  it("locks all actions while an ambiguous send is pending and retains original identity on error", async () => {
    let fail: ((e: Error) => void) | undefined;
    backend.RunCaptureDelivery.mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          fail = reject;
        }),
    );
    render(<CaptureDelivery />);
    await screen.findByText(/awaiting FE/);
    fireEvent.click(screen.getByRole("button", { name: "Send / reconcile original" }));
    expect(screen.getByRole("button", { name: "Retry delivery" })).toBeDisabled();
    expect(screen.getByRole("combobox", { name: "Nil pairing" })).toBeDisabled();
    fail?.(new Error("acceptance unconfirmed"));
    await screen.findByText(/acceptance unconfirmed/);
    expect(backend.RunCaptureDelivery.mock.calls).toEqual([
      [item.captureID, "send", false, "account"],
    ]);
    expect(backend.PairCaptureBroker).not.toHaveBeenCalled();
  });
  it.each([
    "accounts",
    "deliveries",
  ])("serializes late startup %s with pairing, selection and send", async (pending) => {
    let resolveInitial: ((value: unknown) => void) | undefined;
    const initial = new Promise((resolve) => {
      resolveInitial = resolve;
    });
    if (pending === "accounts") backend.GetCaptureAccounts.mockReturnValueOnce(initial);
    else backend.ListCaptureDeliveries.mockReturnValueOnce(initial);
    render(<CaptureDelivery />);
    expect(screen.getByRole("button", { name: "Pair Nil" })).toBeDisabled();
    expect(screen.getByRole("combobox", { name: "Nil pairing" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Reload local outbox" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Pair Nil" }));
    expect(backend.PairCaptureBroker).not.toHaveBeenCalled();
    resolveInitial?.(
      pending === "accounts" ? { accounts: [account], selectedID: "account" } : [item],
    );
    await screen.findByText(/awaiting FE/);
    const b = { ...account, id: "account-b", deviceID: "device-b" };
    backend.GetCaptureAccounts.mockResolvedValue({
      accounts: [account, b],
      selectedID: "account-b",
    });
    backend.ListCaptureDeliveries.mockResolvedValue([
      { ...item, captureID: "fixture-b", accountID: "", state: "local_queued" },
    ]);
    let finishPair: (() => void) | undefined;
    backend.PairCaptureBroker.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          finishPair = resolve;
        }),
    );
    fireEvent.change(screen.getByLabelText("Basalisk origin"), { target: { value: b.origin } });
    fireEvent.change(screen.getByLabelText("One-use Capture app code"), {
      target: { value: "disposable-b" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Pair Nil" }));
    expect(screen.getByRole("button", { name: "Send / reconcile original" })).toBeDisabled();
    expect(screen.getByRole("combobox", { name: "Nil pairing" })).toBeDisabled();
    finishPair?.();
    await screen.findByText(/fixture-b: Queued locally/);
    expect(screen.getByRole("combobox", { name: "Nil pairing" })).toHaveValue("account-b");
    fireEvent.click(screen.getByRole("button", { name: "Send / reconcile original" }));
    await waitFor(() =>
      expect(backend.RunCaptureDelivery).toHaveBeenCalledWith(
        "fixture-b",
        "send",
        false,
        "account-b",
      ),
    );
  });
  it("retains stale last-known evidence on refresh failure, then clears it on identity change", async () => {
    const now = new Date().toISOString();
    backend.GetCaptureEvidence.mockResolvedValue({
      account,
      capabilities: {
        captureMetadataVersions: [1],
        captureContext: { allowed: true },
        captureRecovery: { version: 1 },
      },
      readinessSupported: true,
      readiness: {
        generatedAt: now,
        validForSeconds: 30,
        apps: [
          {
            deviceID: "device",
            authorization: "active",
            connection: { state: "bound" },
            worker: { state: "running", observedAt: now, validForSeconds: 60 },
            queue: { queued: 1, needsAttention: 0, saved: 1 },
            lastConfirmed: {
              captureID: "historical-id",
              fragmentID: "fe-id",
              fragmentRevisionID: "revision",
              updatedAt: now,
            },
          },
        ],
      },
    });
    render(<CaptureDelivery />);
    await screen.findByText(/awaiting FE/);
    fireEvent.click(screen.getByRole("button", { name: "Read setup evidence" }));
    await screen.findByText(/Recent local setup evidence/);
    backend.GetCaptureEvidence.mockRejectedValueOnce(new Error("offline"));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Read setup evidence" })).not.toBeDisabled(),
    );
    fireEvent.click(screen.getByRole("button", { name: "Read setup evidence" }));
    await screen.findByText(/Refresh failed; prior evidence retained/);
    expect(screen.getByText(/Historical FE receipt historical-id/)).toBeInTheDocument();
    expect(screen.getByText(/Stale recorded.*permission active/)).toBeInTheDocument();
    backend.GetCaptureAccounts.mockResolvedValue({
      accounts: [{ ...account, id: "b" }],
      selectedID: "b",
    });
    fireEvent.click(screen.getByRole("button", { name: "Reload local outbox" }));
    await waitFor(() =>
      expect(screen.queryByText(/Historical FE receipt historical-id/)).not.toBeInTheDocument(),
    );
  });
});
