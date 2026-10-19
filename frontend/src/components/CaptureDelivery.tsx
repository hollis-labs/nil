import { useCallback, useEffect, useRef, useState } from "react";
import * as Backend from "../../wailsjs/go/main/App";
import type { broker, capture } from "../../wailsjs/go/models";

export const deliveryStorage = {
  accounts: () => Backend.GetCaptureAccounts(),
  list: () => Backend.ListCaptureDeliveries(),
  select: (id: string) => Backend.SelectCaptureAccount(id),
  pair: (origin: string, code: string) => Backend.PairCaptureBroker(origin, code),
  renew: () => Backend.RenewCaptureAccount(),
  evidence: () => Backend.GetCaptureEvidence(),
  run: (id: string, action: string, ack: boolean) => Backend.RunCaptureDelivery(id, action, ack),
  remote: (before: number) => Backend.ListRemoteCaptureStatuses(before),
};

export function receiptLabel(s: broker.Status): string {
  if (s.state === "saved") {
    return s.fragmentID && s.fragmentRevisionID
      ? "Confirmed in Fragments Engine"
      : "Saved status without confirmed FE identifiers";
  }
  return (
    {
      queued: "Accepted by Basalisk; awaiting FE",
      needs_attention: "Needs attention",
      discarded: "Delivery stopped; FE data retained",
      local_stopped: "Stopped locally; never sent",
    }[s.state] ?? s.state
  );
}

