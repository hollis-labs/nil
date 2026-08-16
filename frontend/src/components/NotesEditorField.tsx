import * as React from "react";
import { EditorContent } from "@tiptap/react";
import type { Editor } from "@tiptap/react";
import { Maximize2 } from "lucide-react";
import CustomScrollbar from "./CustomScrollbar";

type Props = {
  editor: Editor | null;
  editorContainerRef: React.RefObject<HTMLDivElement>;
  isFullscreen: boolean;
  onExpand: () => void;
};

// The inline "Description" field inside EditItemModal's main form: a label +
// expand button, and the TipTap EditorContent itself. This is the collapsed
// counterpart to ExpandedNotesEditorModal — both render the same `editor`
// instance (owned by the parent via useNotesEditor), so switching between
// them doesn't lose editor state.
//
// Deliberately does not own descriptionExpanded — the parent hides this
// component (by not rendering it) rather than this component hiding itself,
// since the parent also needs that same flag to decide whether to render the
// expanded modal instead.
export default function NotesEditorField({ editor, editorContainerRef, isFullscreen, onExpand }: Props) {
  return (
    <div>
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        marginBottom: '4px',
      }}>
        <label style={{
          fontSize: '12px',
          opacity: 0.8,
        }}>
          Description
        </label>
        <button
          type="button"
          onClick={onExpand}
          style={{
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
            padding: '4px',
            display: 'flex',
            alignItems: 'center',
            color: 'var(--term-dim)',
            transition: 'color 0.15s ease',
          }}
          onMouseEnter={(e) => e.currentTarget.style.color = 'var(--term-accent)'}
          onMouseLeave={(e) => e.currentTarget.style.color = 'var(--term-dim)'}
        >
          <Maximize2 size={16} />
        </button>
      </div>
      <div style={{
        border: '1px solid var(--term-border)',
        borderRadius: '6px',
        background: 'var(--term-bg)',
        overflow: 'hidden',
        // Grow with the viewport in fullscreen; capped in default mode.
        minHeight: isFullscreen ? '60vh' : '300px',
        maxHeight: isFullscreen ? '70vh' : '300px',
      }}>
        <CustomScrollbar style={{
          height: isFullscreen ? '70vh' : '296px',
          minHeight: isFullscreen ? '60vh' : '296px',
          maxHeight: isFullscreen ? '70vh' : '296px',
          background: 'var(--term-bg)',
        }}>
          <div ref={editorContainerRef}>
            <EditorContent editor={editor} />
          </div>
        </CustomScrollbar>
      </div>
    </div>
  );
}
