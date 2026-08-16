import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { CopyrightFooter } from "./CopyrightFooter";

describe("CopyrightFooter", () => {
  it("renders branding and version, and opens the about modal on click", () => {
    render(<CopyrightFooter version="1.2.3" buildDate="2026-08-16" />);

    expect(screen.getByText("© HOLLIS LABS")).toBeInTheDocument();
    expect(screen.getByText(/v1\.2\.3/)).toBeInTheDocument();

    // The about modal is not mounted until the branding is clicked.
    expect(screen.queryByText("NIL")).not.toBeInTheDocument();

    fireEvent.click(screen.getByText("© HOLLIS LABS"));

    expect(screen.getByText("NIL")).toBeInTheDocument();
    expect(
      screen.getByText(/Optimize for speed, precision planning/)
    ).toBeInTheDocument();
  });
});
