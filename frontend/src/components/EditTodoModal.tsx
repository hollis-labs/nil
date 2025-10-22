import * as React from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { TodoRow } from "./TerminalList";
import TagsInput from "./TagsInput";

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSubmit: (line: string, extras: any) => void;
  onUpdate?: (todo: TodoRow) => void;
  onDelete?: (id: number) => void;
  editTodo?: TodoRow | null;
};

export default function EditTodoModal({ open, onOpenChange, onSubmit, onUpdate, onDelete, editTodo }: Props) {
  const isEditMode = !!editTodo;
  
  const [line, setLine] = React.useState("");
  const [priority, setPriority] = React.useState("");
  const [due, setDue] = React.useState("");
  const [tags, setTags] = React.useState<string[]>([]);
  const [showDeleteConfirm, setShowDeleteConfirm] = React.useState(false);

  const editor = useEditor({
    extensions: [StarterKit],
    content: "",
    editorProps: {
      attributes: {
        class: "prose prose-sm max-w-none focus:outline-none bg-transparent tiptap-editor",
        style: "min-height: 250px; max-height: 400px; overflow-y: auto; background: var(--term-bg); color: var(--term-fg); padding: 8px 12px 12px 16px;",
      },
    },
  });

  React.useEffect(() => {
    if (open) {
      setShowDeleteConfirm(false);
      if (isEditMode && editTodo) {
        setLine(editTodo.title);
        setPriority(editTodo.priority || "");
        setDue(editTodo.due_at ? editTodo.due_at.slice(0, 10) : "");
        setTags(editTodo.tags || []);
        if (editor) {
          editor.commands.setContent(editTodo.notes_md || "");
        }
      } else {
        setLine("");
        setPriority("");
        setDue("");
        setTags([]);
        if (editor) {
          editor.commands.setContent("");
        }
      }
    }
  }, [open, isEditMode, editTodo, editor]);

  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onOpenChange(false);
      }
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [open, onOpenChange]);

  if (!open) return null;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const notes_md = editor?.getHTML() || "";
    
    if (isEditMode && editTodo && onUpdate) {
      const updated: TodoRow = {
        ...editTodo,
        title: line,
        priority: priority || undefined,
        due_at: due || undefined,
        tags,
        notes_md,
      };
      onUpdate(updated);
    } else {
      const extras: any = { tags };
      if (priority) extras.priority = priority;
      if (due) extras.due = due;
      if (notes_md) extras.notes_md = notes_md;
      onSubmit(line, extras);
    }
  }



  const modalStyle = {
    position: 'fixed' as const,
    inset: 0,
    background: 'rgba(0,0,0,0.7)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 50
  };

  const inputStyle = {
    width: '100%',
    padding: '8px 12px',
    background: 'var(--term-bg)',
    border: '1px solid var(--term-border)',
    borderRadius: '6px',
    color: 'var(--term-fg)',
    fontSize: '13px'
  };

  return (
    <div style={modalStyle}>
      <div className="terminal-card" style={{ width: '720px', maxWidth: '95vw', padding: '20px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
          <div style={{ fontWeight: 600 }}>{isEditMode ? 'Edit Todo' : 'Quick Add Todo'}</div>
          <button className="badge" onClick={() => onOpenChange(false)}>Close</button>
        </div>
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Task</label>
            <input
              autoFocus
              style={inputStyle}
              placeholder="e.g. Review pull request +project @context"
              value={line}
              onChange={(e) => setLine(e.target.value)}
            />
          </div>
          <div style={{ display: 'flex', gap: '12px' }}>
            <div style={{ flex: 1 }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Priority</label>
              <select 
                style={{
                  ...inputStyle,
                  appearance: 'none',
                  backgroundImage: `url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e")`,
                  backgroundRepeat: 'no-repeat',
                  backgroundPosition: 'right 8px center',
                  backgroundSize: '16px',
                  paddingRight: '32px'
                }}
                value={priority} 
                onChange={(e) => setPriority(e.target.value)}
              >
                <option value="" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>None</option>
                <option value="A" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>A (High)</option>
                <option value="B" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>B (Medium)</option>
                <option value="C" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>C (Low)</option>
              </select>
            </div>
            <div style={{ flex: 1 }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Due Date</label>
              <input type="date" style={inputStyle} value={due} onChange={(e) => setDue(e.target.value)} placeholder="Optional" />
            </div>
          </div>

          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Description</label>
            <div style={{ 
              border: '1px solid var(--term-border)', 
              borderRadius: '6px', 
              background: 'var(--term-bg)',
              overflow: 'hidden',
            }}>
              <EditorContent editor={editor} />
            </div>
          </div>

          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Tags</label>
            <TagsInput tags={tags} onTagsChange={setTags} placeholder="Add tag..." />
          </div>

          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px' }}>
            {isEditMode && onDelete && (
              <div>
                {!showDeleteConfirm ? (
                  <button type="button" className="badge warn" onClick={() => setShowDeleteConfirm(true)}>Delete</button>
                ) : (
                  <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    <span style={{ fontSize: '12px', color: 'var(--term-dim)' }}>Confirm delete?</span>
                    <button type="button" className="badge warn" onClick={() => { onDelete(editTodo!.id); onOpenChange(false); }}>Yes, Delete</button>
                    <button type="button" className="badge" onClick={() => setShowDeleteConfirm(false)}>Cancel</button>
                  </div>
                )}
              </div>
            )}
            {!isEditMode && <div />}
            <div style={{ display: 'flex', gap: '8px' }}>
              <button type="button" className="badge" onClick={() => onOpenChange(false)}>Cancel</button>
              <button type="submit" className="badge success">{isEditMode ? 'Save' : 'Create'}</button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
