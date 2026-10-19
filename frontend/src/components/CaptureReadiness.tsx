import { useEffect, useState } from "react";
import type { capture } from "../../wailsjs/go/models";

export type EvidenceObservation = {
  value: capture.SetupEvidence;
  receivedAt: number;
  receivedWall: number;
  refreshFailed: boolean;
};

function fresh(observation: EvidenceObservation, now: number, sourceTime: string, ttl: number) {
  const source = Date.parse(sourceTime);
  const elapsed = now - observation.receivedAt;
  const ageAtReceipt = observation.receivedWall - source;
  return (
    !observation.refreshFailed &&
    Number.isFinite(source) &&
    ttl > 0 &&
    elapsed >= 0 &&
    ageAtReceipt >= 0 &&
    ageAtReceipt + elapsed < ttl * 1000
  );
}

export default function CaptureReadiness({ observation }: { observation: EvidenceObservation }) {
  const [tick, setNow] = useState(() => performance.now());
  useEffect(() => {
    setNow(performance.now());
    const timer = window.setInterval(() => setNow(performance.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  const now = Math.max(tick, performance.now());
  const e = observation.value;
  const current =
    e.readinessSupported &&
    fresh(observation, now, e.readiness.generatedAt, e.readiness.validForSeconds);
  return (
    <div role="status">
      <p>
        {current ? "Recent local setup evidence" : "Stale last-known setup evidence"} for{" "}
        {e.account.deviceID}; recorded {e.readiness.generatedAt || "time unavailable"}, TTL{" "}
        {e.readiness.validForSeconds || 0} seconds.{" "}
        {observation.refreshFailed ? "Refresh failed; prior evidence retained." : ""} This does not
        test upstream reachability or routing/enrichment completion.
      </p>
      <p>
        Recorded capability: metadata v1{" "}
        {e.capabilities.captureMetadataVersions?.includes(1)
          ? "supported"
          : "server update required"}
        ; context {e.capabilities.captureContext.allowed ? "allowed" : "unavailable"}; recovery v1{" "}
        {e.capabilities.captureRecovery.version === 1 ? "supported" : "server update required"}.
      </p>
      {e.readinessSupported ? (
        e.readiness.apps.map((app) => {
          const workerCurrent =
            current && fresh(observation, now, app.worker.observedAt, app.worker.validForSeconds);
          return (
            <div key={app.deviceID}>
              <p>
                {current ? "Recent recorded" : "Stale recorded"} permission {app.authorization}.
                binding {app.connection.state}. Worker evidence {workerCurrent ? "recent" : "stale"}
                : recorded {app.worker.state} at {app.worker.observedAt || "unknown time"}, TTL{" "}
                {app.worker.validForSeconds} seconds. Recorded queue {app.queue.queued}, attention{" "}
                {app.queue.needsAttention}, saved {app.queue.saved}.
              </p>
              {app.lastConfirmed && (
                <p>
                  Historical FE receipt {app.lastConfirmed.captureID}: fragment{" "}
                  {app.lastConfirmed.fragmentID || "unavailable"}, revision{" "}
                  {app.lastConfirmed.fragmentRevisionID || "unavailable"}, recorded{" "}
                  {app.lastConfirmed.updatedAt}. A historical receipt does not establish current
                  binding, worker state or reachability.
                </p>
              )}
            </div>
          );
        })
      ) : (
        <p>Readiness evidence requires a server update.</p>
      )}
    </div>
  );
}
