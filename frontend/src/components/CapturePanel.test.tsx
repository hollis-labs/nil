import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import CapturePanel from "./CapturePanel";

const backend = vi.hoisted(() => ({
  ListCaptureDrafts: vi.fn(),
  CreateCaptureDraft: vi.fn(),
  SaveCaptureDraft: vi.fn(),
}));
vi.mock("../../wailsjs/go/main/App", () => backend);
const draft = {
  captureID: "fixture-capture-0001",
  revision: 1,
  markdown: "original",
  createdAt: "2026-10-19",
};

describe("capture custody", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    backend.ListCaptureDrafts.mockResolvedValue([{ ...draft }]);
  });

  it.each([
    "list",
    "create",
  ])("allows retry and dismissal after initial %s failure", async (failure) => {
    const onClose = vi.fn();
    if (failure === "list") {
      backend.ListCaptureDrafts.mockRejectedValueOnce(
        new Error("unavailable"),
      ).mockResolvedValueOnce([{ ...draft }]);
    } else {
      backend.ListCaptureDrafts.mockResolvedValue([]);
      backend.CreateCaptureDraft.mockRejectedValueOnce(
        new Error("disk full"),
      ).mockResolvedValueOnce({ ...draft });
    }
    render(<CapturePanel open onClose={onClose} />);
    await screen.findByRole("alert");
    expect(screen.getByText("Close capture")).not.toBeDisabled();
    fireEvent.click(screen.getByText("Close capture"));
    expect(onClose).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText("Retry loading drafts"));
    await waitFor(() => expect(screen.getByLabelText("Markdown")).toHaveValue("original"));
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("retains failed edits, prevents closing, and retries the same identity", async () => {
    const onClose = vi.fn();
    backend.SaveCaptureDraft.mockRejectedValueOnce(
      new Error("disk unavailable"),
    ).mockResolvedValueOnce({ ...draft, revision: 2, markdown: "unsaved" });
    render(<CapturePanel open onClose={onClose} />);
    const editor = await screen.findByLabelText("Markdown");
    await waitFor(() => expect(editor).toHaveValue("original"));
    fireEvent.change(editor, { target: { value: "unsaved" } });
    await screen.findByRole("alert");
    expect(editor).toHaveValue("unsaved");
    expect(screen.getByText("Close capture")).toBeDisabled();
    fireEvent.click(screen.getByText("Retry save"));
    await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
    expect(backend.SaveCaptureDraft.mock.calls).toEqual([
      [draft.captureID, 1, "unsaved"],
      [draft.captureID, 1, "unsaved"],
    ]);
    fireEvent.click(screen.getByText("Close capture"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("serializes rapid edits using the last persisted revision", async () => {
    let finish: ((value: unknown) => void) | undefined;
    backend.SaveCaptureDraft.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    ).mockResolvedValueOnce({ ...draft, revision: 3, markdown: "second" });
    render(<CapturePanel open onClose={() => {}} />);
    const editor = await screen.findByLabelText("Markdown");
    await waitFor(() => expect(editor).toHaveValue("original"));
    fireEvent.change(editor, { target: { value: "first" } });
    fireEvent.change(editor, { target: { value: "second" } });
    await waitFor(() => expect(backend.SaveCaptureDraft).toHaveBeenCalledTimes(1));
    finish?.({ ...draft, revision: 2, markdown: "first" });
    await waitFor(() => expect(backend.SaveCaptureDraft).toHaveBeenCalledTimes(2));
    expect(backend.SaveCaptureDraft.mock.calls[1]).toEqual([draft.captureID, 2, "second"]);
    expect(editor).toHaveValue("second");
  });
});