// Mount only when explicitly opened. Loading local custody never probes a server.
export default function CaptureDelivery() {
  const [accounts, setAccounts] = useState<capture.AccountSelection | null>(null);
  const [items, setItems] = useState<capture.DeliveryView[]>([]);
  const [origin, setOrigin] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const locked = useRef(false);
  const [poll, setPoll] = useState(false);
  const [ack, setAck] = useState<Record<string, boolean>>({});
  const [evidence, setEvidence] = useState<capture.SetupEvidence | null>(null);
  const [remote, setRemote] = useState<broker.Status[]>([]);
  const [cursor, setCursor] = useState(0);
  const [remoteLoaded, setRemoteLoaded] = useState(false);
  const load = useCallback(async () => {
    const [selection, deliveries] = await Promise.all([
      deliveryStorage.accounts(),
      deliveryStorage.list(),
    ]);
    setAccounts(selection);
    setItems(deliveries);
  }, []);
  const operate = useCallback(
    async (work: () => Promise<unknown>) => {
      if (locked.current) return;
      locked.current = true;
      setBusy(true);
      setError("");
      try {
        await work();
      } catch (e) {
        setError(String(e));
      } finally {
        try {
          await load();
        } catch (e) {
          setError(String(e));
        }
        locked.current = false;
        setBusy(false);
      }
    },
    [load],
  );
  useEffect(() => {
    void load().catch((e) => setError(String(e)));
  }, [load]);
  useEffect(() => {
    if (!poll) return;
    const timer = window.setInterval(() => {
      void operate(async () => {
        const local = await deliveryStorage.list();
        for (const item of local) {
          if (
            item.accountID === accounts?.selectedID &&
            ["queued", "needs_attention"].includes(item.state)
          ) {
            await deliveryStorage.run(item.captureID, "refresh", false);
          }
        }
      });
    }, 5000);
    return () => window.clearInterval(timer);
  }, [poll, accounts?.selectedID, operate]);
  const selected = accounts?.accounts.find((a) => a.id === accounts.selectedID);
  return (
    <section aria-label="Basalisk capture delivery">
      <h3>Basalisk delivery</h3>
      <p>
        Pair Nil independently with Basalisk’s Capture app profile. Local drafts remain available
        offline.
      </p>
      <label>
        Nil pairing{" "}
        <select
          aria-label="Nil pairing"
          disabled={busy}
          value={accounts?.selectedID ?? ""}
          onChange={(e) => {
            const id = e.target.value;
            setEvidence(null);
            setRemote([]);
            setRemoteLoaded(false);
            setCursor(0);
            setPoll(false);
            void operate(() => deliveryStorage.select(id));
          }}
        >
          <option value="" disabled>
            Choose pairing
          </option>
          {accounts?.accounts.map((a) => (
            <option key={a.id} value={a.id}>
              {a.label || a.deviceID} — {a.origin}
            </option>
          ))}
        </select>
      </label>
      {selected && (
        <p>
          Device {selected.deviceID}; expires {selected.expiresAt}. Existing operations stay with
          their original pairing.
        </p>
      )}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const oneUseCode = code;
          setCode("");
          setEvidence(null);
          setRemote([]);
          setRemoteLoaded(false);
          setPoll(false);
          void operate(() => deliveryStorage.pair(origin, oneUseCode));
        }}
      >
        <label>
          Basalisk origin{" "}
          <input
            value={origin}
            onChange={(e) => setOrigin(e.target.value)}
            disabled={busy}
            placeholder="https://basalisk.example"
          />
        </label>
        <label>
          One-use Capture app code{" "}
          <input
            type="password"
            autoComplete="off"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            disabled={busy}
          />
        </label>
        <button type="submit" disabled={busy || !origin || !code}>
          Pair Nil
        </button>
      </form>
      <button
        type="button"
        disabled={busy || !selected}
        onClick={() => void operate(() => deliveryStorage.renew())}
      >
        Renew selected pairing
      </button>
      <button
        type="button"
        disabled={busy || !selected}
        onClick={() => {
          setEvidence(null);
          void operate(async () => setEvidence(await deliveryStorage.evidence()));
        }}
      >
        Read setup evidence
      </button>
      {evidence && (
        <div role="status">
          <p>
            Recorded evidence at {evidence.readiness.generatedAt || "request time"}; valid for{" "}
            {evidence.readiness.validForSeconds || 0} seconds. This does not test upstream
            reachability or routing/enrichment completion.
          </p>
          <p>
            Metadata v1:{" "}
            {evidence.capabilities.captureMetadataVersions?.includes(1)
              ? "supported"
              : "server update required"}
            . Context: {evidence.capabilities.captureContext.allowed ? "allowed" : "unavailable"}.
            Recovery v1:{" "}
            {evidence.capabilities.captureRecovery.version === 1
              ? "supported"
              : "server update required"}
            .
          </p>
          {evidence.readinessSupported ? (
            evidence.readiness.apps.map((app) => (
              <p key={app.deviceID}>
                Permission {app.authorization}; binding {app.connection.state}; worker{" "}
                {app.worker.state} observed {app.worker.observedAt}. Queue {app.queue.queued},
                attention {app.queue.needsAttention}, recorded saved {app.queue.saved}.
                {app.lastConfirmed && (
                  <span>
                    {" "}
                    Historical FE receipt {app.lastConfirmed.captureID}: fragment{" "}
                    {app.lastConfirmed.fragmentID || "unavailable"}, revision{" "}
                    {app.lastConfirmed.fragmentRevisionID || "unavailable"}, recorded{" "}
                    {app.lastConfirmed.updatedAt}.
                  </span>
                )}
              </p>
            ))
          ) : (
            <p>Readiness evidence requires a server update.</p>
          )}
        </div>
      )}
      <label>
        <input
          type="checkbox"
          checked={poll}
          disabled={!selected}
          onChange={(e) => setPoll(e.target.checked)}
        />
        Poll pending accepted captures every five seconds while this panel is open
      </label>
      <button type="button" disabled={busy} onClick={() => void operate(async () => {})}>
        Reload local outbox
      </button>
      <p>
        Captures remain local until sent here. Accepted by Basalisk is pending delivery; only a
        saved receipt with FE identifiers confirms an FE copy.
      </p>
      <ul>
        {items.map((item) => {
          const wrongAccount = !!item.accountID && item.accountID !== accounts?.selectedID;
          const terminal = ["saved", "discarded", "local_stopped"].includes(item.state);
          return (
            <li key={item.captureID}>
              <p>
                {item.captureID}:{" "}
                {item.confirmed
                  ? "Confirmed in Fragments Engine"
                  : item.state === "local_queued"
                    ? "Queued locally; acceptance unconfirmed"
                    : item.state === "local_stopped"
                      ? "Stopped locally; never sent"
                      : receiptLabel(item.status)}
                . {item.status.code || item.code}
              </p>
              <p>
                Original context: {item.context.state} {item.context.code} {item.context.draftAt}
              </p>
              {item.confirmed && (
                <p>
                  FE fragment {item.status.fragmentID}; revision {item.status.fragmentRevisionID}
                </p>
              )}
              {wrongAccount && <p>Select the original pairing to reconcile this capture.</p>}
              {item.status.mayHaveSent && (
                <label>
                  <input
                    type="checkbox"
                    checked={ack[item.captureID] ?? false}
                    onChange={(e) =>
                      setAck((old) => ({ ...old, [item.captureID]: e.target.checked }))
                    }
                  />
                  An FE copy may already exist. Stopping never deletes FE data.
                </label>
              )}
              {(["send", "refresh", "retry", "stop"] as const).map((action) => (
                <button
                  type="button"
                  key={action}
                  disabled={
                    busy ||
                    wrongAccount ||
                    (terminal && action !== "refresh") ||
                    (!selected && action !== "stop") ||
                    (action === "stop" && item.status.mayHaveSent && !ack[item.captureID])
                  }
                  onClick={() =>
                    void operate(() =>
                      deliveryStorage.run(item.captureID, action, ack[item.captureID] ?? false),
                    )
                  }
                >
                  {action === "send"
                    ? "Send / reconcile original"
                    : action === "refresh"
                      ? "Refresh receipt"
                      : action === "retry"
                        ? "Retry delivery"
                        : "Stop delivery"}
                </button>
              ))}
            </li>
          );
        })}
      </ul>
      <button
        type="button"
        disabled={busy || !selected || (remoteLoaded && cursor === 0)}
        onClick={() =>
          void operate(async () => {
            const page = await deliveryStorage.remote(cursor);
            setRemote((old) => [...old, ...page.items]);
            setCursor(page.nextBefore);
            setRemoteLoaded(true);
          })
        }
      >
        Read next page of this device’s receipts
      </button>
      {remote.length > 0 && (
        <>
          <p>
            Remote receipts contain no Markdown bodies; they cannot restore missing local drafts.
          </p>
          <ul>
            {remote.map((s) => (
              <li key={s.captureID}>
                {s.captureID}: {receiptLabel(s)}
              </li>
            ))}
          </ul>
        </>
      )}
      {error && (
        <p role="alert">{error} Local content and original operation identity are retained.</p>
      )}
    </section>
  );
}
