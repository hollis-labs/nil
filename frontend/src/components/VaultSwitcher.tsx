import * as React from "react";
import { config } from "../../wailsjs/go/models";

type Props = {
  open: boolean;
  vaults: config.Vault[];
  activeVaultId: string;
  onSwitch: (vault: config.Vault) => void;
  onClose: () => void;
};

export default function VaultSwitcher({ open, vaults, activeVaultId, onSwitch, onClose }: Props) {
  const [filter, setFilter] = React.useState("");
  const [focusedIndex, setFocusedIndex] = React.useState(0);
  const inputRef = React.useRef<HTMLInputElement>(null);
  const listRef = React.useRef<HTMLDivElement>(null);

  const filtered = vaults.filter(v =>
    v.name.toLowerCase().includes(filter.toLowerCase())
  );

  // Refs so the document capture handler always sees current values
  const filteredRef = React.useRef(filtered);
  filteredRef.current = filtered;
  const focusedIndexRef = React.useRef(focusedIndex);
  focusedIndexRef.current = focusedIndex;
  const onSwitchRef = React.useRef(onSwitch);
  onSwitchRef.current = onSwitch;
  const onCloseRef = React.useRef(onClose);
  onCloseRef.current = onClose;

  // Reset filter and focus when opened
  React.useEffect(() => {
    if (open) {
      setFilter("");
      setFocusedIndex(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  // Keep focused index in bounds when filter changes
  React.useEffect(() => {
    setFocusedIndex(0);
  }, [filter]);

  // Scroll focused row into view
  React.useEffect(() => {
    const row = listRef.current?.children[focusedIndex] as HTMLElement | undefined;
    row?.scrollIntoView({ block: "nearest" });
  }, [focusedIndex]);

  // Single document capture handler — fires before WKWebView's scroll routing
  React.useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          e.stopPropagation();
          setFocusedIndex(i => Math.min(i + 1, filteredRef.current.length - 1));
          break;
        case "ArrowUp":
          e.preventDefault();
          e.stopPropagation();
          setFocusedIndex(i => Math.max(i - 1, 0));
          break;
        case "Enter": {
          e.preventDefault();
          e.stopPropagation();
          const vault = filteredRef.current[focusedIndexRef.current];
          if (vault) onSwitchRef.current(vault);
          break;
        }
        case "Escape":
          e.preventDefault();
          e.stopPropagation();
          onCloseRef.current();
          break;
      }
    };
    document.addEventListener("keydown", handler, true);
    return () => document.removeEventListener("keydown", handler, true);
  }, [open]);

  if (!open) return null;

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.6)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 100,
      }}
      onClick={onClose}
    >
      <div
        className="terminal-card"
        style={{ width: "480px", maxWidth: "95vw", overflow: "hidden" }}
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div style={{
          padding: "14px 16px 10px",
          borderBottom: "1px solid var(--term-border)",
          display: "flex",
          alignItems: "center",
          gap: "8px",
        }}>
          <span style={{ fontSize: "11px", color: "var(--term-dim)", fontWeight: 600, letterSpacing: "0.06em", textTransform: "uppercase" }}>
            Switch Vault
          </span>
          <span style={{ fontSize: "10px", color: "var(--term-dim)", marginLeft: "auto" }}>
            ↑↓ navigate · Enter switch · Esc close
          </span>
        </div>

        {/* Filter input */}
        <div style={{ padding: "10px 16px", borderBottom: "1px solid var(--term-border)" }}>
          <input
            ref={inputRef}
            value={filter}
            onChange={e => setFilter(e.target.value)}
            placeholder="Filter vaults…"
            style={{
              width: "100%",
              padding: "6px 10px",
              background: "var(--term-bg)",
              border: "1px solid var(--term-border)",
              borderRadius: "4px",
              color: "var(--term-fg)",
              fontSize: "13px",
              outline: "none",
            }}
          />
        </div>

        {/* Vault list */}
        <div ref={listRef} style={{ maxHeight: "320px", overflowY: "auto" }}>
          {filtered.length === 0 ? (
            <div style={{ padding: "20px 16px", fontSize: "12px", color: "var(--term-dim)", textAlign: "center" }}>
              No vaults match "{filter}"
            </div>
          ) : (
            filtered.map((vault, idx) => {
              const isActive = vault.id === activeVaultId;
              const isFocused = idx === focusedIndex;
              return (
                <div
                  key={vault.id}
                  onClick={() => onSwitch(vault)}
                  style={{
                    padding: "10px 16px",
                    cursor: "pointer",
                    background: isFocused ? "var(--term-panel)" : "transparent",
                    borderLeft: `3px solid ${isActive ? "var(--term-accent)" : "transparent"}`,
                    borderBottom: "1px solid var(--term-border)",
                    transition: "background 0.1s",
                  }}
                  onMouseEnter={() => setFocusedIndex(idx)}
                >
                  <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                    <span style={{
                      fontSize: "13px",
                      fontWeight: isActive ? 600 : 400,
                      color: isActive ? "var(--term-accent)" : "var(--term-fg)",
                    }}>
                      {vault.name}
                    </span>
                    {isActive && (
                      <span className="badge success" style={{ fontSize: "10px", padding: "2px 6px", borderRadius: "3px" }}>
                        active
                      </span>
                    )}
                  </div>
                  <div style={{
                    fontSize: "11px",
                    color: "var(--term-dim)",
                    marginTop: "2px",
                    fontFamily: "monospace",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}>
                    {vault.path}
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}
