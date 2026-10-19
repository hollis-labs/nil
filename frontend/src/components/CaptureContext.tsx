import { useEffect, useState } from "react";
import * as Backend from "../../wailsjs/go/main/App";
import type { capture } from "../../wailsjs/go/models";

// Reads native local evidence only; never starts sampling for a recovered draft.
export default function CaptureContext({ captureID }: { captureID: string | undefined }) {
  const [evidence, setEvidence] = useState<capture.ContextEvidence | null>(null);
  useEffect(() => {
    setEvidence(null);
    if (!captureID) return;
    let active = true;
    let timer: number | undefined;
    const read = async () => {
      try {
        const result = await Backend.GetCaptureContextEvidence(captureID);
        if (!active) return;
        setEvidence(result);
        if (result.state === "pending") timer = window.setTimeout(() => void read(), 500);
      } catch {
        if (active) setEvidence(null);
      }
    };
    void read();
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [captureID]);
  return (
    <small>
      Original Basalisk context: {evidence?.state ?? "unavailable"}. {evidence?.code}{" "}
      {evidence?.draftAt}. Sessionless capture is valid; reconnect never replaces the original
      focus.
    </small>
  );
}
