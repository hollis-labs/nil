import { act, renderHook } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { useCaptureEntry } from "./useCaptureEntry";

const fixture = vi.hoisted(() => ({
  listener: undefined as (() => void) | undefined,
  off: vi.fn(),
  ready: vi.fn(),
}));
vi.mock("../../wailsjs/go/main/App", () => ({ CaptureEntryReady: fixture.ready }));
vi.mock("../../wailsjs/runtime/runtime", () => ({
  EventsOn: vi.fn((_name: string, callback: () => void) => {
    fixture.listener = callback;
    return fixture.off;
  }),
}));
it("listens before native readiness and repeated requests focus without toggling closed", () => {
  fixture.ready.mockImplementation(() => {
    fixture.listener?.();
    return Promise.resolve();
  });
  const { result, unmount } = renderHook(() => useCaptureEntry());
  expect(result.current.open).toBe(true);
  expect(result.current.focusRequest).toBe(1);
  act(() => fixture.listener?.());
  expect(result.current.open).toBe(true);
  expect(result.current.focusRequest).toBe(2);
  act(() => result.current.setOpen(false));
  act(() => fixture.listener?.());
  expect(result.current.open).toBe(true);
  unmount();
  expect(fixture.off).toHaveBeenCalledOnce();
});
