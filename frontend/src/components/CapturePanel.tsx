import { useCallback, useEffect, useRef, useState } from "react";
import * as Backend from "../../wailsjs/go/main/App";
import type { capture } from "../../wailsjs/go/models";

// All persistence stays behind this boundary; the source editor has no native,
// credential, vault, or network dependencies.
export const captureStorage = {
  list: () => Backend.ListCaptureDrafts(),
  create: () => Backend.CreateCaptureDraft(),
  save: (id: string, revision: number, markdown: string) =>
    Backend.SaveCaptureDraft(id, revision, markdown),
};

export default function CapturePanel({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [drafts, setDrafts] = useState<capture.Draft[]>([]);
  const [draft, setDraft] = useState<capture.Draft | null>(null);
  const [source, setSource] = useState("");
  const [status, setStatus] = useState("Loading drafts…");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const current = useRef<capture.Draft | null>(null);
  const queue = useRef<Promise<void>>(Promise.resolve());
  const pending = useRef(0);
  const input = useRef<HTMLTextAreaElement>(null);

  const select = useCallback((d: capture.Draft) => {
    current.current = d;
    setDraft(d);
    setSource(d.markdown);
    setError("");
    setStatus("Stored locally");
  }, []);

  useEffect(() => {
    if (!open || current.current) return;
    let cancelled = false;
    setBusy(true);
    captureStorage
      .list()
      .then(async (items) => {
        const selected = items[0] ?? (await captureStorage.create());
        if (cancelled) return;
        setDrafts(items.length ? items : [selected]);
        select(selected);
      })
      .catch(() => {
        if (!cancelled)
          setError("Cannot open capture storage. Your existing drafts have been kept.");
      })
      .finally(() => {
        if (!cancelled) setBusy(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, select]);

  useEffect(() => {
    if (open && draft) input.current?.focus();
  }, [open, draft]);

  function save(markdown: string) {
    setSource(markdown);
    setStatus("Saving locally…");
    setBusy(true);
    pending.current++;
    queue.current = queue.current
      .then(async () => {
        const d = current.current;
        if (!d) return;
        const saved = await captureStorage.save(d.captureID, d.revision, markdown);
        current.current = saved;
        setDrafts((items) =>
          items.map((item) => (item.captureID === saved.captureID ? saved : item)),
        );
        setError("");
        setStatus("Stored locally");
      })
      .catch(() => {
        setError(
          "Could not store this edit. Keep this window open, copy your Markdown, or retry saving.",
        );
        setStatus("Unsaved changes");
      })
      .finally(() => {
        pending.current--;
        if (!pending.current) setBusy(false);
      });
  }

  async function newDraft() {
    setBusy(true);
    try {
      const created = await captureStorage.create();
      setDrafts((items) => [created, ...items]);
      select(created);
    } catch {
      setError("Could not create a draft. Your current draft has been kept.");
    } finally {
      setBusy(false);
    }
  }

  if (!open) return null;
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="capture-title"
      className="terminal-card"
      style={{
        position: "fixed",
        inset: "8% 10%",
        zIndex: 70,
        padding: 20,
        background: "var(--term-bg)",
        color: "var(--term-fg)",
        border: "1px solid var(--term-border)",
        display: "flex",
        flexDirection: "column",
        gap: 12,
      }}
    >
      <h2 id="capture-title">Capture</h2>
      <div style={{ display: "flex", gap: 8 }}>
        <label htmlFor="capture-drafts">Recover draft</label>
        <select
          id="capture-drafts"
          value={draft?.captureID ?? ""}
          disabled={busy || !!error}
          onChange={(e) => {
            const d = drafts.find((item) => item.captureID === e.target.value);
            if (d) select(d);
          }}
        >
          {drafts.map((d) => (
            <option key={d.captureID} value={d.captureID}>
              {d.markdown.slice(0, 50) || "Empty draft"} · {d.createdAt}
            </option>
          ))}
        </select>
        <button
          type="button"
          className="badge"
          disabled={busy || !!error}
          onClick={() => void newDraft()}
        >
          New draft
        </button>
      </div>
      <label htmlFor="capture-source">Markdown</label>
      <textarea
        id="capture-source"
        ref={input}
        value={source}
        disabled={!draft}
        onChange={(e) => save(e.target.value)}
        style={{
          flex: 1,
          minHeight: 120,
          resize: "none",
          background: "var(--term-panel)",
          color: "var(--term-fg)",
          border: "1px solid var(--term-border)",
          padding: 12,
          fontFamily: "monospace",
        }}
      />
      <p role="status">
        {busy ? "Saving locally…" : status}. Delivery to Fragments Engine is not configured.
      </p>
      {error && (
        <p role="alert">
          {error}{" "}
          <button type="button" disabled={busy || !draft} onClick={() => save(source)}>
            Retry save
          </button>
        </p>
      )}
      <button
        type="button"
        className="badge"
        disabled={busy || (!!error && !!draft)}
        onClick={onClose}
      >
        Close capture
      </button>
    </div>
  );
}
