import * as React from "react";
import { TodoRow } from "./TerminalList";
import CustomScrollbar from "./CustomScrollbar";
import * as Backend from "../../wailsjs/go/main/App";
import { Archive, Trash2, CheckCircle, CheckSquare, Square } from "lucide-react";

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
  const [selectedIds, setSelectedIds] = React.useState<Set<number>>(new Set());
  const [bulkDeleteConfirm, setBulkDeleteConfirm] = React.useState(false);
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

  React.useEffect(() => { loadItems(""); }, [loadItems]);

  // Debounced search
  React.useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => { loadItems(searchQuery); }, 300);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [searchQuery, loadItems]);

  // Clear stale selections when items reload
  React.useEffect(() => {
    const validIds = new Set(items.map(i => i.id));
    setSelectedIds(prev => {
      const next = new Set<number>();
      prev.forEach(id => { if (validIds.has(id)) next.add(id); });
      return next.size === prev.size ? prev : next;
    });
    setFocusedIndex(0);
  }, [items]);

  // Keyboard navigation
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      const onSearchInput = target.classList.contains("inbox-search-input");

      switch (e.key) {
        case "Escape":
          if (selectedIds.size > 0) {
            e.preventDefault();
            setSelectedIds(new Set());
            setBulkDeleteConfirm(false);
          } else {
            e.preventDefault();
            onClose();
          }
          break;
        case "ArrowDown":
          if (!onSearchInput) { e.preventDefault(); setFocusedIndex(i => Math.min(i + 1, items.length - 1)); }
          break;
        case "ArrowUp":
          if (!onSearchInput) { e.preventDefault(); setFocusedIndex(i => Math.max(i - 1, 0)); }
          break;
        case "Enter":
          if (!onSearchInput) { e.preventDefault(); if (items[focusedIndex]) onEdit(items[focusedIndex]); }
          break;
        case "x":
          if (!onSearchInput) {
            e.preventDefault();
            const item = items[focusedIndex];
            if (item) toggleSelect(item.id);
          }
          break;
        case "p":
          if (!onSearchInput) { e.preventDefault(); if (items[focusedIndex]) handleProcess(items[focusedIndex].id); }
          break;
        case "a":
          if (!onSearchInput) { e.preventDefault(); if (items[focusedIndex]) handleArchive(items[focusedIndex].id); }
          break;
        case "d":
          if (!onSearchInput) { e.preventDefault(); if (items[focusedIndex]) handleDelete(items[focusedIndex].id); }
          break;
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [items, focusedIndex, selectedIds, onClose, onEdit]);

  // Scroll focused item into view
  React.useEffect(() => {
    const row = listRef.current?.children[focusedIndex] as HTMLElement | undefined;
    row?.scrollIntoView({ block: "nearest" });
  }, [focusedIndex]);

  // ── Selection helpers ──────────────────────────────────────────────────────

  function toggleSelect(id: number) {
    setSelectedIds(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
    setBulkDeleteConfirm(false);
  }

  function toggleSelectAll() {
    if (selectedIds.size === items.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(items.map(i => i.id)));
    }
    setBulkDeleteConfirm(false);
  }

  const allSelected = items.length > 0 && selectedIds.size === items.length;
  const someSelected = selectedIds.size > 0 && !allSelected;

  // ── Single-item actions ────────────────────────────────────────────────────

  async function handleProcess(id: number) {
    try {
      await Backend.ProcessInboxItem(id);
      await loadItems(searchQuery);
      onProcessed();
    } catch (err) { console.error("Failed to process:", err); }
  }

  async function handleArchive(id: number) {
    try {
      await Backend.Archive(id, true);
      await loadItems(searchQuery);
      onProcessed();
    } catch (err) { console.error("Failed to archive:", err); }
  }

  async function handleDelete(id: number) {
    if (showDeleteConfirm !== id) { setShowDeleteConfirm(id); return; }
    try {
      await Backend.DeleteTodo(id);
      setShowDeleteConfirm(null);
      await loadItems(searchQuery);
      onProcessed();
    } catch (err) { console.error("Failed to delete:", err); }
  }

  // ── Bulk actions ───────────────────────────────────────────────────────────

  async function handleBulkProcess() {
    const ids = Array.from(selectedIds);
    await Promise.all(ids.map(id => Backend.ProcessInboxItem(id)));
    setSelectedIds(new Set());
    await loadItems(searchQuery);
    onProcessed();
  }

  async function handleBulkArchive() {
    const ids = Array.from(selectedIds);
    await Promise.all(ids.map(id => Backend.Archive(id, true)));
    setSelectedIds(new Set());
    await loadItems(searchQuery);
    onProcessed();
  }

  async function handleBulkDelete() {
    if (!bulkDeleteConfirm) { setBulkDeleteConfirm(true); return; }
    const ids = Array.from(selectedIds);
    await Promise.all(ids.map(id => Backend.DeleteTodo(id)));
    setSelectedIds(new Set());
    setBulkDeleteConfirm(false);
    await loadItems(searchQuery);
    onProcessed();
  }

  // ── Helpers ────────────────────────────────────────────────────────────────

  function formatDate(iso?: string) {
    if (!iso) return "";
    return new Date(iso).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
  }

  const selCount = selectedIds.size;

  // ── Checkbox component ─────────────────────────────────────────────────────

  function Checkbox({ checked, indeterminate, onChange, title }: { checked: boolean; indeterminate?: boolean; onChange: () => void; title?: string }) {
    const ref = React.useRef<HTMLInputElement>(null);
    React.useEffect(() => {
      if (ref.current) ref.current.indeterminate = indeterminate ?? false;
    }, [indeterminate]);
    return (
      <input
        ref={ref}
        type="checkbox"
        checked={checked}
        onChange={onChange}
        title={title}
        style={{
          width: "14px",
          height: "14px",
          flexShrink: 0,
          cursor: "pointer",
          accentColor: "var(--term-accent)",
        }}
      />
    );
  }

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>

      {/* Search bar + select-all + count */}
      <div style={{ display: "flex", gap: "8px", alignItems: "center", marginBottom: "8px" }}>
        <Checkbox
          checked={allSelected}
          indeterminate={someSelected}
          onChange={toggleSelectAll}
          title={allSelected ? "Deselect all" : "Select all"}
        />
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
        Click to edit · x select · p process · a archive · d delete · ↑↓ navigate
      </div>

      {/* List */}
      <CustomScrollbar style={{ flex: 1 }}>
        {loading && items.length === 0 ? (
          <div style={{ textAlign: "center", color: "var(--term-dim)", padding: "40px 0", fontSize: "13px" }}>Loading…</div>
        ) : items.length === 0 ? (
          <div style={{ textAlign: "center", color: "var(--term-dim)", padding: "40px 0", fontSize: "13px" }}>
            {searchQuery ? "No matching inbox items." : "Inbox is clear ✓"}
          </div>
        ) : (
          <div ref={listRef}>
            {items.map((item, idx) => {
              const summary = notesSummary(item.notes_md);
              const isFocused = focusedIndex === idx;
              const isSelected = selectedIds.has(item.id);
              const isConfirmingDelete = showDeleteConfirm === item.id;

              return (
                <div
                  key={item.id}
                  onClick={() => { setFocusedIndex(idx); onEdit(item); }}
                  style={{
                    padding: "10px 12px",
                    marginBottom: "4px",
                    borderRadius: "6px",
                    background: isSelected
                      ? "color-mix(in srgb, var(--term-accent) 12%, var(--term-panel))"
                      : isFocused ? "var(--term-panel)" : "transparent",
                    border: isSelected
                      ? "1px solid color-mix(in srgb, var(--term-accent) 40%, transparent)"
                      : isFocused ? "1px solid var(--term-border)" : "1px solid transparent",
                    cursor: "pointer",
                    transition: "background 0.1s, border-color 0.1s",
                  }}
                >
                  {/* Row header: checkbox + type badge + title + date */}
                  <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: summary ? "3px" : "6px" }}>
                    {/* Checkbox — stops propagation so clicking it doesn't open edit */}
                    <div onClick={(e) => { e.stopPropagation(); toggleSelect(item.id); }} style={{ display: "flex", alignItems: "center" }}>
                      {isSelected
                        ? <CheckSquare size={15} style={{ color: "var(--term-accent)", flexShrink: 0 }} />
                        : <Square size={15} style={{ color: "var(--term-dim)", flexShrink: 0, opacity: 0.5 }} />
                      }
                    </div>

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
                      paddingLeft: "23px", // align under title (past checkbox + gap)
                    }}>
                      {summary}
                    </div>
                  )}

                  {/* Taxonomy chips */}
                  {((item.contexts?.length || 0) + (item.projects?.length || 0) + (item.tags?.length || 0)) > 0 && (
                    <div style={{ display: "flex", gap: "4px", flexWrap: "wrap", marginBottom: "6px", paddingLeft: "23px" }}>
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

                  {/* Per-row actions — stop propagation so they don't trigger edit */}
                  <div
                    style={{ display: "flex", gap: "6px", alignItems: "center", paddingLeft: "23px" }}
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
                        <button className="badge warn" onClick={() => handleDelete(item.id)}
                          style={{ fontSize: "11px", padding: "3px 8px", borderRadius: "4px", display: "flex", alignItems: "center", gap: "3px" }}>
                          <Trash2 size={11} /> Yes, Delete
                        </button>
                        <button className="badge" onClick={() => setShowDeleteConfirm(null)}
                          style={{ fontSize: "11px", padding: "3px 8px", borderRadius: "4px" }}>
                          Cancel
                        </button>
                      </>
                    ) : (
                      <button className="badge warn" onClick={() => handleDelete(item.id)}
                        style={{ fontSize: "11px", padding: "3px 8px", borderRadius: "4px", display: "flex", alignItems: "center", gap: "3px" }}
                        title="Delete">
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

      {/* Bulk action bar — slides in when items are selected */}
      {selCount > 0 && (
        <div style={{
          marginTop: "8px",
          padding: "10px 14px",
          background: "var(--term-panel)",
          border: "1px solid var(--term-border)",
          borderRadius: "8px",
          display: "flex",
          alignItems: "center",
          gap: "8px",
          flexWrap: "wrap",
        }}>
          <span style={{ fontSize: "12px", color: "var(--term-fg)", fontWeight: 600, marginRight: "4px" }}>
            {selCount} selected
          </span>

          <button
            className="badge success"
            onClick={handleBulkProcess}
            style={{ fontSize: "12px", padding: "4px 10px", borderRadius: "5px", display: "flex", alignItems: "center", gap: "4px" }}
            title="Move selected items out of inbox"
          >
            <CheckCircle size={12} /> Process All
          </button>

          <button
            className="badge"
            onClick={handleBulkArchive}
            style={{ fontSize: "12px", padding: "4px 10px", borderRadius: "5px", display: "flex", alignItems: "center", gap: "4px" }}
            title="Archive selected items"
          >
            <Archive size={12} /> Archive All
          </button>

          {bulkDeleteConfirm ? (
            <>
              <button
                className="badge warn"
                onClick={handleBulkDelete}
                style={{ fontSize: "12px", padding: "4px 10px", borderRadius: "5px", display: "flex", alignItems: "center", gap: "4px" }}
              >
                <Trash2 size={12} /> Confirm Delete {selCount}
              </button>
              <button
                className="badge"
                onClick={() => setBulkDeleteConfirm(false)}
                style={{ fontSize: "12px", padding: "4px 10px", borderRadius: "5px" }}
              >
                Cancel
              </button>
            </>
          ) : (
            <button
              className="badge warn"
              onClick={handleBulkDelete}
              style={{ fontSize: "12px", padding: "4px 10px", borderRadius: "5px", display: "flex", alignItems: "center", gap: "4px" }}
              title="Delete selected items"
            >
              <Trash2 size={12} /> Delete All
            </button>
          )}

          <button
            className="badge"
            onClick={() => { setSelectedIds(new Set()); setBulkDeleteConfirm(false); }}
            style={{ fontSize: "12px", padding: "4px 8px", borderRadius: "5px", marginLeft: "auto", opacity: 0.7 }}
            title="Clear selection"
          >
            ✕ Clear
          </button>
        </div>
      )}
    </div>
  );
}
