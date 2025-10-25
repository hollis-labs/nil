import * as React from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  todo: any;
  onSave: (md: string) => void;
};

export default function NotesModal({ open, onOpenChange, todo, onSave }: Props) {
  const editor = useEditor({
    extensions: [StarterKit],
    content: todo?.notes_md || "",
    editorProps: {
      attributes: {
        class: "prose prose-sm max-w-none min-h-[200px] p-3 border rounded focus:outline-none bg-transparent",
      },
    },
  });

  React.useEffect(() => {
    if (editor && open && todo) {
      editor.commands.setContent(todo.notes_md || "");
    }
  }, [editor, open, todo]);

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

  function handleSave() {
    const markdown = editor?.getHTML() || "";
    onSave(markdown);
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="terminal-card w-[720px] max-w-[95vw] p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="font-semibold">Notes: {todo?.title}</div>
          <button className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>
            Close
          </button>
        </div>
        <EditorContent editor={editor} />
        <div className="flex justify-end gap-2 mt-4">
          <button className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>
            Cancel
          </button>
          <button className="badge success" onClick={handleSave} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
