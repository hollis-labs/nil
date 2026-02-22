import * as React from "react";
import { chat } from "../../wailsjs/go/models";
import * as Backend from "../../wailsjs/go/main/App";

type Props = {
  proposal: chat.ActionProposal;
  vaultName?: string;
  onResolved?: (outcome: "executed" | "denied" | "dry_run" | "failed") => void;
};

const ACTION_COLORS: Record<string, string> = {
  create: "var(--term-success, #4ade80)",
  update: "var(--term-warn, #fbbf24)",
  delete: "var(--term-error, #f87171)",
};

const ACTION_LABELS: Record<string, string> = {
  create: "CREATE",
  update: "UPDATE",
  delete: "DELETE",
};

export default function ActionCard({ proposal, vaultName, onResolved }: Props) {
  const [status, setStatus] = React.useState(proposal.status);
  const [busy, setBusy] = React.useState(false);

  // Keep status in sync if prop changes (e.g. parent reload).
  React.useEffect(() => {
    setStatus(proposal.status);
  }, [proposal.status]);

  const resolved = status !== "pending";
  const accentColor = ACTION_COLORS[proposal.action_type] ?? "var(--term-fg)";
  const label = ACTION_LABELS[proposal.action_type] ?? proposal.action_type.toUpperCase();
  const p = proposal.payload ?? {};

  async function handleApprove() {
    if (busy || resolved) return;
    setBusy(true);
    try {
      const result = await Backend.ApproveChatAction(proposal.id);
      const outcome = result?.dry_run ? "dry_run" : "executed";
      setStatus(outcome);
      onResolved?.(outcome);
    } catch (err: any) {
      setStatus("failed");
      onResolved?.("failed");
      console.error("ApproveChatAction failed:", err);
    } finally {
      setBusy(false);
    }
  }

  async function handleDeny() {
    if (busy || resolved) return;
    setBusy(true);
    try {
      await Backend.DenyChatAction(proposal.id);
      setStatus("denied");
      onResolved?.("denied");
    } catch (err: any) {
      console.error("DenyChatAction failed:", err);
    } finally {
      setBusy(false);
    }
  }

  const statusLabel: Record<string, string> = {
    executed: "✓ Executed",
    dry_run: "○ Dry run — not executed",
    denied: "✗ Denied",
    failed: "⚠ Failed",
    approved: "✓ Approved",
  };

  return (
    <div style={{
      marginTop: "8px",
      border: `1px solid ${accentColor}`,
      borderRadius: "6px",
      overflow: "hidden",
      fontSize: "12px",
      fontFamily: "monospace",
    }}>
      {/* Header */}
      <div style={{
        background: accentColor,
        color: "#000",
        padding: "4px 10px",
        display: "flex",
        alignItems: "center",
        gap: "8px",
        fontWeight: 700,
        letterSpacing: "0.05em",
      }}>
        <span>{label}</span>
        <span style={{ opacity: 0.7, fontWeight: 400 }}>{proposal.item_type}</span>
        {vaultName && (
          <span style={{
            marginLeft: "auto",
            opacity: 0.7,
            fontWeight: 400,
            fontSize: "11px",
          }}>
            {vaultName}
          </span>
        )}
      </div>

      {/* Body */}
      <div style={{
        padding: "10px",
        background: "var(--term-panel)",
      }}>
        {proposal.action_type === "delete" ? (
          <DeleteBody payload={p} />
        ) : proposal.action_type === "update" && proposal.diff ? (
          <UpdateBody payload={p} diff={proposal.diff} />
        ) : (
          <CreateBody payload={p} />
        )}

        {/* Controls */}
        <div style={{
          marginTop: "10px",
          display: "flex",
          alignItems: "center",
          gap: "8px",
        }}>
          {resolved ? (
            <span style={{
              fontSize: "11px",
              color: status === "executed" || status === "approved"
                ? "var(--term-success, #4ade80)"
                : status === "denied"
                ? "var(--term-dim)"
                : status === "dry_run"
                ? "var(--term-info, var(--term-fg))"
                : "var(--term-error, #f87171)",
              fontStyle: "italic",
            }}>
              {statusLabel[status] ?? status}
            </span>
          ) : (
            <>
              <button
                onClick={handleApprove}
                disabled={busy}
                style={{
                  padding: "4px 12px",
                  background: "var(--term-success, #4ade80)",
                  color: "#000",
                  border: "none",
                  borderRadius: "4px",
                  cursor: busy ? "not-allowed" : "pointer",
                  fontFamily: "monospace",
                  fontSize: "11px",
                  fontWeight: 700,
                  opacity: busy ? 0.6 : 1,
                }}
              >
                {busy ? "…" : "Approve"}
              </button>
              <button
                onClick={handleDeny}
                disabled={busy}
                style={{
                  padding: "4px 12px",
                  background: "var(--term-panel)",
                  color: "var(--term-dim)",
                  border: "1px solid var(--term-border)",
                  borderRadius: "4px",
                  cursor: busy ? "not-allowed" : "pointer",
                  fontFamily: "monospace",
                  fontSize: "11px",
                  opacity: busy ? 0.6 : 1,
                }}
              >
                Deny
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function CreateBody({ payload }: { payload: any }) {
  const fields = [
    ["Title", payload.title],
    ["Type", payload.type],
    ["Priority", payload.priority],
    ["Section", payload.section],
    ["Projects", (payload.projects ?? []).join(", ")],
    ["Contexts", (payload.contexts ?? []).join(", ")],
    ["Tags", (payload.tags ?? []).join(", ")],
    ["Due", payload.due_at],
  ].filter(([, v]) => v);

  return (
    <div>
      {fields.map(([k, v]) => (
        <div key={k as string} style={{ display: "flex", gap: "8px", marginBottom: "2px" }}>
          <span style={{ color: "var(--term-dim)", minWidth: "64px" }}>{k}</span>
          <span style={{ color: "var(--term-fg)" }}>{v as string}</span>
        </div>
      ))}
    </div>
  );
}

function UpdateBody({ payload, diff }: { payload: any; diff: Record<string, any> }) {
  const entries = Object.entries(diff);
  return (
    <div>
      <div style={{ color: "var(--term-fg)", marginBottom: "6px", fontWeight: 600 }}>
        {payload.title ?? ""}
      </div>
      <table style={{ width: "100%", borderCollapse: "collapse", fontSize: "11px" }}>
        <thead>
          <tr>
            <th style={{ textAlign: "left", color: "var(--term-dim)", paddingBottom: "4px", fontWeight: 400 }}>Field</th>
            <th style={{ textAlign: "left", color: "var(--term-dim)", paddingBottom: "4px", fontWeight: 400 }}>Before</th>
            <th style={{ textAlign: "left", color: "var(--term-dim)", paddingBottom: "4px", fontWeight: 400 }}>After</th>
          </tr>
        </thead>
        <tbody>
          {entries.map(([field, vals]) => {
            const [before, after] = Array.isArray(vals) ? vals : ["", vals];
            return (
              <tr key={field}>
                <td style={{ color: "var(--term-dim)", paddingRight: "12px", paddingBottom: "2px" }}>{field}</td>
                <td style={{ color: "var(--term-error, #f87171)", paddingRight: "12px", textDecoration: "line-through", opacity: 0.8 }}>
                  {String(before ?? "")}
                </td>
                <td style={{ color: "var(--term-success, #4ade80)" }}>{String(after ?? "")}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function DeleteBody({ payload }: { payload: any }) {
  return (
    <div>
      <div style={{ color: "var(--term-error, #f87171)", marginBottom: "4px" }}>
        ⚠ This item will be permanently deleted.
      </div>
      <div style={{ color: "var(--term-fg)", fontWeight: 600 }}>
        {payload.title ?? `Item #${payload.id}`}
      </div>
    </div>
  );
}
