import * as React from "react";
import { ItemRow } from "./TerminalList";
import CustomScrollbar from "./CustomScrollbar";
import * as Backend from "../../wailsjs/go/main/App";

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onOpenItem: (row: ItemRow) => void;
};

function fmtShortDate(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export default function QuickSearchModal({ open, onOpenChange, onOpenItem }: Props) {
  const [query, setQuery] = React.useState("");
  const [typeFilter, setTypeFilter] = React.useState<"" | "todo" | "note">("");
  const [results, setResults] = React.useState<ItemRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [focusedIndex, setFocusedIndex] = React.useState(0);
  const debounceRef = React.useRef<NodeJS.Timeout | null>(null);
  const inputRef = React.useRef<HTMLInputElement>(null);
  const rowRefs = React.useRef<(HTMLDivElement | null)[]>([]);

  const doSearch = React.useCallback(async (q: string, type: string) => {
    if (!q.trim()) {
      setResults([]);
      return;
    }
    setLoading(true);
    try {
      const res = await Backend.Search({
        query: q.trim(),
        page: 0,
        page_size: 20,
        sort_by: "created_at",
        sort_dir: "desc",
        type: type === "" ? "all" : type,
        include_inbox: false,
      } as any);
      setResults(Array.isArray(res) ? (res as ItemRow[]) : []);
      setFocusedIndex(0);
    } catch (err) {
      console.error("QuickSearch error:", err);
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Debounced search trigger
  React.useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(query, typeFilter), 150);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query, typeFilter, doSearch]);

  // Reset state when modal opens
  React.useEffect(() => {
    if (open) {
      setQuery("");
      setResults([]);
      setFocusedIndex(0);
      setTypeFilter("");
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open]);

  // Scroll focused row into view
  React.useEffect(() => {
    rowRefs.current[focusedIndex]?.scrollIntoView({ block: "nearest" });
  }, [focusedIndex]);

  // Keyboard navigation
  React.useEffect(() => {
    if (!open) return;
    const handle = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onOpenChange(false);
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setFocusedIndex((i) => Math.min(i + 1, results.length - 1));
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setFocusedIndex((i) => Math.max(i - 1, 0));
        return;
      }
      if (e.key === "Enter" && results[focusedIndex]) {
        e.preventDefault();
        onOpenItem(results[focusedIndex]);
        return;
      }
    };
    window.addEventListener("keydown", handle);
    return () => window.removeEventListener("keydown", handle);
  }, [open, results, focusedIndex, onOpenChange, onOpenItem]);

  if (!open) return null;

  const filterChips: { label: string; value: "" | "todo" | "note" }[] = [
    { label: "All", value: "" },
    { label: "Todos", value: "todo" },
    { label: "Notes", value: "note" },
  ];

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.7)",
        display: "flex",
        alignItems: "flex-start",
        justifyContent: "center",
        paddingTop: "120px",
        zIndex: 50,
      }}
      onClick={() => onOpenChange(false)}
    >
      <div
        className="terminal-card"
        style={{
          width: "620px",
          maxWidth: "95vw",
          height: "480px",
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search input row */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "8px",
            padding: "12px 14px",
            borderBottom: "1px solid var(--term-border)",
          }}
        >
          <span style={{ color: "var(--term-dim)", fontSize: "16px", lineHeight: 1 }}>🔍</span>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search everything…"
            style={{
              flex: 1,
              background: "transparent",
              border: "none",
              outline: "none",
              color: "var(--term-fg)",
              fontSize: "14px",
              fontFamily: "inherit",
            }}
          />
          {query && (
            <button
              onClick={() => { setQuery(""); setResults([]); inputRef.current?.focus(); }}
              style={{
                background: "none",
                border: "none",
                cursor: "pointer",
                color: "var(--term-dim)",
                fontSize: "14px",
                padding: "0 2px",
                lineHeight: 1,
              }}
              title="Clear"
            >
              ✕
            </button>
          )}
        </div>

        {/* Type filter chips */}
        <div
          style={{
            display: "flex",
            gap: "6px",
            padding: "8px 14px",
            borderBottom: "1px solid var(--term-border)",
          }}
        >
          {filterChips.map((chip) => (
            <button
              key={chip.value}
              className={`badge ${typeFilter === chip.value ? "warn" : "fg"}`}
              onClick={() => setTypeFilter(chip.value)}
              style={{
                padding: "3px 10px",
                fontSize: "11px",
                borderRadius: "4px",
                border: "none",
                cursor: "pointer",
                opacity: typeFilter === chip.value ? 1 : 0.6,
              }}
            >
              {chip.label}
            </button>
          ))}
        </div>

        {/* Result list */}
        <div style={{ flex: 1, overflow: "hidden" }}>
          <CustomScrollbar style={{ height: "100%" }}>
            {query === "" ? (
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  height: "100%",
                  color: "var(--term-dim)",
                  fontSize: "13px",
                  opacity: 0.6,
                  padding: "24px",
                  textAlign: "center",
                }}
              >
                Type to search todos and notes…
              </div>
            ) : loading && results.length === 0 ? (
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  height: "100%",
                  color: "var(--term-dim)",
                  fontSize: "13px",
                  opacity: 0.6,
                }}
              >
                Searching…
              </div>
            ) : results.length === 0 ? (
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  height: "100%",
                  color: "var(--term-dim)",
                  fontSize: "13px",
                  opacity: 0.6,
                  padding: "24px",
                  textAlign: "center",
                }}
              >
                No results for "{query}"
              </div>
            ) : (
              <div>
                {results.map((row, i) => {
                  const isFocused = i === focusedIndex;
                  const taxonomy = [
                    ...(row.contexts || []).map((c) => `@${c}`),
                    ...(row.projects || []).map((p) => `+${p}`),
                    ...(row.tags || []).map((t) => `#${t}`),
                  ].slice(0, 3);

                  return (
                    <div
                      key={row.id}
                      ref={(el) => { rowRefs.current[i] = el; }}
                      onClick={() => onOpenItem(row)}
                      onMouseEnter={() => setFocusedIndex(i)}
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: "10px",
                        padding: "10px 14px",
                        cursor: "pointer",
                        background: isFocused ? "var(--term-panel)" : "transparent",
                        borderLeft: isFocused ? "2px solid var(--term-accent)" : "2px solid transparent",
                        transition: "background 0.1s ease",
                        minHeight: "48px",
                      }}
                    >
                      {/* Type badge */}
                      <span
                        className={`badge ${row.type === "note" ? "info" : "success"}`}
                        style={{
                          fontSize: "9px",
                          padding: "2px 5px",
                          borderRadius: "3px",
                          flexShrink: 0,
                          fontWeight: 700,
                          letterSpacing: "0.05em",
                          minWidth: "36px",
                          textAlign: "center",
                        }}
                      >
                        {row.type === "note" ? "NOTE" : "TODO"}
                      </span>

                      {/* Title */}
                      <span
                        style={{
                          flex: 1,
                          color: "var(--term-fg)",
                          fontSize: "13px",
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                        }}
                      >
                        {row.title || <em style={{ opacity: 0.4 }}>Untitled</em>}
                      </span>

                      {/* Taxonomy */}
                      {taxonomy.length > 0 && (
                        <div style={{ display: "flex", gap: "4px", flexShrink: 0 }}>
                          {taxonomy.map((t, ti) => (
                            <span
                              key={ti}
                              style={{
                                fontSize: "10px",
                                color: "var(--term-dim)",
                                background: "var(--term-bgAlt)",
                                borderRadius: "3px",
                                padding: "1px 5px",
                                maxWidth: "80px",
                                overflow: "hidden",
                                textOverflow: "ellipsis",
                                whiteSpace: "nowrap",
                              }}
                            >
                              {t}
                            </span>
                          ))}
                        </div>
                      )}

                      {/* Date */}
                      <span
                        style={{
                          fontSize: "10px",
                          color: "var(--term-dim)",
                          flexShrink: 0,
                          opacity: 0.7,
                          minWidth: "40px",
                          textAlign: "right",
                        }}
                      >
                        {fmtShortDate(row.created_at)}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}
          </CustomScrollbar>
        </div>

        {/* Footer hint */}
        <div
          style={{
            padding: "8px 14px",
            borderTop: "1px solid var(--term-border)",
            fontSize: "11px",
            color: "var(--term-dim)",
            opacity: 0.6,
            textAlign: "center",
          }}
        >
          ↑↓ navigate · Enter open · Esc close
        </div>
      </div>
    </div>
  );
}
