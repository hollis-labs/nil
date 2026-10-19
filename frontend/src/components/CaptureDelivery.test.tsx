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
      expect(backend.RunCaptureDelivery).toHaveBeenCalledWith(item.captureID, "stop", true),
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
    fail?.(new Error("acceptance unconfirmed"));
    await screen.findByText(/acceptance unconfirmed/);
    expect(backend.RunCaptureDelivery.mock.calls).toEqual([[item.captureID, "send", false]]);
    expect(backend.PairCaptureBroker).not.toHaveBeenCalled();
  });
});
