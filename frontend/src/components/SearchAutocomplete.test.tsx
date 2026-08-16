import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import * as React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as Backend from "../../wailsjs/go/main/App";
import SearchAutocomplete from "./SearchAutocomplete";

// Search critical path — query-typing UI behavior.
//
// SearchAutocomplete is the actual search-bar input (App.tsx wires it as
// `value={query} onChange={setQuery} onSearch={handleInputSubmit}`). It
// loads the taxonomy via Backend.GetFilters and offers @/+/# suggestions
// against the last word being typed, and triggers the search on Enter. The
// filter/parse semantics of the query string itself are covered separately
// by lib/query.test.ts's parseQuery tests; this test covers the
// taxonomy-aware typeahead and search-trigger behavior, mocking only the
// one Wails call this component makes.

vi.mock("../../wailsjs/go/main/App", () => ({
  GetFilters: vi.fn(),
}));

const mockGetFilters = vi.mocked(Backend.GetFilters);

beforeEach(() => {
  mockGetFilters.mockReset();
  mockGetFilters.mockResolvedValue({
    projects: ["todoapp", "taxes"],
    contexts: ["home", "office"],
    tags: ["urgent", "someday"],
  });
});

function Controlled({ onSearch }: { onSearch: () => void }) {
  const [value, setValue] = React.useState("");
  return <SearchAutocomplete value={value} onChange={setValue} onSearch={onSearch} />;
}

describe("SearchAutocomplete", () => {
  it("loads the taxonomy on mount and offers matching suggestions across all three prefixes", async () => {
    const onSearch = vi.fn();
    render(<Controlled onSearch={onSearch} />);
    await waitFor(() => expect(mockGetFilters).toHaveBeenCalledTimes(1));

    const input = screen.getByPlaceholderText(/search…/i);
    fireEvent.change(input, { target: { value: "ho" } });

    await waitFor(() => expect(screen.getByText("@home")).toBeInTheDocument());
  });

  it("scopes suggestions to the typed prefix (+ only suggests projects)", async () => {
    render(<Controlled onSearch={vi.fn()} />);
    await waitFor(() => expect(mockGetFilters).toHaveBeenCalledTimes(1));

    const input = screen.getByPlaceholderText(/search…/i);
    fireEvent.change(input, { target: { value: "+tod" } });

    await waitFor(() => expect(screen.getByText("+todoapp")).toBeInTheDocument());
    expect(screen.queryByText("@home")).not.toBeInTheDocument();
    expect(screen.queryByText("#urgent")).not.toBeInTheDocument();
  });

  it("does not suggest until the current word is at least two characters", async () => {
    render(<Controlled onSearch={vi.fn()} />);
    await waitFor(() => expect(mockGetFilters).toHaveBeenCalledTimes(1));

    // "@" alone is a 1-character word, under the 2-character minimum.
    fireEvent.change(screen.getByPlaceholderText(/search…/i), { target: { value: "@" } });
    expect(screen.queryByText("@home")).not.toBeInTheDocument();
  });

  it("Enter with no suggestions showing triggers onSearch", async () => {
    const onSearch = vi.fn();
    render(<Controlled onSearch={onSearch} />);
    await waitFor(() => expect(mockGetFilters).toHaveBeenCalledTimes(1));

    const input = screen.getByPlaceholderText(/search…/i);
    fireEvent.change(input, { target: { value: "just keywords" } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(onSearch).toHaveBeenCalledTimes(1);
  });

  it("Enter with a suggestion showing selects it instead of searching", async () => {
    const onSearch = vi.fn();
    render(<Controlled onSearch={onSearch} />);
    await waitFor(() => expect(mockGetFilters).toHaveBeenCalledTimes(1));

    const input = screen.getByPlaceholderText(/search…/i);
    fireEvent.change(input, { target: { value: "@ho" } });
    await waitFor(() => expect(screen.getByText("@home")).toBeInTheDocument());

    fireEvent.keyDown(input, { key: "Enter" });

    expect(onSearch).not.toHaveBeenCalled();
  });

  it("clicking a suggestion replaces the last word and appends a trailing space", async () => {
    render(<Controlled onSearch={vi.fn()} />);
    await waitFor(() => expect(mockGetFilters).toHaveBeenCalledTimes(1));

    const input = screen.getByPlaceholderText<HTMLInputElement>(/search…/i);
    fireEvent.change(input, { target: { value: "review @ho" } });
    await waitFor(() => expect(screen.getByText("@home")).toBeInTheDocument());

    fireEvent.click(screen.getByText("@home"));

    expect(input.value).toBe("review @home ");
  });

  it("Escape dismisses the dropdown", async () => {
    render(<Controlled onSearch={vi.fn()} />);
    await waitFor(() => expect(mockGetFilters).toHaveBeenCalledTimes(1));

    const input = screen.getByPlaceholderText(/search…/i);
    fireEvent.change(input, { target: { value: "@ho" } });
    await waitFor(() => expect(screen.getByText("@home")).toBeInTheDocument());

    fireEvent.keyDown(input, { key: "Escape" });
    expect(screen.queryByText("@home")).not.toBeInTheDocument();
  });
});
