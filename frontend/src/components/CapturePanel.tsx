import { useCallback, useEffect, useRef, useState } from "react";
import * as Backend from "../../wailsjs/go/main/App";
import type { capture } from "../../wailsjs/go/models";
import MarkdownComposer from "./MarkdownComposer";

// All persistence stays behind this boundary; the source editor has no native,
// credential, vault, or network dependencies.
export const captureStorage = {
  list: () => Backend.ListCaptureDrafts(),
  create: () => Backend.CreateCaptureDraft(),
  save: (id: string, revision: number, markdown: string, quickAdd: string) =>
    Backend.SaveCaptureContent(id, revision, markdown, quickAdd),
  search: (query: string) => Backend.SearchCaptureReferences(query),
};

export default function CapturePanel({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [drafts, setDrafts] = useState<capture.Draft[]>([]);
  const [draft, setDraft] = useState<capture.Draft | null>(null);
  const [source, setSource] = useState("");
  const [quickAdd, setQuickAdd] = useState("");
  const sourceRef = useRef("");
  const quickRef = useRef("");
  const panel = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState("Loading drafts…");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [loadAttempt, setLoadAttempt] = useState(0);
  const current = useRef<capture.Draft | null>(null);
  const queue = useRef<Promise<void>>(Promise.resolve());
  const pending = useRef(0);

  const select = useCallback((d: capture.Draft) => {
    current.current = d;
    setDraft(d);
    setSource(d.markdown);
    sourceRef.current = d.markdown;
    quickRef.current = d.quickAdd ?? "";
    setQuickAdd(d.quickAdd ?? "");
    setError("");
    setStatus("Stored locally");
  }, []);

  // biome-ignore lint/correctness/useExhaustiveDependencies: loadAttempt explicitly retries failed bootstrap with unchanged inputs.
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
  }, [open, select, loadAttempt]);

  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement;
    return () => {
      if (previous instanceof HTMLElement) previous.focus();
    };
  }, [open]);

  function save(markdown: string, quickInput = quickRef.current) {
    sourceRef.current = markdown;
    quickRef.current = quickInput;
    setSource(markdown);
    setQuickAdd(quickInput);
    setStatus("Saving locally…");
    setBusy(true);
    pending.current++;
    queue.current = queue.current
      .then(async () => {
        const d = current.current;
        if (!d) return;
        const saved = await captureStorage.save(d.captureID, d.revision, markdown, quickInput);
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
      ref={panel}
      onKeyDown={(e) => {
        e.stopPropagation();
        if (e.key === "Tab") {
          const elements = panel.current?.querySelectorAll<HTMLElement>(
            "button:not(:disabled),input:not(:disabled),textarea:not(:disabled),select:not(:disabled)",
          );
          const first = elements?.[0],
            last = elements?.[elements.length - 1];
          if (e.shiftKey && document.activeElement === first) {
            e.preventDefault();
            last?.focus();
          } else if (!e.shiftKey && document.activeElement === last) {
            e.preventDefault();
            first?.focus();
          }
        }
      }}
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
      <label htmlFor="capture-quick">Quick Add title and defaults</label>
      <input
        id="capture-quick"
        value={quickAdd}
        disabled={!draft}
        placeholder="(B) Follow up +project #tag @desk due:2026-10-20"
        onChange={(e) => save(sourceRef.current, e.target.value)}
      />
      <MarkdownComposer
        key={draft?.captureID ?? "loading"}
        value={source}
        onChange={save}
        disabled={!draft}
        searchReferences={captureStorage.search}
        onSubmit={() => save(sourceRef.current)}
      />
      <button type="button" disabled={!draft || busy} onClick={() => save(sourceRef.current)}>
        Keep draft locally
      </button>
      <p role="status">
        {busy ? "Saving locally…" : status}. Delivery to Fragments Engine is not configured.
      </p>
      {error && (
        <p role="alert">
          {error}{" "}
          <button
            type="button"
            disabled={busy}
            onClick={() => (draft ? save(source) : setLoadAttempt((attempt) => attempt + 1))}
          >
            {draft ? "Retry save" : "Retry loading drafts"}
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
