import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import CaptureShortcutSettings from "./CaptureShortcutSettings";

const backend = vi.hoisted(() => ({ GetCaptureEntryStatus: vi.fn(), SetCaptureShortcut: vi.fn() }));
vi.mock("../../wailsjs/go/main/App", () => backend);
const shortcut = {
  enabled: false,
  key: "I",
  command: true,
  control: false,
  option: false,
  shift: false,
};
beforeEach(() => {
  vi.resetAllMocks();
  backend.GetCaptureEntryStatus.mockResolvedValue({
    shortcut,
    available: true,
    registered: false,
    error: "",
  });
});
it("does not register until explicit enable and apply", async () => {
  render(<CaptureShortcutSettings />);
  const enabled = await screen.findByLabelText("Enable global capture shortcut");
  expect(enabled).not.toBeChecked();
  expect(backend.SetCaptureShortcut).not.toHaveBeenCalled();
  fireEvent.click(enabled);
  expect(backend.SetCaptureShortcut).not.toHaveBeenCalled();
  backend.SetCaptureShortcut.mockResolvedValue({
    shortcut: { ...shortcut, enabled: true },
    available: true,
    registered: true,
    error: "",
  });
  fireEvent.click(screen.getByRole("button", { name: "Apply shortcut" }));
  await waitFor(() =>
    expect(backend.SetCaptureShortcut).toHaveBeenCalledWith({ ...shortcut, enabled: true }),
  );
  await screen.findByText(/Global shortcut registered/);
});
it("keeps actual prior status after conflict and locks registration while pending", async () => {
  render(<CaptureShortcutSettings />);
  await screen.findByLabelText("Enable global capture shortcut");
  let reject: ((error: Error) => void) | undefined;
  backend.SetCaptureShortcut.mockImplementation(
    () =>
      new Promise((_resolve, fail) => {
        reject = fail;
      }),
  );
  fireEvent.click(screen.getByLabelText("Enable global capture shortcut"));
  fireEvent.click(screen.getByRole("button", { name: "Apply shortcut" }));
  expect(screen.getByRole("button", { name: "Apply shortcut" })).toBeDisabled();
  reject?.(new Error("shortcut belongs to another app"));
  await screen.findByText(/shortcut belongs to another app/);
  expect(screen.getByText(/Global shortcut not registered/)).toBeInTheDocument();
  expect(backend.SetCaptureShortcut).toHaveBeenCalledTimes(1);
});
