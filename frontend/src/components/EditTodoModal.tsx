import * as React from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { TodoRow } from "./TerminalList";
import TagsInput from "./TagsInput";
import TodoTitleInput from "./TodoTitleInput";
import CustomScrollbar from "./CustomScrollbar";
import { useSettings } from "./SettingsModal";
import * as Backend from "../../wailsjs/go/main/App";

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSubmit: (line: string, extras: any) => void;
  onUpdate?: (todo: TodoRow) => void;
  onDelete?: (id: number) => void;
  editTodo?: TodoRow | null;
  defaultContexts?: string[];
  defaultProjects?: string[];
  defaultTags?: string[];
};

export default function EditTodoModal({ open, onOpenChange, onSubmit, onUpdate, onDelete, editTodo, defaultContexts = [], defaultProjects = [], defaultTags = [] }: Props) {
  const isEditMode = !!editTodo;
  const { settings } = useSettings();

  const [line, setLine] = React.useState("");
  const [priority, setPriority] = React.useState("");
  const [due, setDue] = React.useState("");
  const [tags, setTags] = React.useState<string[]>([]);
  const [contexts, setContexts] = React.useState<string[]>([]);
  const [projects, setProjects] = React.useState<string[]>([]);
  const [useDefaults, setUseDefaults] = React.useState(true);
  const [showDeleteConfirm, setShowDeleteConfirm] = React.useState(false);
  const [availableProjects, setAvailableProjects] = React.useState<string[]>([]);
  const [availableContexts, setAvailableContexts] = React.useState<string[]>([]);
  const [availableTags, setAvailableTags] = React.useState<string[]>([]);

  React.useEffect(() => {
    if (open) {
      Backend.GetFilters().then((result: any) => {
        console.log("GetFilters result:", result);
        
        if (result && typeof result === 'object') {
          const projects = result.projects || [];
          const contexts = result.contexts || [];
          const tags = result.tags || [];
          
          setAvailableProjects(projects);
          setAvailableContexts(contexts);
          setAvailableTags(tags);
          
          console.log("Loaded filters:", { projects, contexts, tags });
        } else {
          console.warn("Unexpected GetFilters result format:", result);
          setAvailableProjects([]);
          setAvailableContexts([]);
          setAvailableTags([]);
        }
      }).catch(err => {
        console.error("Failed to load filters:", err);
        setAvailableProjects([]);
        setAvailableContexts([]);
        setAvailableTags([]);
      });
    }
  }, [open]);

  const editor = useEditor({
    extensions: [StarterKit],
    content: "",
    editorProps: {
      attributes: {
        class: "prose prose-sm max-w-none focus:outline-none bg-transparent tiptap-editor",
        style: "min-height: 120px; max-height: 200px; overflow-y: auto; background: var(--term-bg); color: var(--term-fg); padding: 8px 12px 12px 16px;",
      },
    },
  });

  const prevOpenRef = React.useRef(open);

  React.useEffect(() => {
    // Only run this effect when modal is first opened (transition from closed to open)
    if (open && !prevOpenRef.current) {
      setShowDeleteConfirm(false);
      if (isEditMode && editTodo) {
        setLine(editTodo.title);
        setPriority(editTodo.priority || "");
        setDue(editTodo.due_at ? editTodo.due_at.slice(0, 10) : "");
        setTags(editTodo.tags || []);
        setContexts(editTodo.contexts || []);
        setProjects(editTodo.projects || []);
        setUseDefaults(false);
        if (editor) {
          editor.commands.setContent(editTodo.notes_md || "");
        }
      } else {
        setLine("");
        setPriority("");
        setDue("");
        setTags(defaultTags);
        setContexts(defaultContexts);
        setProjects(defaultProjects);
        setUseDefaults(true);
        if (editor) {
          editor.commands.setContent("");
        }
      }
    }
    prevOpenRef.current = open;
  }, [open, isEditMode, editTodo, editor, defaultContexts, defaultProjects, defaultTags]);

  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onOpenChange(false);
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && open) {
        e.preventDefault();
        handleSubmit(e as any);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [open, onOpenChange, line, priority, due, tags, contexts, projects]);

  if (!open) return null;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const notes_md = editor?.getHTML() || "";

    // Strip prefixes from tags/contexts/projects (in case user typed them)
    let cleanTags = tags.map(t => t.replace(/^#/, ''));
    const cleanContexts = contexts.map(c => c.replace(/^@/, ''));
    const cleanProjects = projects.map(p => p.replace(/^\+/, ''));

    // Apply default tags if no tags or projects are specified (only for create mode)
    if (!isEditMode && cleanTags.length === 0 && cleanProjects.length === 0 && settings.defaultTags && settings.defaultTags.length > 0) {
      cleanTags = [...settings.defaultTags];
    }

    if (isEditMode && editTodo && onUpdate) {
      const updated: TodoRow = {
        ...editTodo,
        title: line,
        priority: priority || undefined,
        due_at: due || undefined,
        tags: cleanTags,
        contexts: cleanContexts,
        projects: cleanProjects,
        notes_md,
      };
      onUpdate(updated);
    } else {
      const extras: any = {
        tags: cleanTags,
        contexts: cleanContexts,
        projects: cleanProjects
      };
      if (priority) extras.priority = priority;
      if (due) extras.due = due;
      if (notes_md) extras.notes_md = notes_md;
      onSubmit(line, extras);
    }
  }
  
  const toggleUseDefaults = () => {
    setUseDefaults(!useDefaults);
    if (!useDefaults) {
      setContexts(defaultContexts);
      setProjects(defaultProjects);
      setTags(defaultTags);
    } else {
      setContexts([]);
      setProjects([]);
      setTags([]);
    }
  };

  const handleClear = () => {
    console.log("Clear button clicked");
    setLine("");
    setPriority("");
    setDue("");
    setTags([]);
    setContexts([]);
    setProjects([]);
    setUseDefaults(false);
    if (editor) {
      editor.commands.clearContent();
    }
  };



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
      <div className="terminal-card" style={{ 
        width: '720px', 
        maxWidth: '95vw', 
        height: '650px',
        display: 'flex',
        flexDirection: 'column'
      }}>
        {/* Fixed Header */}
        <div style={{ padding: '20px 20px 0 20px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
            <div style={{ fontWeight: 600 }}>{isEditMode ? 'Edit Todo' : 'Quick Add Todo'}</div>
            <button className="badge" onClick={() => onOpenChange(false)}>Close</button>
          </div>
        </div>

        {/* Scrollable Body */}
        <CustomScrollbar style={{ 
          flex: 1, 
          minHeight: 0
        }}>
          <div style={{ padding: '0 20px' }}>
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Task</label>
            <TodoTitleInput
              value={line}
              onChange={setLine}
              placeholder="e.g. Review pull request +project @context"
              autoFocus={true}
              recentContexts={availableContexts}
              recentProjects={availableProjects}
              recentTags={availableTags}
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

          {!isEditMode && (defaultContexts.length > 0 || defaultProjects.length > 0 || defaultTags.length > 0) && (
            <div style={{ 
              padding: '12px', 
              background: 'var(--term-panel)', 
              borderRadius: '6px', 
              border: '1px solid var(--term-border)' 
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                <input
                  type="checkbox"
                  id="use-defaults"
                  checked={useDefaults}
                  onChange={toggleUseDefaults}
                  style={{ cursor: 'pointer' }}
                />
                <label htmlFor="use-defaults" style={{ fontSize: '12px', cursor: 'pointer' }}>
                  Use active filter context
                </label>
              </div>
              {useDefaults && (
                <div style={{ fontSize: '11px', marginLeft: '24px' }} className="text-dim">
                  {defaultContexts.length > 0 && <div>Contexts: {defaultContexts.map(c => `@${c}`).join(', ')}</div>}
                  {defaultProjects.length > 0 && <div>Projects: {defaultProjects.map(p => `+${p}`).join(', ')}</div>}
                  {defaultTags.length > 0 && <div>Tags: {defaultTags.map(t => `#${t}`).join(', ')}</div>}
                </div>
              )}
            </div>
          )}

          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Contexts</label>
            <TagsInput tags={contexts} onTagsChange={setContexts} placeholder="Add context (e.g., work, home)..." prefix="@" />
          </div>

          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Projects</label>
            <TagsInput tags={projects} onTagsChange={setProjects} placeholder="Add project (e.g., myproject)..." prefix="+" />
          </div>

          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Tags</label>
            <TagsInput tags={tags} onTagsChange={setTags} placeholder="Add tag..." prefix="#" />
          </div>
        </form>
          </div>
        </CustomScrollbar>

        {/* Fixed Footer */}
        <div style={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          padding: '16px 20px 20px 20px',
          borderTop: '1px solid var(--term-border)' 
        }}>
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
          {!isEditMode && (
            <div style={{ display: 'flex', gap: '8px' }}>
              <button type="button" className="badge" onClick={(e) => { e.preventDefault(); handleClear(); }}>Clear</button>
              <button type="button" className="badge" onClick={() => onOpenChange(false)}>Cancel</button>
            </div>
          )}
          {isEditMode && (
            <button type="button" className="badge" onClick={(e) => { e.preventDefault(); handleClear(); }}>Clear</button>
          )}
          <button type="button" className="badge success" onClick={(e) => { e.preventDefault(); handleSubmit(e as any); }}>{isEditMode ? 'Save' : 'Create'}</button>
        </div>
      </div>
    </div>
  );
}
