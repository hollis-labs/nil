import * as React from "react";
import { TodoRow } from "./TerminalList";
import CustomScrollbar from "./CustomScrollbar";
import * as Backend from "../../wailsjs/go/main/App";
import { Archive, Trash2, CheckCircle } from "lucide-react";

type Props = {
  onClose: () => void;
  onEdit: (row: TodoRow) => void;
  onProcessed: () => void;
};

/** Strip HTML tags and collapse whitespace for a plain-text preview. */
function stripHtml(html: string): string {
  return html
    .replace(/<[^>]+>/g, " ")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/\s+/g, " ")
    .trim();
}

function notesSummary(notesMd: string | undefined, maxLen = 120): string {
  if (!notesMd) return "";
  const text = stripHtml(notesMd);
  if (!text) return "";
  return text.length > maxLen ? text.slice(0, maxLen) + "…" : text;
}

export default function InboxView({ onClose, onEdit, onProcessed }: Props) {
  const [items, setItems] = React.useState<TodoRow[]>([]);
  const [searchQuery, setSearchQuery] = React.useState("");
  const [loading, setLoading] = React.useState(true);
  const [focusedIndex, setFocusedIndex] = React.useState(0);
  const [showDeleteConfirm, setShowDeleteConfirm] = React.useState<number | null>(null);
  const debounceRef = React.useRef<NodeJS.Timeout | null>(null);
  const listRef = React.useRef<HTMLDivElement>(null);

  const loadItems = React.useCallback(async (query: string) => {
    setLoading(true);
    try {
      const results = await Backend.GetInboxItems({
        query,
        page: 0,
        page_size: 200,
        sort_by: "created_at",
        sort_dir: "desc",
      } as any);
      setItems(Array.isArray(results) ? (results as TodoRow[]) : []);
    } catch (err) {
      console.error("Failed to load inbox items:", err);
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  React.useEffect(() => {
    loadItems("");
  }, [loadItems]);

  // Debounced search
  React.useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      loadItems(searchQuery);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [searchQuery, loadItems]);

  // Keyboard navigation
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      const onSearchInput = target.classList.contains("inbox-search-input");

      switch (e.key) {
        case "Escape":
          e.preventDefault();
          onClose();
          break;
        case "ArrowDown":
          if (!onSearchInput) {
            e.preventDefault();
            setFocusedIndex(i => Math.min(i + 1, items.length - 1));
          }
          break;
        case "ArrowUp":
          if (!onSearchInput) {
            e.preventDefault();
            setFocusedIndex(i => Math.max(i - 1, 0));
          }
          break;
        case "Enter":
          if (!onSearchInput) {
            e.preventDefault();
            if (items[focusedIndex]) onEdit(items[focusedIndex]);
          }
          break;
        case "p":
        case " ":
          if (!onSearchInput) {
            e.preventDefault();
            if (items[focusedIndex]) handleProcess(items[focusedIndex].id);
          }
          break;
        case "d":
          if (!onSearchInput) {
            e.preventDefault();
            if (items[focusedIndex]) handleDelete(items[focusedIndex].id);
          }
          break;
        case "a":
          if (!onSearchInput) {
            e.preventDefault();
            if (items[focusedIndex]) handleArchive(items[focusedIndex].id);
          }
          break;
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [items, focusedIndex, onClose, onEdit]);

  // Keep focused item scrolled into view
  React.useEffect(() => {
    const row = listRef.current?.children[focusedIndex] as HTMLElement | undefined;
    row?.scrollIntoView({ block: "nearest" });
  }, [focusedIndex]);

  // Reset focus when item count changes
  React.useEffect(() => {
    setFocusedIndex(0);
  }, [items.length]);

  async function handleProcess(id: number) {
    try {
      await Backend.ProcessInboxItem(id);
      await loadItems(searchQuery);
      onProcessed();
    } catch (err) {
      console.error("Failed to process inbox item:", err);
    }
  }

  async function handleArchive(id: number) {
    try {
      await Backend.Archive(id, true);
      await loadItems(searchQuery);
      onProcessed();
    } catch (err) {
      console.error("Failed to archive inbox item:", err);
    }
  }

  async function handleDelete(id: number) {
    if (showDeleteConfirm !== id) {
      setShowDeleteConfirm(id);
      return;
    }
    try {
      await Backend.DeleteTodo(id);
      setShowDeleteConfirm(null);
      await loadItems(searchQuery);
      onProcessed();
    } catch (err) {
      console.error("Failed to delete inbox item:", err);
    }
  }

  function formatDate(iso?: string) {
    if (!iso) return "";
    const d = new Date(iso);
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      {/* Search + item count */}
      <div style={{ display: "flex", gap: "8px", alignItems: "center", marginBottom: "8px" }}>
        <input
          className="inbox-search-input"
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search inbox…"
          style={{
            flex: 1,
            padding: "6px 10px",
            background: "var(--term-bg)",
            border: "1px solid var(--term-border)",
            borderRadius: "6px",
            color: "var(--term-fg)",
            fontSize: "13px",
          }}
        />
        <span style={{ fontSize: "11px", color: "var(--term-dim)", whiteSpace: "nowrap" }}>
          {loading ? "Loading…" : `${items.length} item${items.length !== 1 ? "s" : ""}`}
        </span>
      </div>

      {/* Keyboard hints */}
      <div style={{ fontSize: "11px", color: "var(--term-dim)", marginBottom: "8px", opacity: 0.6 }}>
        Click or Enter to edit · p process · a archive · d delete · ↑↓ navigate
      </div>

      {/* List */}
      <CustomScrollbar style={{ flex: 1 }}>
        {loading && items.length === 0 ? (
          <div style={{ textAlign: "center", color: "var(--term-dim)", padding: "40px 0", fontSize: "13px" }}>
            Loading…
          </div>
        ) : items.length === 0 ? (
          <div style={{ textAlign: "center", color: "var(--term-dim)", padding: "40px 0", fontSize: "13px" }}>
            {searchQuery ? "No matching inbox items." : "Inbox is clear ✓"}
          </div>
        ) : (
          <div ref={listRef}>
            {items.map((item, idx) => {
              const summary = notesSummary(item.notes_md);
              const isFocused = focusedIndex === idx;
              const isConfirmingDelete = showDeleteConfirm === item.id;

              return (
                <div
                  key={item.id}
                  onClick={() => {
                    setFocusedIndex(idx);
                    onEdit(item);
                  }}
                  style={{
                    padding: "10px 12px",
                    marginBottom: "4px",
                    borderRadius: "6px",
                    background: isFocused ? "var(--term-panel)" : "transparent",
                    border: isFocused ? "1px solid var(--term-border)" : "1px solid transparent",
                    cursor: "pointer",
                    transition: "background 0.1s",
                  }}
                >
                  {/* Row header: type badge + title + date */}
                  <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: summary ? "3px" : "6px" }}>
                    <span
                      className={`badge ${item.type === "note" ? "info" : "warn"}`}
                      style={{ fontSize: "10px", padding: "2px 5px", borderRadius: "3px", flexShrink: 0 }}
                    >
                      {item.type === "note" ? "NOTE" : "TODO"}
                    </span>
                    <span style={{
                      flex: 1,
                      fontSize: "13px",
                      fontWeight: item.title ? 500 : 400,
                      color: item.title ? "var(--term-fg)" : "var(--term-dim)",
                      fontStyle: item.title ? "normal" : "italic",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}>
                      {item.title || "(untitled)"}
                    </span>
                    <span style={{ fontSize: "11px", color: "var(--term-dim)", flexShrink: 0 }}>
                      {formatDate(item.created_at)}
                    </span>
                  </div>

                  {/* Notes summary */}
                  {summary && (
                    <div style={{
                      fontSize: "11px",
                      color: "var(--term-dim)",
                      marginBottom: "6px",
                      lineHeight: "1.4",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                      opacity: 0.8,
                    }}>
                      {summary}
                    </div>
                  )}

                  {/* Taxonomy chips */}
                  {((item.contexts?.length || 0) + (item.projects?.length || 0) + (item.tags?.length || 0)) > 0 && (
                    <div style={{ display: "flex", gap: "4px", flexWrap: "wrap", marginBottom: "6px" }}>
                      {item.contexts?.map(c => (
                        <span key={c} style={{ fontSize: "11px", color: "var(--term-dim)", background: "var(--term-bgAlt)", padding: "1px 5px", borderRadius: "3px" }}>@{c}</span>
                      ))}
                      {item.projects?.map(p => (
                        <span key={p} style={{ fontSize: "11px", color: "var(--term-dim)", background: "var(--term-bgAlt)", padding: "1px 5px", borderRadius: "3px" }}>+{p}</span>
                      ))}
                      {item.tags?.map(t => (
                        <span key={t} style={{ fontSize: "11px", color: "var(--term-dim)", background: "var(--term-bgAlt)", padding: "1px 5px", borderRadius: "3px" }}>#{t}</span>
                      ))}
                    </div>
                  )}

                  {/* Row actions — stop propagation so row click (edit) doesn't fire */}
                  <div
                    style={{ display: "flex", gap: "6px", alignItems: "center" }}
                    onClick={(e) => e.stopPropagation()}
                  >
                    <button
                      className="badge"
                      onClick={() => handleArchive(item.id)}
                      style={{ fontSize: "11px", padding: "3px 8px", borderRadius: "4px", display: "flex", alignItems: "center", gap: "3px" }}
                      title="Archive"
                    >
                      <Archive size={11} /> Archive
                    </button>
                    {isConfirmingDelete ? (
                      <>
                        <span style={{ fontSize: "11px", color: "var(--term-dim)" }}>Confirm?</span>
                        <button
                          className="badge warn"
                          onClick={() => handleDelete(item.id)}
                          style={{ fontSize: "11px", padding: "3px 8px", borderRadius: "4px", display: "flex", alignItems: "center", gap: "3px" }}
                        >
                          <Trash2 size={11} /> Yes, Delete
                        </button>
                        <button
                          className="badge"
                          onClick={() => setShowDeleteConfirm(null)}
                          style={{ fontSize: "11px", padding: "3px 8px", borderRadius: "4px" }}
                        >
                          Cancel
                        </button>
                      </>
                    ) : (
                      <button
                        className="badge warn"
                        onClick={() => handleDelete(item.id)}
                        style={{ fontSize: "11px", padding: "3px 8px", borderRadius: "4px", display: "flex", alignItems: "center", gap: "3px" }}
                        title="Delete"
                      >
                        <Trash2 size={11} /> Delete
                      </button>
                    )}
                    <button
                      className="badge success"
                      onClick={() => handleProcess(item.id)}
                      style={{ fontSize: "11px", padding: "3px 8px", borderRadius: "4px", display: "flex", alignItems: "center", gap: "3px", marginLeft: "auto" }}
                      title="Process — move out of inbox"
                    >
                      <CheckCircle size={11} /> Process
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </CustomScrollbar>
    </div>
  );
}
