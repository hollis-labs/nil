import { useEffect, useState } from "react";
import { CaptureEntryReady } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

export function useCaptureEntry() {
  const [open, setOpen] = useState(false);
  const [focusRequest, setFocusRequest] = useState(0);
  const [error, setError] = useState("");
  useEffect(() => {
    // Subscribe before acknowledging readiness: native early requests are
    // coalesced until this listener exists, including during renderer startup.
    const off = EventsOn("nil:capture-invoke", () => {
      setOpen(true);
      setFocusRequest((value) => value + 1);
    });
    let active = true;
    void CaptureEntryReady().catch(() => {
      if (active) setError("Native capture invocation is unavailable; use Capture Markdown.");
    });
    return () => {
      active = false;
      off();
    };
  }, []);
  return { open, setOpen, focusRequest, error };
}
