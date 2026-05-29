import * as React from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { WikilinkExtension } from "@/lib/WikilinkExtension";
import * as Backend from "../../wailsjs/go/main/App";

type RefItem = {
  id: number;
  title: string;
  kind: string;
};

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  todo: any;
  // onSave receives (notesDoc JSON, notesHTML cache) — caller persists both.
  onSave: (notesDoc: string, notesHTML: string) => void;
  onRefClick?: (id: number, refType: string) => void;
};

export default function NotesModal({ open, onOpenChange, todo, onSave, onRefClick }: Props) {
  const [backlinks, setBacklinks] = React.useState<RefItem[]>([]);
  const [backlinksExpanded, setBacklinksExpanded] = React.useState(false);

  // Stable ref so the native listener always calls the latest handler
  const onRefClickRef = React.useRef(onRefClick);
  onRefClickRef.current = onRefClick;

  // Wrapper div that hosts the native capture-phase listener
  const editorContainerRef = React.useRef<HTMLDivElement>(null);

  const editor = useEditor({
    extensions: [StarterKit, WikilinkExtension],
    content: "",
    editorProps: {
      attributes: {
        class: "prose",
        style: "min-height: 200px; padding: 12px; border: 1px solid var(--term-border); border-radius: 4px; background: var(--term-bg); color: var(--term-fg); font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace; font-size: 13px; outline: none; overflow-x: hidden;",
      },
    },
  });

  // Attach a native mousedown listener in capture phase.
  // Capture fires top-down BEFORE ProseMirror's bubble-phase handlers,
  // so stopPropagation() here prevents ProseMirror from ever seeing the event.
  React.useEffect(() => {
    if (!open) return;
    const container = editorContainerRef.current;
    if (!container) return;

    const handleMouseDown = (e: MouseEvent) => {
      const target = (e.target as Element).closest('[data-type="wikilink"]');
      if (!target) return;
      e.preventDefault();
      e.stopPropagation();
      const id = parseInt(target.getAttribute("data-id") || "0", 10);
      const refType = target.getAttribute("data-ref-type") || "todo";
      if (id) onRefClickRef.current?.(id, refType);
    };

    container.addEventListener("mousedown", handleMouseDown, { capture: true });
    return () => container.removeEventListener("mousedown", handleMouseDown, { capture: true });
  }, [open]);

  React.useEffect(() => {
    if (editor && open && todo) {
      // Load the stored PM doc (JSON). Empty falls back to a blank doc.
      if (todo.notes_doc) {
        try {
          editor.commands.setContent(JSON.parse(todo.notes_doc));
        } catch {
          editor.commands.setContent("");
        }
      } else {
        editor.commands.setContent("");
      }
      setBacklinksExpanded(false);
      if (todo.id) {
        Backend.GetBackrefs(todo.id).then((results: any) => {
          setBacklinks(
            (results || []).map((t: any) => ({ id: t.id, title: t.title, kind: t.kind || "todo" }))
          );
        }).catch(() => setBacklinks([]));
      }
    }
  }, [editor, open, todo]);

  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open) {
        onOpenChange(false);
      }
    };
    window.addEventListener("keydown", handleEscape);
    return () => window.removeEventListener("keydown", handleEscape);
  }, [open, onOpenChange]);

  if (!open) return null;

  function handleSave() {
    // Send both — caller stamps both notes_doc (source of truth) and notes_html (cache).
    const doc = editor ? JSON.stringify(editor.getJSON()) : "";
    const html = editor?.getHTML() || "";
    onSave(doc, html);
  }

  return (
    <div style={{ position: "fixed", inset: 0, background: "rgba(0,0,0,0.7)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 50 }}>
      <div className="terminal-card" style={{ width: "720px", maxWidth: "95vw", padding: "16px" }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "12px" }}>
          <div style={{ fontWeight: 600, fontSize: "14px" }}>Notes: {todo?.title}</div>
          <button
            className="badge"
            onClick={() => onOpenChange(false)}
            style={{ padding: "8px 12px", fontSize: "13px", borderRadius: "6px" }}
          >
            Close
          </button>
        </div>

        {/* Hint */}
        <div style={{ fontSize: "11px", opacity: 0.5, marginBottom: "6px", fontFamily: "monospace" }}>
          Type @ to link a todo or note
        </div>

        <div ref={editorContainerRef}>
          <EditorContent editor={editor} />
        </div>

        {/* Backlinks */}
        {backlinks.length > 0 && (
          <div style={{ marginTop: "16px", borderTop: "1px solid var(--term-border)", paddingTop: "10px" }}>
            <button
              onClick={() => setBacklinksExpanded(!backlinksExpanded)}
              style={{
                display: "flex",
                alignItems: "center",
                gap: "6px",
                background: "transparent",
                border: "none",
                cursor: "pointer",
                color: "var(--term-dim)",
                fontSize: "11px",
                fontFamily: "monospace",
                padding: 0,
              }}
            >
              <span>{backlinksExpanded ? "▾" : "▸"}</span>
              <span>Referenced by {backlinks.length} item{backlinks.length !== 1 ? "s" : ""}</span>
            </button>
            {backlinksExpanded && (
              <div style={{ display: "flex", flexWrap: "wrap", gap: "6px", marginTop: "8px" }}>
                {backlinks.map((bl) => (
                  <button
                    key={bl.id}
                    onClick={() => onRefClick?.(bl.id, bl.kind)}
                    style={{
                      display: "inline-flex",
                      alignItems: "center",
                      gap: "5px",
                      padding: "3px 8px",
                      borderRadius: "4px",
                      fontSize: "12px",
                      cursor: "pointer",
                      background: bl.kind === "note"
                        ? "rgba(96,165,250,0.12)"
                        : "rgba(74,222,128,0.12)",
                      color: bl.kind === "note" ? "var(--term-info)" : "var(--term-success)",
                      border: `1px solid ${bl.kind === "note" ? "var(--term-info)" : "var(--term-success)"}`,
                      fontFamily: "monospace",
                    }}
                  >
                    <span style={{ fontSize: "10px", opacity: 0.7 }}>{bl.kind}</span>
                    {bl.title}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        <div style={{ display: "flex", justifyContent: "flex-end", gap: "8px", marginTop: "16px" }}>
          <button
            className="badge"
            onClick={() => onOpenChange(false)}
            style={{ padding: "8px 12px", fontSize: "13px", borderRadius: "6px" }}
          >
            Cancel
          </button>
          <button
            className="badge success"
            onClick={handleSave}
            style={{ padding: "8px 12px", fontSize: "13px", borderRadius: "6px" }}
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
