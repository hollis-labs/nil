import { fireEvent, render, screen } from "@testing-library/react";
import * as React from "react";
import { describe, expect, it, vi } from "vitest";
import ItemTitleInput from "./ItemTitleInput";

// Quick add critical path — capture-syntax parsing half.
//
// Item creation itself is a pass-through: App.tsx's handleQuickAdd sends the
// raw title line to Backend.CreateItemFromLine, and the todo.txt-inspired
// `@context`/`+project`/`#tag` parsing happens entirely server-side in
// parse/line.go — there is no frontend-side re-implementation of that
// parser to unit test. What the frontend *does* own is recognizing those
// same trigger characters live as the user types, to offer autocomplete
// from recently-used contexts/projects/tags (ItemTitleInput, used by
// EditItemModal's title field). That's genuine frontend logic and is what
// this test exercises: trigger detection, suggestion filtering, and
// insertion — a plain-input component with no Wails dependency.
//
// ItemTitleInput is a controlled input whose own insertion logic
// (insertSuggestion) reads from the `value` prop, so it's rendered through a
// small stateful wrapper here (mirroring EditItemModal's real
// `value={line} onChange={setLine}` usage) rather than a bare `vi.fn()` —
// a mock that doesn't feed back into `value` would leave the component
// computing insertions against a stale, empty string.

const recentContexts = ["home", "office", "phone"];
const recentProjects = ["todoapp", "taxes"];
const recentTags = ["urgent", "someday"];

function ControlledItemTitleInput({
  onChange,
  initialValue = "",
}: {
  onChange: (v: string) => void;
  initialValue?: string;
}) {
  const [value, setValue] = React.useState(initialValue);
  return (
    <ItemTitleInput
      value={value}
      onChange={(v) => {
        setValue(v);
        onChange(v);
      }}
      recentContexts={recentContexts}
      recentProjects={recentProjects}
      recentTags={recentTags}
    />
  );
}

function setup(initialValue = "") {
  const onChange = vi.fn();
  render(<ControlledItemTitleInput onChange={onChange} initialValue={initialValue} />);
  const input = screen.getByPlaceholderText<HTMLInputElement>(/review pull request/i);
  return { onChange, input };
}

describe("ItemTitleInput", () => {
  it("shows no suggestions for plain text", () => {
    const { input } = setup();
    fireEvent.change(input, { target: { value: "Buy milk" } });
    expect(screen.queryByText("home")).not.toBeInTheDocument();
  });

  it("suggests recent contexts after typing @ and filters as the query narrows", () => {
    const { input } = setup();
    fireEvent.change(input, { target: { value: "Call @" } });
    expect(screen.getByText("home")).toBeInTheDocument();
    expect(screen.getByText("office")).toBeInTheDocument();
    expect(screen.getByText("phone")).toBeInTheDocument();

    fireEvent.change(input, { target: { value: "Call @ho" } });
    expect(screen.getByText("home")).toBeInTheDocument();
    expect(screen.queryByText("office")).not.toBeInTheDocument();
    expect(screen.queryByText("phone")).not.toBeInTheDocument();
  });

  it("suggests recent projects after + and recent tags after #", () => {
    const { input } = setup();
    fireEvent.change(input, { target: { value: "Ship +tod" } });
    expect(screen.getByText("todoapp")).toBeInTheDocument();
    expect(screen.queryByText("taxes")).not.toBeInTheDocument();

    fireEvent.change(input, { target: { value: "Ship #urg" } });
    expect(screen.getByText("urgent")).toBeInTheDocument();
    expect(screen.queryByText("someday")).not.toBeInTheDocument();
  });

  it("stops offering suggestions once a space follows the trigger", () => {
    const { input } = setup();
    fireEvent.change(input, { target: { value: "Call @home " } });
    expect(screen.queryByText("home")).not.toBeInTheDocument();
  });

  it("clicking a suggestion inserts it with the trigger prefix and a trailing space", () => {
    const { input, onChange } = setup();
    fireEvent.change(input, { target: { value: "Call @ho" } });

    fireEvent.click(screen.getByText("home"));
    expect(onChange).toHaveBeenLastCalledWith("Call @home ");
  });

  it("Tab inserts the currently selected suggestion", () => {
    const { input, onChange } = setup();
    fireEvent.change(input, { target: { value: "Ship +tax" } });
    fireEvent.keyDown(input, { key: "Tab" });
    expect(onChange).toHaveBeenLastCalledWith("Ship +taxes ");
  });

  it("ArrowDown/ArrowUp cycle the selected suggestion before inserting", () => {
    const { input, onChange } = setup();
    // Both projects match an empty post-trigger query.
    fireEvent.change(input, { target: { value: "Ship +" } });
    expect(screen.getByText("todoapp")).toBeInTheDocument();
    expect(screen.getByText("taxes")).toBeInTheDocument();

    fireEvent.keyDown(input, { key: "ArrowDown" }); // move from todoapp -> taxes
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onChange).toHaveBeenLastCalledWith("Ship +taxes ");
  });

  it("Escape dismisses the suggestion dropdown", () => {
    const { input } = setup();
    fireEvent.change(input, { target: { value: "Call @ho" } });
    expect(screen.getByText("home")).toBeInTheDocument();

    fireEvent.keyDown(input, { key: "Escape" });
    expect(screen.queryByText("home")).not.toBeInTheDocument();
  });

  it("forwards every keystroke to onChange regardless of trigger state", () => {
    const { input, onChange } = setup();
    fireEvent.change(input, { target: { value: "Just typing" } });
    expect(onChange).toHaveBeenCalledWith("Just typing");
  });
});
