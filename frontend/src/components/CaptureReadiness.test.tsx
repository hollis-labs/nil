import { act, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import type { capture } from "../../wailsjs/go/models";
import CaptureReadiness from "./CaptureReadiness";

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});
it("ages snapshot and worker evidence independently using monotonic receipt time", () => {
  vi.useFakeTimers();
  let monotonic = 1000;
  vi.spyOn(performance, "now").mockImplementation(() => monotonic);
  const wall = Date.parse("2026-10-19T00:00:00Z");
  const value = {
    account: { deviceID: "fixture" },
    capabilities: {
      captureMetadataVersions: [1],
      captureContext: { allowed: true },
      captureRecovery: { version: 1 },
    },
    readinessSupported: true,
    readiness: {
      generatedAt: new Date(wall).toISOString(),
      validForSeconds: 30,
      apps: [
        {
          deviceID: "fixture",
          authorization: "active",
          connection: { state: "bound" },
          worker: {
            state: "running",
            observedAt: new Date(wall).toISOString(),
            validForSeconds: 5,
          },
          queue: { queued: 1 },
          lastConfirmed: {
            captureID: "historic",
            fragmentID: "fe-id",
            fragmentRevisionID: "rev-id",
            updatedAt: "yesterday",
          },
        },
      ],
    },
  } as unknown as capture.SetupEvidence;
  render(
    <CaptureReadiness
      observation={{ value, receivedAt: 1000, receivedWall: wall, refreshFailed: false }}
    />,
  );
  expect(screen.getByText(/Recent local setup evidence/)).toBeInTheDocument();
  expect(screen.getByText(/Worker evidence recent/)).toBeInTheDocument();
  // A wall clock adjustment cannot prolong evidence freshness.
  vi.setSystemTime(new Date(wall - 3600000));
  act(() => {
    monotonic = 7000;
    vi.advanceTimersByTime(1000);
  });
  expect(screen.getByText(/Recent local setup evidence/)).toBeInTheDocument();
  expect(screen.getByText(/Worker evidence stale/)).toBeInTheDocument();
  act(() => {
    monotonic = 32000;
    vi.advanceTimersByTime(1000);
  });
  expect(screen.getByText(/Stale last-known setup evidence/)).toBeInTheDocument();
  expect(screen.getByText(/Historical FE receipt historic/)).toBeInTheDocument();
});
