import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { capture } from "../../wailsjs/go/models";
import MarkdownComposer from "./MarkdownComposer";

const candidate = capture.Suggestion.createFrom({
  reference: {
    relation: "mention",
    provider: "nil",
    instance: "vault-a",
    kind: "item",
    id: "42",
    revision: "r1",
    label: "Plan",
  },
  url: "nil://vault/vault-a/item/42?revision=r1",
});

function setup(source = "") {
  const search = vi.fn().mockResolvedValue([candidate]);
  const submit = vi.fn();
  const changes = vi.fn();
  function Controlled() {
    const [value, setValue] = useState(source);
    return (
      <MarkdownComposer
        value={value}
        onChange={(next) => {
          changes(next);
          setValue(next);
        }}
        disabled={false}
        searchReferences={search}
        onSubmit={submit}
      />
    );
  }
  render(<Controlled />);
  return {
    editor: screen.getByLabelText<HTMLTextAreaElement>("Markdown"),
    search,
    submit,
    changes,
  };
}

describe("Markdown source composer", () => {
  it("formats only the selected source and pastes literal text without normalizing unknown syntax", () => {
    const { editor } = setup("---\nunknown: yes\n---\n\nhello world");
    const from = editor.value.indexOf("hello");
    editor.setSelectionRange(from, from + 5);
    fireEvent.click(screen.getByText("Bold"));
    expect(editor.value).toBe("---\nunknown: yes\n---\n\n**hello** world");
    editor.setSelectionRange(editor.value.length, editor.value.length);
    fireEvent.paste(editor, {
      clipboardData: {
        getData: (type: string) =>
          type === "text/plain" ? "\n<script>raw</script>\n| x | y |" : "",
      },
    });
    expect(editor.value).toBe(
      "---\nunknown: yes\n---\n\n**hello** world\n<script>raw</script>\n| x | y |",
    );
  });

  it("preserves CRLF and unsupported source outside the edited selection", () => {
    const { editor, changes } = setup("---\r\nunknown: yes\r\n---\r\n\r\nhello world");
    const from = editor.value.indexOf("hello");
    editor.setSelectionRange(from, from + 5);
    fireEvent.click(screen.getByText("Bold"));
    expect(changes).toHaveBeenLastCalledWith("---\r\nunknown: yes\r\n---\r\n\r\n**hello** world");
    fireEvent.change(editor, { target: { value: editor.value + "!" } });
    expect(changes).toHaveBeenLastCalledWith("---\r\nunknown: yes\r\n---\r\n\r\n**hello** world!");
  });

  it("keeps Enter/newline and IME input separate from submit", () => {
    const { editor, submit } = setup("日本語");
    fireEvent.keyDown(editor, { key: "Enter" });
    expect(submit).not.toHaveBeenCalled();
    fireEvent.compositionStart(editor);
    fireEvent.keyDown(editor, { key: "Enter", ctrlKey: true });
    expect(submit).not.toHaveBeenCalled();
    fireEvent.compositionEnd(editor);
    fireEvent.keyDown(editor, { key: "Enter", ctrlKey: true });
    expect(submit).toHaveBeenCalledOnce();
  });

  it("keeps plain @context unless a typed suggestion is selected, with menu priority over submit", async () => {
    const { editor, search, submit } = setup();
    fireEvent.change(editor, { target: { value: "@desk", selectionStart: 5, selectionEnd: 5 } });
    await screen.findByRole("option");
    expect(search).toHaveBeenCalledWith("desk");
    expect(editor.value).toBe("@desk");
    fireEvent.keyDown(editor, { key: "Escape" });
    expect(editor.value).toBe("@desk");
    fireEvent.change(editor, { target: { value: "@plan", selectionStart: 5, selectionEnd: 5 } });
    await screen.findByRole("option");
    fireEvent.keyDown(editor, { key: "ArrowDown" });
    fireEvent.keyDown(editor, { key: "Enter" });
    expect(editor.value).toBe("[Plan](nil://vault/vault-a/item/42?revision=r1) ");
    expect(submit).not.toHaveBeenCalled();
    await waitFor(() => expect(document.activeElement).toBe(editor));
  });

  it("locks conflicting source edits during link editing and restores focus on cancel", async () => {
    const original = "before [plan](https://old.test) after";
    const { editor, changes } = setup(original);
    editor.setSelectionRange(12, 12);
    fireEvent.click(screen.getByText("Edit link"));
    expect(editor).toBeDisabled();
    expect(screen.getByText("Bold")).toBeDisabled();
    fireEvent.change(editor, { target: { value: `PREFIX ${original}` } });
    fireEvent.paste(editor, { clipboardData: { getData: () => "PREFIX " } });
    expect(changes).not.toHaveBeenCalled();
    expect(editor.value).toBe(original);
    fireEvent.click(screen.getByText("Cancel link"));
    await waitFor(() => expect(document.activeElement).toBe(editor));
    expect(editor).not.toBeDisabled();
    expect(editor.value).toBe(original);
  });

  it.each([
    "PREFIX before [plan](https://old.test) after",
    "different draft content",
  ])("refuses stale link offsets after external source change to %s", async (next) => {
    const original = "before [plan](https://old.test) after";
    const onChange = vi.fn(),
      search = vi.fn().mockResolvedValue([]),
      onSubmit = vi.fn();
    const props = {
      value: original,
      onChange,
      searchReferences: search,
      onSubmit,
      disabled: false,
    };
    const view = render(<MarkdownComposer {...props} />);
    const editor = screen.getByLabelText<HTMLTextAreaElement>("Markdown");
    editor.setSelectionRange(12, 12);
    fireEvent.click(screen.getByText("Edit link"));
    view.rerender(<MarkdownComposer {...props} value={next} />);
    fireEvent.change(screen.getByLabelText("Link URL"), { target: { value: "https://new.test" } });
    fireEvent.click(screen.getByText("Apply link"));
    expect(onChange).not.toHaveBeenCalled();
    expect(editor.value).toBe(next);
    expect(screen.getByRole("alert")).toHaveTextContent("Draft changed");
    fireEvent.keyDown(screen.getByLabelText("Link URL"), { key: "Escape" });
    await waitFor(() => expect(document.activeElement).toBe(editor));
    expect(editor.value).toBe(next);
  });

  it("edits a supported inline link in place and rejects active URL schemes", () => {
    const { editor } = setup("before [plan](https://old.test) after");
    editor.setSelectionRange(12, 12);
    fireEvent.click(screen.getByText("Edit link"));
    fireEvent.change(screen.getByLabelText("Link URL"), {
      target: { value: "javascript:alert(1)" },
    });
    fireEvent.click(screen.getByText("Apply link"));
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(editor.value).toBe("before [plan](https://old.test) after");
    fireEvent.change(screen.getByLabelText("Link URL"), { target: { value: "https://new.test" } });
    fireEvent.click(screen.getByText("Apply link"));
    expect(editor.value).toBe("before [plan](https://new.test) after");
  });
});
