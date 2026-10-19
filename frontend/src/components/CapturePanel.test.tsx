import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import CapturePanel from "./CapturePanel";

const backend = vi.hoisted(() => ({
  ListCaptureDrafts: vi.fn(),
  GetCaptureContextEvidence: vi.fn(),
  CreateCaptureDraft: vi.fn(),
  SaveCaptureContent: vi.fn(),
  SearchCaptureReferences: vi.fn(),
  GetCapturePresets: vi.fn(),
  SelectCapturePreset: vi.fn(),
  SaveCapturePreset: vi.fn(),
  QueueCaptureDraft: vi.fn(),
}));
vi.mock("../../wailsjs/go/main/App", () => backend);
const draft = {
  captureID: "fixture-capture-0001",
  revision: 1,
  quickAdd: "",
  markdown: "original",
  createdAt: "2026-10-19",
};

describe("capture custody", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    backend.GetCaptureContextEvidence.mockResolvedValue({
      state: "unavailable",
      code: "not_sampled_at_invocation",
    });
    backend.ListCaptureDrafts.mockResolvedValue([{ ...draft }]);
    backend.GetCapturePresets.mockResolvedValue({ presets: [], selectedID: "" });
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

  it("freezes a local queue operation and leaves its source available to copy", async () => {
    backend.QueueCaptureDraft.mockResolvedValue({
      schemaVersion: 1,
      draft: { ...draft },
      content: draft.markdown,
      submittedAt: "2026-10-19",
    });
    render(<CapturePanel open onClose={() => {}} />);
    const editor = await screen.findByDisplayValue("original");
    fireEvent.keyDown(editor, { key: "Enter", ctrlKey: true });
    await waitFor(() => expect(backend.QueueCaptureDraft).toHaveBeenCalledWith(draft.captureID, 1));
    await waitFor(() => expect(editor).toHaveAttribute("readonly"));
    expect(editor).not.toBeDisabled();
    expect(editor).toHaveValue("original");
    expect(screen.getByRole("status")).toHaveTextContent(
      "Queued locally; not saved to Fragments Engine",
    );
    fireEvent.change(editor, { target: { value: "changed after queue" } });
    expect(backend.SaveCaptureContent).not.toHaveBeenCalled();
    fireEvent.keyDown(editor, { key: "Enter", ctrlKey: true });
    expect(backend.QueueCaptureDraft).toHaveBeenCalledOnce();
  });

  it("reconciles lost local queue responses under the original identity", async () => {
    backend.QueueCaptureDraft.mockRejectedValueOnce(new Error("response lost"));
    backend.ListCaptureDrafts.mockResolvedValueOnce([{ ...draft }]).mockResolvedValueOnce([
      { ...draft, localQueued: true },
    ]);
    render(<CapturePanel open onClose={() => {}} />);
    const editor = await screen.findByDisplayValue("original");
    fireEvent.keyDown(editor, { key: "Enter", ctrlKey: true });
    await waitFor(() => expect(editor).toHaveAttribute("readonly"));
    expect(backend.QueueCaptureDraft).toHaveBeenCalledOnce();
    expect(backend.CreateCaptureDraft).not.toHaveBeenCalled();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("changes future preset selection without rewriting an existing draft or removed tags", async () => {
    const p1 = { id: "p1", revision: 1, name: "Work", quickAdd: "#removed @desk" };
    const p2 = { id: "p2", revision: 3, name: "Home", quickAdd: "#home @home" };
    backend.GetCapturePresets.mockResolvedValue({ presets: [p1, p2], selectedID: "p1" });
    backend.SelectCapturePreset.mockResolvedValue(undefined);
    backend.ListCaptureDrafts.mockResolvedValue([
      { ...draft, quickAdd: "@desk", snapshot: { preset: p1, context: { state: "unavailable" } } },
    ]);
    render(<CapturePanel open onClose={() => {}} />);
    await screen.findByDisplayValue("original");
    await screen.findByText("Home · revision 3");
    fireEvent.change(screen.getByLabelText("Preset for new drafts"), { target: { value: "p2" } });
    await waitFor(() => expect(backend.SelectCapturePreset).toHaveBeenCalledWith("p2"));
    expect(screen.getByLabelText("Quick Add title and defaults")).toHaveValue("@desk");
    expect(screen.getByText(/Draft preset:/)).toHaveTextContent("Work · revision 1");
    expect(backend.SaveCaptureContent).not.toHaveBeenCalled();
  });

  it("keeps draft switching locked when autosave finishes before the queue reply", async () => {
    let finishSave: ((value: unknown) => void) | undefined,
      finishQueue: ((value: unknown) => void) | undefined;
    backend.SaveCaptureContent.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishSave = resolve;
        }),
    );
    backend.QueueCaptureDraft.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishQueue = resolve;
        }),
    );
    const onClose = vi.fn();
    render(<CapturePanel open onClose={onClose} />);
    const editor = await screen.findByDisplayValue("original");
    fireEvent.change(editor, { target: { value: "edited" } });
    await waitFor(() => expect(backend.SaveCaptureContent).toHaveBeenCalledOnce());
    fireEvent.keyDown(editor, { key: "Enter", ctrlKey: true });
    finishSave?.({ ...draft, markdown: "edited", revision: 2 });
    await waitFor(() => expect(backend.QueueCaptureDraft).toHaveBeenCalledWith(draft.captureID, 2));
    expect(screen.getByText("New draft")).toBeDisabled();
    expect(screen.getByLabelText("Recover draft")).toBeDisabled();
    expect(screen.getByText("Close capture")).toBeDisabled();
    fireEvent.click(screen.getByText("Close capture"));
    fireEvent.keyDown(editor, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(screen.getByText("New draft"));
    expect(backend.CreateCaptureDraft).not.toHaveBeenCalled();
    finishQueue?.({ draft: { ...draft, markdown: "edited", revision: 2 } });
    await waitFor(() => expect(editor).toHaveAttribute("readonly"));
    expect(editor).toHaveValue("edited");
  });

  it("reconciles a lost queue reply for the submitted ID after an attempted draft switch", async () => {
    const other = { ...draft, captureID: "other-capture-0002", markdown: "other draft" };
    let rejectQueue: ((error: Error) => void) | undefined;
    backend.ListCaptureDrafts.mockResolvedValueOnce([{ ...draft }, other]).mockResolvedValueOnce([
      { ...draft, localQueued: true },
      other,
    ]);
    backend.QueueCaptureDraft.mockImplementationOnce(
      () =>
        new Promise((_resolve, reject) => {
          rejectQueue = reject;
        }),
    );
    render(<CapturePanel open onClose={() => {}} />);
    const editor = await screen.findByDisplayValue("original");
    fireEvent.keyDown(editor, { key: "Enter", ctrlKey: true });
    await waitFor(() => expect(backend.QueueCaptureDraft).toHaveBeenCalledWith(draft.captureID, 1));
    fireEvent.change(screen.getByLabelText("Recover draft"), {
      target: { value: other.captureID },
    });
    rejectQueue?.(new Error("reply lost"));
    await waitFor(() => expect(editor).toHaveAttribute("readonly"));
    expect(screen.getByLabelText("Recover draft")).toHaveValue(draft.captureID);
    expect(editor).toHaveValue("original");
    expect(backend.QueueCaptureDraft).toHaveBeenCalledOnce();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("prevents old-draft edits from leaking into a pending new draft", async () => {
    let finish: ((value: unknown) => void) | undefined;
    backend.CreateCaptureDraft.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    render(<CapturePanel open onClose={() => {}} />);
    const editor = await screen.findByDisplayValue("original");
    await waitFor(() => expect(screen.getByText("New draft")).not.toBeDisabled());
    fireEvent.click(screen.getByText("New draft"));
    await waitFor(() => expect(backend.CreateCaptureDraft).toHaveBeenCalledOnce());
    expect(editor).toBeDisabled();
    fireEvent.change(editor, { target: { value: "old edit" } });
    expect(backend.SaveCaptureContent).not.toHaveBeenCalled();
    finish?.({ ...draft, captureID: "new-capture-0002", markdown: "", quickAdd: "#preset" });
    await screen.findByDisplayValue("#preset");
    expect(screen.getByLabelText("Markdown")).toHaveValue("");
    expect(backend.SaveCaptureContent).not.toHaveBeenCalled();
  });

  it("locks uncertain queue results and retries the original identity", async () => {
    backend.ListCaptureDrafts.mockResolvedValueOnce([{ ...draft }]).mockRejectedValueOnce(
      new Error("storage unavailable"),
    );
    backend.QueueCaptureDraft.mockRejectedValueOnce(new Error("reply lost")).mockResolvedValueOnce({
      draft: { ...draft },
    });
    render(<CapturePanel open onClose={() => {}} />);
    const editor = await screen.findByDisplayValue("original");
    fireEvent.keyDown(editor, { key: "Enter", ctrlKey: true });
    await screen.findByText("Retry local queue");
    expect(editor).toHaveAttribute("readonly");
    expect(screen.getByText("New draft")).toBeDisabled();
    fireEvent.change(editor, { target: { value: "unsafe replacement" } });
    expect(backend.SaveCaptureContent).not.toHaveBeenCalled();
    fireEvent.click(screen.getByText("Retry local queue"));
    await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
    expect(backend.QueueCaptureDraft.mock.calls).toEqual([
      [draft.captureID, 1],
      [draft.captureID, 1],
    ]);
    expect(backend.CreateCaptureDraft).not.toHaveBeenCalled();
  });

  it("binds preset editing to its captured identity and revision", async () => {
    const p1 = { id: "p1", revision: 1, name: "Work", quickAdd: "#work" };
    const p2 = { id: "p2", revision: 7, name: "Home", quickAdd: "#home" };
    backend.GetCapturePresets.mockResolvedValue({ presets: [p1, p2], selectedID: "p1" });
    backend.SaveCapturePreset.mockResolvedValue({ ...p1, revision: 2, name: "Renamed" });
    render(<CapturePanel open onClose={() => {}} />);
    await screen.findByDisplayValue("original");
    await waitFor(() => expect(screen.getByText("Edit presets")).not.toBeDisabled());
    fireEvent.click(screen.getByText("Edit presets"));
    expect(screen.getByLabelText("Preset for new drafts")).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Preset for new drafts"), { target: { value: "p2" } });
    fireEvent.change(screen.getByLabelText("Preset name"), { target: { value: "Renamed" } });
    fireEvent.click(screen.getByText("Update preset"));
    await waitFor(() =>
      expect(backend.SaveCapturePreset).toHaveBeenCalledWith("p1", 1, "Renamed", "#work"),
    );
    expect(backend.SelectCapturePreset).not.toHaveBeenCalled();
    expect(backend.SaveCaptureContent).not.toHaveBeenCalled();
  });

  it("retains failed edits, prevents closing, and retries the same identity", async () => {
    const onClose = vi.fn();
    backend.SaveCaptureContent.mockRejectedValueOnce(
      new Error("disk unavailable"),
    ).mockResolvedValueOnce({ ...draft, revision: 2, markdown: "unsaved" });
    render(<CapturePanel open onClose={onClose} />);
    const editor = await screen.findByDisplayValue("original");
    fireEvent.change(editor, { target: { value: "unsaved" } });
    await screen.findByRole("alert");
    expect(editor).toHaveValue("unsaved");
    expect(screen.getByText("Close capture")).toBeDisabled();
    fireEvent.click(screen.getByText("Retry save"));
    await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
    expect(backend.SaveCaptureContent.mock.calls).toEqual([
      [draft.captureID, 1, "unsaved", ""],
      [draft.captureID, 1, "unsaved", ""],
    ]);
    fireEvent.click(screen.getByText("Close capture"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("serializes rapid edits using the last persisted revision", async () => {
    let finish: ((value: unknown) => void) | undefined;
    backend.SaveCaptureContent.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    ).mockResolvedValueOnce({ ...draft, revision: 3, markdown: "second" });
    render(<CapturePanel open onClose={() => {}} />);
    const editor = await screen.findByDisplayValue("original");
    fireEvent.change(editor, { target: { value: "first" } });
    fireEvent.change(editor, { target: { value: "second" } });
    await waitFor(() => expect(backend.SaveCaptureContent).toHaveBeenCalledTimes(1));
    finish?.({ ...draft, revision: 2, markdown: "first" });
    await waitFor(() => expect(backend.SaveCaptureContent).toHaveBeenCalledTimes(2));
    expect(backend.SaveCaptureContent.mock.calls[1]).toEqual([draft.captureID, 2, "second", ""]);
    expect(editor).toHaveValue("second");
  });
  it("repeated native invocation refocuses the same unsaved draft without replacing source or selection", async () => {
    backend.SaveCaptureContent.mockImplementation(() => new Promise(() => {}));
    const onClose = vi.fn();
    const { rerender } = render(<CapturePanel open onClose={onClose} focusRequest={0} />);
    await waitFor(() => expect(screen.getByLabelText("Markdown")).not.toBeDisabled());
    const source = screen.getByLabelText("Markdown") as HTMLTextAreaElement;
    fireEvent.change(source, { target: { value: "unsaved source" } });
    source.setSelectionRange(2, 5);
    screen.getByLabelText("Quick Add title and defaults").focus();
    rerender(<CapturePanel open onClose={onClose} focusRequest={1} />);
    await waitFor(() => expect(source).toHaveFocus());
    expect(source).toHaveValue("unsaved source");
    expect(source.selectionStart).toBe(2);
    expect(source.selectionEnd).toBe(5);
    expect(backend.CreateCaptureDraft).not.toHaveBeenCalled();
    expect(backend.QueueCaptureDraft).not.toHaveBeenCalled();
    expect(backend.SaveCaptureContent).toHaveBeenCalledTimes(1);
  });
  it.each([
    "Link label",
    "Link URL",
  ])("repeated invocation preserves unsaved link editor %s and its selection", async (fieldName) => {
    backend.SaveCaptureContent.mockImplementation(() => new Promise(() => {}));
    const onClose = vi.fn();
    const { rerender } = render(<CapturePanel open onClose={onClose} focusRequest={0} />);
    await waitFor(() => expect(screen.getByLabelText("Markdown")).not.toBeDisabled());
    const source = screen.getByLabelText("Markdown") as HTMLTextAreaElement;
    fireEvent.change(source, { target: { value: "unsaved source" } });
    source.setSelectionRange(0, 7);
    fireEvent.click(screen.getByRole("button", { name: "Edit link" }));
    fireEvent.change(screen.getByLabelText("Link label"), { target: { value: "unsaved label" } });
    fireEvent.change(screen.getByLabelText("Link URL"), {
      target: { value: "https://fixture.example/unsaved" },
    });
    const field = screen.getByLabelText(fieldName) as HTMLInputElement;
    field.focus();
    field.setSelectionRange(2, 5);
    fireEvent.compositionStart(field);
    rerender(<CapturePanel open onClose={onClose} focusRequest={1} />);
    await act(async () => {
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
    });
    await waitFor(() => expect(field).toHaveFocus());
    expect(field.selectionStart).toBe(2);
    expect(field.selectionEnd).toBe(5);
    expect(screen.getByLabelText("Link label")).toHaveValue("unsaved label");
    expect(screen.getByLabelText("Link URL")).toHaveValue("https://fixture.example/unsaved");
    expect(source).toHaveValue("unsaved source");
    fireEvent.compositionEnd(field);
    expect(backend.CreateCaptureDraft).not.toHaveBeenCalled();
    expect(backend.QueueCaptureDraft).not.toHaveBeenCalled();
    expect(backend.SaveCaptureContent).toHaveBeenCalledTimes(1);
  });
  it.each([
    "Markdown",
    "Quick Add title and defaults",
  ])("repeated invocation keeps %s IME composition active without submitting", async (fieldName) => {
    backend.SaveCaptureContent.mockImplementation(() => new Promise(() => {}));
    const onClose = vi.fn();
    const { rerender } = render(<CapturePanel open onClose={onClose} focusRequest={0} />);
    await waitFor(() => expect(screen.getByLabelText("Markdown")).not.toBeDisabled());
    const field = screen.getByLabelText(fieldName) as HTMLInputElement;
    field.focus();
    fireEvent.compositionStart(field);
    fireEvent.change(field, { target: { value: "未確定の入力" } });
    field.setSelectionRange(1, 3);
    rerender(<CapturePanel open onClose={onClose} focusRequest={1} />);
    await act(async () => {
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
    });
    await waitFor(() => expect(field).toHaveFocus());
    expect(field).toHaveValue("未確定の入力");
    expect(field.selectionStart).toBe(1);
    expect(field.selectionEnd).toBe(3);
    fireEvent.keyDown(field, { key: "Enter", ctrlKey: true, isComposing: true, keyCode: 229 });
    expect(backend.QueueCaptureDraft).not.toHaveBeenCalled();
    expect(backend.CreateCaptureDraft).not.toHaveBeenCalled();
    fireEvent.compositionEnd(field);
  });

  it("refocuses enabled link label instead of disabled Markdown after a menu invocation", async () => {
    const onClose = vi.fn();
    const { rerender } = render(<CapturePanel open onClose={onClose} focusRequest={0} />);
    await waitFor(() => expect(screen.getByLabelText("Markdown")).not.toBeDisabled());
    fireEvent.click(screen.getByRole("button", { name: "Edit link" }));
    screen.getByRole("button", { name: "Cancel link" }).focus();
    expect(screen.getByLabelText("Markdown")).toBeDisabled();
    rerender(<CapturePanel open onClose={onClose} focusRequest={1} />);
    await waitFor(() => expect(screen.getByLabelText("Link label")).toHaveFocus());
    expect(backend.CreateCaptureDraft).not.toHaveBeenCalled();
    expect(backend.QueueCaptureDraft).not.toHaveBeenCalled();
  });
});
