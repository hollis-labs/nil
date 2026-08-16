import * as React from "react";
import { EditorContent } from "@tiptap/react";
import type { Editor } from "@tiptap/react";
import { Save, Minimize2, Bold, Italic, Code, List, ListOrdered, Heading1, Heading2, Heading3 } from "lucide-react";
import CustomScrollbar from "./CustomScrollbar";

type Props = {
  open: boolean;
  editor: Editor | null;
  editorContainerRef: React.RefObject<HTMLDivElement>;
  onClose: () => void;
  // Mirrors the original inline handler: collapse back to the normal modal
  // and then run the same save-and-close path as the main form's submit.
  onSaveAndClose: (e: React.MouseEvent) => void;
};

const ToolbarButton = ({ onClick, isActive, icon: Icon, title }: { onClick: () => void; isActive?: boolean; icon: any; title: string }) => (
  <button
    type="button"
    onClick={onClick}
    title={title}
    style={{
      padding: '6px 8px',
      background: isActive ? 'var(--term-accent)' : 'transparent',
      border: 'none',
      borderRadius: '4px',
      cursor: 'pointer',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      color: isActive ? '#000' : 'var(--term-fg)',
      transition: 'all 0.15s ease',
      opacity: isActive ? 1 : 0.7,
    }}
    onMouseEnter={(e) => {
      if (!isActive) {
        e.currentTarget.style.background = 'var(--term-panel)';
        e.currentTarget.style.opacity = '1';
      }
    }}
    onMouseLeave={(e) => {
      if (!isActive) {
        e.currentTarget.style.background = 'transparent';
        e.currentTarget.style.opacity = '0.7';
      }
    }}
  >
    <Icon size={16} />
  </button>
);

// The fullscreen "expanded description" editor: the same TipTap `editor`
// instance as NotesEditorField (owned by the parent via useNotesEditor), but
// rendered with a formatting toolbar in its own full-viewport overlay. Only
// depends on the editor instance + open/close/save callbacks — no
// line/priority/tags/contexts/projects coupling.
export default function ExpandedNotesEditorModal({ open, editor, editorContainerRef, onClose, onSaveAndClose }: Props) {
  if (!open) return null;

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      background: 'rgba(0, 0, 0, 0.15)',
      backdropFilter: 'blur(1px)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 100,
    }}>
      <div className="terminal-card" style={{
        width: '90%',
        height: '90%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}>
        {/* Header */}
        <div style={{
          padding: '16px 20px',
          borderBottom: '1px solid var(--term-border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}>
          <label style={{ fontSize: '14px', fontWeight: 600 }}>Description</label>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              type="button"
              onClick={onClose}
              className="badge"
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                borderRadius: '6px',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
                background: 'var(--term-bg)',
                border: '1px solid var(--term-border)',
              }}
            >
              <Minimize2 size={14} />
              Close
            </button>
            <button
              type="button"
              onClick={onSaveAndClose}
              className="badge success"
              style={{
                padding: '6px 12px',
                fontSize: '12px',
                borderRadius: '6px',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
              }}
            >
              <Save size={14} />
              Save & Close
            </button>
          </div>
        </div>

        {/* Toolbar */}
        {editor && (
          <div style={{
            padding: '12px 20px',
            borderBottom: '1px solid var(--term-border)',
            display: 'flex',
            gap: '4px',
            flexWrap: 'wrap',
            background: 'var(--term-panel)',
          }}>
            <ToolbarButton
              onClick={() => editor.chain().focus().toggleBold().run()}
              isActive={editor.isActive('bold')}
              icon={Bold}
              title="Bold"
            />
            <ToolbarButton
              onClick={() => editor.chain().focus().toggleItalic().run()}
              isActive={editor.isActive('italic')}
              icon={Italic}
              title="Italic"
            />
            <ToolbarButton
              onClick={() => editor.chain().focus().toggleCode().run()}
              isActive={editor.isActive('code')}
              icon={Code}
              title="Inline Code"
            />
            <div style={{ width: '1px', height: '28px', background: 'var(--term-border)', margin: '0 4px' }} />
            <ToolbarButton
              onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
              isActive={editor.isActive('heading', { level: 1 })}
              icon={Heading1}
              title="Heading 1"
            />
            <ToolbarButton
              onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
              isActive={editor.isActive('heading', { level: 2 })}
              icon={Heading2}
              title="Heading 2"
            />
            <ToolbarButton
              onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
              isActive={editor.isActive('heading', { level: 3 })}
              icon={Heading3}
              title="Heading 3"
            />
            <div style={{ width: '1px', height: '28px', background: 'var(--term-border)', margin: '0 4px' }} />
            <ToolbarButton
              onClick={() => editor.chain().focus().toggleBulletList().run()}
              isActive={editor.isActive('bulletList')}
              icon={List}
              title="Bullet List"
            />
            <ToolbarButton
              onClick={() => editor.chain().focus().toggleOrderedList().run()}
              isActive={editor.isActive('orderedList')}
              icon={ListOrdered}
              title="Numbered List"
            />
          </div>
        )}

        {/* Editor */}
        <CustomScrollbar style={{ flex: 1, background: 'var(--term-bg)' }}>
          <div style={{ padding: '20px' }} ref={editorContainerRef}>
            <EditorContent editor={editor} />
          </div>
        </CustomScrollbar>
      </div>
    </div>
  );
}
