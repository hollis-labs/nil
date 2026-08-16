import * as React from "react";
import { useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { Markdown } from "tiptap-markdown";
import { WikilinkExtension } from "@/lib/WikilinkExtension";

// The TipTap editor instance + the wikilink-click plumbing used by
// EditItemModal's notes editor (both the inline collapsed view and the
// expanded fullscreen view share this same editor instance).
//
// This is deliberately state-free with respect to the surrounding modal's
// line/priority/tags/contexts/projects fields — it only depends on
// onRefClick, so it lifts cleanly out of EditItemModal without threading any
// of that other state through.
export function useNotesEditor(onRefClick?: (id: number, refType: string) => void) {
  // Stable ref so the native listener below always calls the latest handler
  // without needing to be re-registered on every onRefClick identity change.
  const onRefClickRef = React.useRef(onRefClick);
  onRefClickRef.current = onRefClick;

  const editorContainerRef = React.useRef<HTMLDivElement>(null);

  // Capture-phase mousedown on the editor wrapper — fires before ProseMirror's
  // bubble-phase handlers so stopPropagation() fully prevents cursor movement.
  React.useEffect(() => {
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
  }, []);

  const editor = useEditor({
    extensions: [
      StarterKit,
      Markdown.configure({
        html: true,
        transformPastedText: true,
        transformCopiedText: false,
      }),
      WikilinkExtension,
    ],
    content: "",
    parseOptions: {
      preserveWhitespace: 'full',
    },
    editorProps: {
      attributes: {
        class: "tiptap-editor prose prose-sm sm:prose lg:prose-lg xl:prose-2xl mx-auto focus:outline-none",
        style: "min-height: 120px; background: var(--term-bg); color: var(--term-fg); font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace; font-size: 13px; overflow-x: hidden;",
      },
      // No custom handlePaste — tiptap-markdown's transformPastedText handles
      // markdown paste uniformly (headings, lists, inline emphasis, links,
      // code), avoiding the first-line-only heuristic of the old custom code.
    },
  });

  return { editor, editorContainerRef };
}
