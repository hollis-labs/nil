import * as React from "react";
import { chat, config } from "../../wailsjs/go/models";
import * as Backend from "../../wailsjs/go/main/App";
import ChatMessage from "./ChatMessage";

type MessageEntry = {
  message: chat.ChatMessage;
  proposal?: chat.ActionProposal;
};

type Props = {
  open: boolean;
  onClose: () => void;
};

export default function ChatPanel({ open, onClose }: Props) {
  const [messages, setMessages] = React.useState<MessageEntry[]>([]);
  const [input, setInput] = React.useState("");
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // Vault context
  const [vaults, setVaults] = React.useState<config.Vault[]>([]);
  const [activeVault, setActiveVault] = React.useState<config.Vault | null>(null);
  const [showVaultPicker, setShowVaultPicker] = React.useState(false);

  // Session — use ref so cleanup callback always has the latest value.
  const sessionIdRef = React.useRef<number | null>(null);

  const threadRef = React.useRef<HTMLDivElement>(null);
  const inputRef = React.useRef<HTMLTextAreaElement>(null);

  // Scroll to bottom whenever messages change.
  React.useEffect(() => {
    if (threadRef.current) {
      threadRef.current.scrollTop = threadRef.current.scrollHeight;
    }
  }, [messages, loading]);

  // Start/end session when panel opens/closes.
  React.useEffect(() => {
    if (!open) return;

    let cancelled = false;

    (async () => {
      try {
        setMessages([]);
        setError(null);

        // Load vault info.
        const [allVaults, vault] = await Promise.all([
          Backend.GetVaults(),
          Backend.GetActiveVault(),
        ]);
        if (!cancelled) {
          setVaults(allVaults ?? []);
          setActiveVault(vault ?? null);
        }

        // Start chat session.
        const sess = await Backend.StartChatSession();
        if (!cancelled) {
          sessionIdRef.current = sess.id;
          // Focus input.
          setTimeout(() => inputRef.current?.focus(), 50);
        }
      } catch (err: any) {
        if (!cancelled) setError("Failed to start chat session.");
        console.error("ChatPanel: start session error:", err);
      }
    })();

    return () => {
      cancelled = true;
      const id = sessionIdRef.current;
      if (id !== null) {
        Backend.EndChatSession(id).catch(() => {});
        sessionIdRef.current = null;
      }
    };
  }, [open]);

  // Switch vault: end current session, load new vault, start new session.
  async function handleSwitchVault(vault: config.Vault) {
    setShowVaultPicker(false);
    if (vault.id === activeVault?.id) return;

    // End current session.
    const oldId = sessionIdRef.current;
    if (oldId !== null) {
      Backend.EndChatSession(oldId).catch(() => {});
      sessionIdRef.current = null;
    }

    try {
      await Backend.SwitchVault(vault.id);
      setActiveVault(vault);
      setMessages([]);
      setError(null);
      const sess = await Backend.StartChatSession();
      sessionIdRef.current = sess.id;
      inputRef.current?.focus();
    } catch (err: any) {
      setError("Failed to switch vault.");
      console.error("ChatPanel: switch vault error:", err);
    }
  }

  async function handleSend() {
    const text = input.trim();
    if (!text || loading || sessionIdRef.current === null) return;

    setInput("");
    setError(null);

    // Optimistic user message (id=0 — not yet persisted).
    const optimisticUser: MessageEntry = {
      message: {
        id: 0,
        session_id: sessionIdRef.current,
        role: "user",
        content: text,
        created_at: "",
      } as chat.ChatMessage,
    };
    setMessages(prev => [...prev, optimisticUser]);
    setLoading(true);

    try {
      const resp = await Backend.SendChatMessage(sessionIdRef.current, text);

      // Replace the optimistic message with the persisted one, then add the assistant reply.
      setMessages(prev => {
        const withoutOptimistic = prev.filter(m => m.message.id !== 0);
        const userEntry: MessageEntry = { message: { ...optimisticUser.message, id: resp.message?.id ?? 0 } as chat.ChatMessage };
        const assistantEntry: MessageEntry = {
          message: resp.message,
          proposal: resp.proposal ?? undefined,
        };
        return [...withoutOptimistic, userEntry, assistantEntry];
      });

      if (resp.error) {
        setError(resp.error);
      }
    } catch (err: any) {
      setMessages(prev => prev.filter(m => m.message.id !== 0));
      setError(String(err?.message ?? "Request failed."));
      console.error("SendChatMessage error:", err);
    } finally {
      setLoading(false);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    // Enter sends; Shift+Enter inserts newline.
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
    // Escape closes the panel.
    if (e.key === "Escape") {
      onClose();
    }
  }

  function handleProposalResolved(msgIndex: number) {
    // Reload the proposal for this message from the backend.
    setMessages(prev => {
      const entry = prev[msgIndex];
      if (!entry?.proposal) return prev;
      // Reload asynchronously — refresh status from backend.
      Backend.GetActionAudit(1).catch(() => {});
      return prev;
    });
  }

  if (!open) return null;

  const dryRunMode = false; // Will be read from config in Settings iteration.

  return (
    <>
      {/* Backdrop */}
      <div
        onClick={onClose}
        style={{
          position: "fixed",
          inset: 0,
          background: "rgba(0,0,0,0.45)",
          zIndex: 900,
        }}
      />

      {/* Panel */}
      <div style={{
        position: "fixed",
        top: 0,
        right: 0,
        bottom: 0,
        width: "400px",
        background: "var(--term-bg)",
        borderLeft: "1px solid var(--term-border)",
        display: "flex",
        flexDirection: "column",
        zIndex: 901,
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
      }}>

        {/* Header */}
        <div style={{
          padding: "12px 14px",
          borderBottom: "1px solid var(--term-border)",
          display: "flex",
          alignItems: "center",
          gap: "8px",
          flexShrink: 0,
        }}>
          <span style={{
            fontSize: "11px",
            fontWeight: 700,
            letterSpacing: "0.1em",
            color: "var(--term-info, var(--term-accent))",
          }}>
            VAULT CHAT
          </span>

          {/* Vault selector */}
          <div style={{ position: "relative", marginLeft: "auto" }}>
            <button
              onClick={() => setShowVaultPicker(v => !v)}
              title="Switch chat vault"
              style={{
                padding: "2px 8px",
                fontSize: "10px",
                background: "var(--term-panel)",
                border: "1px solid var(--term-border)",
                borderRadius: "3px",
                color: "var(--term-dim)",
                cursor: vaults.length > 1 ? "pointer" : "default",
                fontFamily: "monospace",
                display: "flex",
                alignItems: "center",
                gap: "4px",
              }}
            >
              {activeVault?.name ?? "—"}
              {vaults.length > 1 && <span style={{ opacity: 0.6 }}>▾</span>}
            </button>

            {showVaultPicker && vaults.length > 1 && (
              <div style={{
                position: "absolute",
                top: "calc(100% + 4px)",
                right: 0,
                background: "var(--term-bg)",
                border: "1px solid var(--term-border)",
                borderRadius: "6px",
                overflow: "hidden",
                zIndex: 910,
                minWidth: "140px",
                boxShadow: "0 4px 16px rgba(0,0,0,0.4)",
              }}>
                {vaults.map(v => (
                  <button
                    key={v.id}
                    onClick={() => handleSwitchVault(v)}
                    style={{
                      display: "block",
                      width: "100%",
                      padding: "7px 12px",
                      textAlign: "left",
                      background: v.id === activeVault?.id ? "var(--term-panel)" : "transparent",
                      border: "none",
                      color: "var(--term-fg)",
                      fontSize: "12px",
                      fontFamily: "monospace",
                      cursor: "pointer",
                    }}
                    onMouseEnter={e => { e.currentTarget.style.background = "var(--term-panel)"; }}
                    onMouseLeave={e => { e.currentTarget.style.background = v.id === activeVault?.id ? "var(--term-panel)" : "transparent"; }}
                  >
                    {v.id === activeVault?.id ? "✓ " : "  "}{v.name}
                  </button>
                ))}
              </div>
            )}
          </div>

          {dryRunMode && (
            <span style={{
              fontSize: "10px",
              color: "var(--term-warn, var(--term-fg))",
              border: "1px solid var(--term-warn, var(--term-border))",
              borderRadius: "3px",
              padding: "1px 5px",
              letterSpacing: "0.05em",
            }}>
              DRY RUN
            </span>
          )}

          {/* Close button */}
          <button
            onClick={onClose}
            title="Close (Esc)"
            style={{
              marginLeft: "4px",
              padding: "2px 6px",
              background: "transparent",
              border: "1px solid var(--term-border)",
              borderRadius: "3px",
              color: "var(--term-dim)",
              cursor: "pointer",
              fontSize: "14px",
              lineHeight: "1",
            }}
          >
            ×
          </button>
        </div>

        {/* Message thread */}
        <div
          ref={threadRef}
          style={{
            flex: 1,
            overflowY: "auto",
            padding: "14px",
            display: "flex",
            flexDirection: "column",
          }}
        >
          {messages.length === 0 && !loading && (
            <div style={{
              margin: "auto",
              textAlign: "center",
              color: "var(--term-dim)",
              fontSize: "12px",
              lineHeight: "1.8",
            }}>
              <div style={{ fontSize: "22px", marginBottom: "8px" }}>💬</div>
              <div>Ask about your vault or request an action.</div>
              <div style={{ opacity: 0.6, marginTop: "4px" }}>
                Try: "find tasks tagged urgent" or "create a todo to review Q1 report"
              </div>
            </div>
          )}

          {messages.map((entry, i) => (
            <ChatMessage
              key={`${entry.message.session_id}-${entry.message.id}-${i}`}
              message={entry.message}
              proposal={entry.proposal}
              vaultName={activeVault?.name}
              onProposalResolved={() => handleProposalResolved(i)}
            />
          ))}

          {loading && (
            <div style={{
              display: "flex",
              alignItems: "flex-start",
              marginBottom: "12px",
            }}>
              <div style={{
                padding: "8px 12px",
                borderRadius: "12px 12px 12px 2px",
                background: "var(--term-panel)",
                border: "1px solid var(--term-border)",
                color: "var(--term-dim)",
                fontSize: "13px",
              }}>
                <LoadingDots />
              </div>
            </div>
          )}

          {error && (
            <div style={{
              padding: "8px 12px",
              borderRadius: "6px",
              background: "rgba(248,113,113,0.12)",
              border: "1px solid var(--term-error, #f87171)",
              color: "var(--term-error, #f87171)",
              fontSize: "12px",
              marginBottom: "8px",
            }}>
              {error}
            </div>
          )}
        </div>

        {/* Input area */}
        <div style={{
          borderTop: "1px solid var(--term-border)",
          padding: "10px 12px",
          display: "flex",
          gap: "8px",
          alignItems: "flex-end",
          flexShrink: 0,
        }}>
          <textarea
            ref={inputRef}
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask or request an action… (Enter to send)"
            rows={2}
            style={{
              flex: 1,
              resize: "none",
              background: "var(--term-panel)",
              border: "1px solid var(--term-border)",
              borderRadius: "6px",
              color: "var(--term-fg)",
              fontSize: "12px",
              fontFamily: "inherit",
              padding: "6px 10px",
              outline: "none",
              lineHeight: "1.5",
            }}
            onFocus={e => { e.currentTarget.style.borderColor = "var(--term-accent)"; }}
            onBlur={e => { e.currentTarget.style.borderColor = "var(--term-border)"; }}
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || loading || sessionIdRef.current === null}
            title="Send (Enter)"
            style={{
              padding: "6px 14px",
              background: "var(--term-accent)",
              color: "#000",
              border: "none",
              borderRadius: "6px",
              cursor: (!input.trim() || loading) ? "not-allowed" : "pointer",
              fontFamily: "monospace",
              fontWeight: 700,
              fontSize: "12px",
              opacity: (!input.trim() || loading) ? 0.45 : 1,
              height: "32px",
              flexShrink: 0,
              transition: "opacity 0.15s",
            }}
          >
            Send
          </button>
        </div>
      </div>
    </>
  );
}

/** Animated three-dot loading indicator. */
function LoadingDots() {
  const [frame, setFrame] = React.useState(0);
  React.useEffect(() => {
    const t = setInterval(() => setFrame(f => (f + 1) % 4), 400);
    return () => clearInterval(t);
  }, []);
  return <span>{"•".repeat(frame + 1)}&nbsp;&nbsp;&nbsp;</span>;
}
