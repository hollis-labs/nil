import { useEffect, useRef, useState } from "react";
import * as Backend from "../../wailsjs/go/main/App";
import type { entry } from "../../wailsjs/go/models";

export default function CaptureShortcutSettings() {
  const [status, setStatus] = useState<entry.Status | null>(null);
  const [shortcut, setShortcut] = useState<entry.Shortcut | null>(null);
  const [busy, setBusy] = useState(true);
  const locked = useRef(true);
  const [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    void Backend.GetCaptureEntryStatus()
      .then((result) => {
        if (active) {
          setStatus(result);
          setShortcut(result.shortcut);
        }
      })
      .catch(() => {
        if (active) setError("Shortcut settings unavailable; registration has not been changed.");
      })
      .finally(() => {
        if (active) {
          locked.current = false;
          setBusy(false);
        }
      });
    return () => {
      active = false;
    };
  }, []);
  const apply = async () => {
    if (locked.current || !shortcut) return;
    locked.current = true;
    setBusy(true);
    setError("");
    try {
      const result = await Backend.SetCaptureShortcut(shortcut);
      setStatus(result);
      setShortcut(result.shortcut);
    } catch (e) {
      setError(String(e));
      try {
        setStatus(await Backend.GetCaptureEntryStatus());
      } catch {}
    } finally {
      locked.current = false;
      setBusy(false);
    }
  };
  return (
    <section aria-label="Capture invocation settings">
      <p>
        Open Capture from the Capture menu or the Nil status item. The global shortcut is disabled
        by default so Frag can retain Command-I. Enable it explicitly only when you intend Nil to
        own the chosen shortcut.
      </p>
      {status && (
        <p role="status">
          {status.registered ? "Global shortcut registered" : "Global shortcut not registered"}.{" "}
          {status.available
            ? "macOS native entry available."
            : "Global shortcut and status item require macOS."}{" "}
          {status.error}
        </p>
      )}
      {shortcut && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void apply();
          }}
        >
          <label>
            <input
              type="checkbox"
              checked={shortcut.enabled}
              disabled={busy || !status?.available}
              onChange={(e) => setShortcut({ ...shortcut, enabled: e.target.checked })}
            />
            Enable global capture shortcut
          </label>
          <label>
            Shortcut key{" "}
            <select
              value={shortcut.key}
              disabled={busy}
              onChange={(e) => setShortcut({ ...shortcut, key: e.target.value })}
            >
              {[..."ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", "Space"].map((key) => (
                <option key={key}>{key}</option>
              ))}
            </select>
          </label>
          {(["command", "control", "option", "shift"] as const).map((modifier) => (
            <label key={modifier}>
              <input
                type="checkbox"
                checked={shortcut[modifier]}
                disabled={busy}
                onChange={(e) => setShortcut({ ...shortcut, [modifier]: e.target.checked })}
              />
              {modifier}
            </label>
          ))}
          <p>
            Keys use ANSI physical positions. Include Command or Control. Registration conflicts
            leave the prior shortcut in place; no other app is changed.
          </p>
          <button type="submit" disabled={busy}>
            Apply shortcut
          </button>
        </form>
      )}
      {error && <p role="alert">{error}</p>}
    </section>
  );
}
