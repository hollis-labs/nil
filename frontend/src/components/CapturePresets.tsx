import { useEffect, useState } from "react";
import * as Backend from "../../wailsjs/go/main/App";
import type { capture } from "../../wailsjs/go/models";

// This selector changes native preferences for future drafts only. It never
// edits the current draft or reads/migrates legacy localStorage profiles.
export default function CapturePresets({ onBusy }: { onBusy: (busy: boolean) => void }) {
  const [presets, setPresets] = useState<capture.Preset[]>([]);
  const [selected, setSelected] = useState("");
  const [name, setName] = useState("");
  const [defaults, setDefaults] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState(false);
  const [attempt, setAttempt] = useState(0);

  // biome-ignore lint/correctness/useExhaustiveDependencies: attempt explicitly retries the same native read after failure.
  useEffect(() => {
    let cancelled = false;
    setBusy(true);
    onBusy(true);
    Backend.GetCapturePresets()
      .then((data) => {
        if (cancelled) return;
        setPresets(data.presets);
        setSelected(data.selectedID);
        setError("");
      })
      .catch(() => {
        if (!cancelled) setError("Cannot load presets. Your current draft is unchanged.");
      })
      .finally(() => {
        if (!cancelled) {
          setBusy(false);
          onBusy(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [onBusy, attempt]);

  async function choose(id: string) {
    setBusy(true);
    onBusy(true);
    try {
      await Backend.SelectCapturePreset(id);
      setSelected(id);
      setError("");
    } catch {
      setError("Could not select preset. Your current draft is unchanged.");
    } finally {
      setBusy(false);
      onBusy(false);
    }
  }

  async function save(update: boolean) {
    const current = presets.find((p) => p.id === selected);
    if (update && !current) return;
    setBusy(true);
    onBusy(true);
    try {
      const saved = await Backend.SaveCapturePreset(
        update ? selected : "",
        update ? (current?.revision ?? 0) : 0,
        name,
        defaults,
      );
      if (!update) await Backend.SelectCapturePreset(saved.id);
      const data = await Backend.GetCapturePresets();
      setPresets(data.presets);
      setSelected(data.selectedID);
      setEditing(false);
      setError("");
    } catch {
      setError(
        "Could not save/select preset. Reload presets before trying again; your draft is unchanged.",
      );
    } finally {
      setBusy(false);
      onBusy(false);
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <label htmlFor="capture-preset">Preset for new drafts</label>
        <select
          id="capture-preset"
          value={selected}
          disabled={busy}
          onChange={(e) => void choose(e.target.value)}
        >
          <option value="">No preset</option>
          {presets.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name} · revision {p.revision}
            </option>
          ))}
        </select>
        <button
          type="button"
          disabled={busy}
          onClick={() => {
            const p = presets.find((item) => item.id === selected);
            setName(p?.name ?? "");
            setDefaults(p?.quickAdd ?? "");
            setEditing(true);
          }}
        >
          Edit presets
        </button>
      </div>
      {editing && (
        <fieldset disabled={busy}>
          <legend>Capture preset</legend>
          <label htmlFor="capture-preset-name">Preset name</label>
          <input id="capture-preset-name" value={name} onChange={(e) => setName(e.target.value)} />
          <label htmlFor="capture-preset-defaults">
            Default title, tags, projects, contexts, priority and dates
          </label>
          <input
            id="capture-preset-defaults"
            value={defaults}
            placeholder="(B) +project #tag @desk"
            onChange={(e) => setDefaults(e.target.value)}
          />
          <button type="button" disabled={!name.trim()} onClick={() => void save(false)}>
            Save and select new preset
          </button>
          <button
            type="button"
            disabled={!selected || !name.trim()}
            onClick={() => void save(true)}
          >
            Update selected preset
          </button>
          <button type="button" onClick={() => setEditing(false)}>
            Cancel preset edit
          </button>
        </fieldset>
      )}
      {error && (
        <p role="alert">
          {error}{" "}
          <button type="button" disabled={busy} onClick={() => setAttempt((n) => n + 1)}>
            Reload presets
          </button>
        </p>
      )}
      <small>
        Manual selection applies only when you create a new draft. Existing drafts keep their own
        defaults and removals.
      </small>
    </div>
  );
}
