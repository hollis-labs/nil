import * as React from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { Markdown } from 'tiptap-markdown';
import { TodoRow } from "./TerminalList";
import TagsAutocomplete from "./TagsAutocomplete";
import ProjectsAutocomplete from "./ProjectsAutocomplete";
import ContextsAutocomplete from "./ContextsAutocomplete";
import TodoTitleInput from "./TodoTitleInput";
import CustomScrollbar from "./CustomScrollbar";
import { useSettings } from "./SettingsModal";
import TemplatePickerCombobox from "./TemplatePickerCombobox";
import TemplateSaveDialog from "./TemplateSaveDialog";
import { TaskTemplate } from "@/lib/templates";
import { getActiveSession, getSessionProfiles, setActiveSession, SessionProfile } from "@/lib/sessionContext";
import { Save } from "lucide-react";
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
  const [showTemplateSaveDialog, setShowTemplateSaveDialog] = React.useState(false);
  const [mergeContext, setMergeContext] = React.useState(true);
  const [sessionProfiles, setSessionProfiles] = React.useState<SessionProfile[]>([]);

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

      // Load session profiles
      setSessionProfiles(getSessionProfiles());
    }
  }, [open]);

  const editor = useEditor({
    extensions: [
      StarterKit,
      Markdown.configure({
        html: true,
        transformPastedText: true,
        transformCopiedText: false,
      })
    ],
    content: "",
    parseOptions: {
      preserveWhitespace: 'full',
    },
    editorProps: {
      attributes: {
        class: "prose prose-sm max-w-none focus:outline-none bg-transparent tiptap-editor",
        style: "min-height: 120px; background: var(--term-bg); color: var(--term-fg); padding: 8px 12px 12px 16px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace; font-size: 13px; overflow-x: hidden;",
      },
      handlePaste: (view, event) => {
        const text = event.clipboardData?.getData('text/plain');
        if (text && editor) {
          // Check if it looks like markdown
          if (text.match(/^#{1,6}\s|^\*\*|^##|^\-\s|^\*\s|^\d+\.\s/m)) {
            event.preventDefault();
            editor.commands.insertContent(text);
            return true;
          }
        }
        return false;
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
        // Priority order: session context > search filters > default tags
        const activeSession = getActiveSession();

        setLine("");
        setPriority(activeSession?.priority || "");
        setDue("");

        // Merge session context with search filters
        const sessionTags = activeSession?.tags || [];
        const sessionContexts = activeSession?.contexts || [];
        const sessionProjects = activeSession?.projects || [];

        const mergedTags = [...new Set([...sessionTags, ...defaultTags])];
        const mergedContexts = [...new Set([...sessionContexts, ...defaultContexts])];
        const mergedProjects = [...new Set([...sessionProjects, ...defaultProjects])];

        setTags(mergedTags);
        setContexts(mergedContexts);
        setProjects(mergedProjects);
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
      // ESC key disabled - users must click Close/Cancel
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && open) {
        e.preventDefault();
        handleSubmit(e as any);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [open, line, priority, due, tags, contexts, projects]);

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

  const handleTemplateSelect = (template: TaskTemplate) => {
    // Merge template values with existing ones
    setContexts([...new Set([...contexts, ...template.contexts])]);
    setProjects([...new Set([...projects, ...template.projects])]);
    setTags([...new Set([...tags, ...template.tags])]);
    if (template.priority && !priority) {
      setPriority(template.priority);
    }
  };

  const handleContextProfileSelect = (profileId: string) => {
    const profile = sessionProfiles.find(p => p.id === profileId);
    if (!profile) return;

    if (mergeContext) {
      // Merge with existing values
      setContexts([...new Set([...contexts, ...profile.contexts])]);
      setProjects([...new Set([...projects, ...profile.projects])]);
      setTags([...new Set([...tags, ...profile.tags])]);
      if (profile.priority && !priority) {
        setPriority(profile.priority);
      }
    } else {
      // Replace existing values
      setContexts(profile.contexts);
      setProjects(profile.projects);
      setTags(profile.tags);
      if (profile.priority) {
        setPriority(profile.priority);
      }
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
    <>
      <style>{`
        .tiptap-editor ul,
        .tiptap-editor ol {
          padding-left: 1.2em !important;
          margin: 0.5em 0;
        }
        .tiptap-editor ul li,
        .tiptap-editor ol li {
          margin: 0.25em 0;
        }
        .tiptap-editor h1 {
          font-size: 1.5em;
          font-weight: 600;
          margin: 0.75em 0 0.5em 0;
          line-height: 1.3;
        }
        .tiptap-editor h2 {
          font-size: 1.3em;
          font-weight: 600;
          margin: 0.65em 0 0.4em 0;
          line-height: 1.3;
        }
        .tiptap-editor h3 {
          font-size: 1.15em;
          font-weight: 600;
          margin: 0.55em 0 0.35em 0;
          line-height: 1.3;
        }
        .tiptap-editor p {
          margin: 0.5em 0;
          line-height: 1.5;
        }
        .tiptap-editor strong {
          font-weight: 600;
          color: var(--term-fg);
        }
        .tiptap-editor em {
          font-style: italic;
        }
        .tiptap-editor code {
          background: var(--term-panel);
          padding: 0.15em 0.4em;
          border-radius: 3px;
          font-size: 0.9em;
          border: 1px solid var(--term-border);
        }
        .tiptap-editor pre {
          background: var(--term-panel);
          border: 1px solid var(--term-border);
          border-radius: 6px;
          padding: 0.75em;
          margin: 0.75em 0;
          overflow-x: auto;
        }
        .tiptap-editor pre code {
          background: transparent;
          padding: 0;
          border: none;
          font-size: 0.9em;
        }
        .tiptap-editor blockquote {
          border-left: 3px solid var(--term-border);
          padding-left: 1em;
          margin: 0.75em 0;
          color: var(--term-dim);
          font-style: italic;
        }
        .tiptap-editor a {
          color: var(--term-info);
          text-decoration: underline;
        }
        .tiptap-editor hr {
          border: none;
          border-top: 1px solid var(--term-border);
          margin: 1em 0;
        }
      `}</style>
      <div style={modalStyle}>
        <div className="terminal-card" style={{
        width: '720px',
        maxWidth: '95vw',
        height: '650px',
        maxHeight: '650px',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden'
      }}>
        {/* Fixed Header */}
        <div style={{ padding: '20px 20px 0 20px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
            <div style={{ fontWeight: 600 }}>{isEditMode ? 'Edit Todo' : 'Add Todo'}</div>
            <button
              className="badge info"
              onClick={() => onOpenChange(false)}
              style={{
                padding: '6px 12px',
                fontSize: '12px'
              }}
            >
              Close
            </button>
          </div>
        </div>

        {/* Scrollable Body */}
        <CustomScrollbar style={{
          flex: 1,
          minHeight: 0,

        }}>
          <div style={{
            padding: '8px 20px 40px 20px'
          }}>
            {/* Priority/Date chips row */}
            <div style={{
              display: 'flex',
              gap: '8px',
              marginBottom: '16px',
              paddingBottom: '8px',
              borderBottom: '1px solid var(--term-border)',
              flexWrap: 'wrap',
              paddingTop: '4px'
            }}>
              <button
                type="button"
                className={`badge ${priority === 'A' ? 'warn' : ''}`}
                onClick={() => setPriority(priority === 'A' ? '' : 'A')}
                style={{ flex: '0 0 auto' }}
              >
                High
              </button>
              <button
                type="button"
                className={`badge ${priority === 'B' ? 'info' : ''}`}
                onClick={() => setPriority(priority === 'B' ? '' : 'B')}
                style={{ flex: '0 0 auto' }}
              >
                Medium
              </button>
              <button
                type="button"
                className={`badge ${priority === 'C' ? 'success' : ''}`}
                onClick={() => setPriority(priority === 'C' ? '' : 'C')}
                style={{ flex: '0 0 auto' }}
              >
                Low
              </button>
              <button
                type="button"
                className={`badge ${priority === '' ? 'success' : ''}`}
                onClick={() => setPriority('')}
                style={{ flex: '0 0 auto' }}
              >
                None
              </button>
              <button
                type="button"
                className={`badge ${due ? 'info' : ''}`}
                onClick={(e) => {
                  e.stopPropagation();
                  console.log('Date chip clicked! due value:', due, 'truthy?', !!due);
                  if (due) {
                    console.log('Clearing due date');
                    setDue('');
                  } else {
                    console.log('Creating date picker input');
                    const input = document.createElement('input');
                    input.type = 'date';
                    input.value = due;
                    input.style.position = 'absolute';
                    input.style.top = e.currentTarget.getBoundingClientRect().top + 'px';
                    input.style.left = e.currentTarget.getBoundingClientRect().left + 'px';
                    input.style.width = e.currentTarget.getBoundingClientRect().width + 'px';
                    input.style.height = e.currentTarget.getBoundingClientRect().height + 'px';
                    input.style.opacity = '0.01';
                    input.style.zIndex = '9999';
                    input.style.cursor = 'pointer';
                    document.body.appendChild(input);

                    input.focus();

                    requestAnimationFrame(() => {
                      try {
                        console.log('Attempting showPicker, available?', !!input.showPicker);
                        if (input.showPicker) {
                          input.showPicker();
                        } else {
                          input.click();
                        }
                      } catch (err) {
                        console.error('Date picker error:', err);
                      }
                    });

                    input.onchange = (e) => {
                      console.log('Date selected:', (e.target as HTMLInputElement).value);
                      setDue((e.target as HTMLInputElement).value);
                      if (document.body.contains(input)) {
                        document.body.removeChild(input);
                      }
                    };
                    input.onblur = () => {
                      setTimeout(() => {
                        if (document.body.contains(input)) {
                          document.body.removeChild(input);
                        }
                      }, 100);
                    };
                  }
                }}
                style={{ flex: '0 0 auto' }}
              >
                {due ? `Due: ${due}` : 'Set Due Date'}
              </button>
            </div>
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '4px' }}>
              <label style={{ fontSize: '12px' }} className="text-dim">Task</label>
              <div style={{ display: 'flex', gap: '8px' }}>
                <TemplatePickerCombobox
                  onSelect={handleTemplateSelect}
                  currentValues={{ contexts, projects, tags, priority: priority || undefined }}
                />
                <button
                  type="button"
                  className="badge"
                  onClick={() => setShowTemplateSaveDialog(true)}
                  style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '12px' }}
                >
                  <Save size={12} />
                  Save Template
                </button>
              </div>
            </div>
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

          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '12px'
          }}>
            <div>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Projects</label>
              <ProjectsAutocomplete values={projects} onValuesChange={setProjects} placeholder="Add project..." />
            </div>

            <div>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Contexts</label>
              <ContextsAutocomplete values={contexts} onValuesChange={setContexts} placeholder="Add context..." />
            </div>

            <div>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Tags</label>
              <TagsAutocomplete values={tags} onValuesChange={setTags} placeholder="Add tag..." />
            </div>

            {!isEditMode && (() => {
              const activeSession = getActiveSession();
              const hasSessionContext = activeSession && (
                (activeSession.contexts?.length || 0) > 0 ||
                (activeSession.projects?.length || 0) > 0 ||
                (activeSession.tags?.length || 0) > 0 ||
                activeSession.priority
              );
              const hasSearchFilters = defaultContexts.length > 0 || defaultProjects.length > 0 || defaultTags.length > 0;

              // Merge session and search filters for display
              const displayContexts = [...new Set([...(activeSession?.contexts || []), ...defaultContexts])];
              const displayProjects = [...new Set([...(activeSession?.projects || []), ...defaultProjects])];
              const displayTags = [...new Set([...(activeSession?.tags || []), ...defaultTags])];

              return (
                <div>
                  <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Session Context</label>
                  <div style={{
                    padding: '8px 12px',
                    background: 'var(--term-bg)',
                    border: '1px solid var(--term-border)',
                    borderRadius: '6px',
                    minHeight: '38px',
                    display: 'flex',
                    alignItems: 'flex-start',
                    gap: '8px'
                  }}>
                    <input
                      type="checkbox"
                      id="use-defaults"
                      checked={useDefaults}
                      onChange={toggleUseDefaults}
                      style={{ cursor: 'pointer', marginTop: '2px', flexShrink: 0 }}
                    />
                    <div style={{ fontSize: '11px', flex: 1 }} className="text-dim">
                      {!hasSessionContext && !hasSearchFilters && <div>No active context</div>}
                      {useDefaults && (
                        <>
                          {displayContexts.length > 0 && <div>@{displayContexts.join(', @')}</div>}
                          {displayProjects.length > 0 && <div>+{displayProjects.join(', +')}</div>}
                          {displayTags.length > 0 && <div>#{displayTags.join(', #')}</div>}
                          {activeSession?.priority && <div>Pri: {activeSession.priority}</div>}
                        </>
                      )}
                      {!useDefaults && (hasSessionContext || hasSearchFilters) && <div>Context disabled</div>}
                    </div>
                  </div>
                </div>
              );
            })()}

            {isEditMode && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '4px', height: '16px' }}>
                  <label style={{ fontSize: '12px', lineHeight: '16px' }} className="text-dim">Apply Context</label>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '4px', height: '16px' }}>
                    <input
                      type="checkbox"
                      id="merge-context"
                      checked={mergeContext}
                      onChange={(e) => setMergeContext(e.target.checked)}
                      style={{
                        cursor: 'pointer',
                        width: '12px',
                        height: '12px',
                        margin: 0
                      }}
                    />
                    <label htmlFor="merge-context" style={{ fontSize: '11px', cursor: 'pointer', lineHeight: '16px', margin: 0 }} className="text-dim">
                      Merge
                    </label>
                  </div>
                </div>
                <select
                  style={{
                    width: '100%',
                    padding: '8px',
                    background: 'var(--term-bg)',
                    border: '1px solid var(--term-border)',
                    borderRadius: '6px',
                    color: 'var(--term-fg)',
                    fontSize: '13px',
                    cursor: 'pointer',
                    height: '44px',
                    appearance: 'none',
                    backgroundImage: `url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e")`,
                    backgroundRepeat: 'no-repeat',
                    backgroundPosition: 'right 8px center',
                    backgroundSize: '16px',
                    paddingRight: '32px'
                  }}
                  onChange={(e) => e.target.value && handleContextProfileSelect(e.target.value)}
                  defaultValue=""
                >
                  <option value="">Select context profile...</option>
                  {sessionProfiles.map(profile => (
                    <option key={profile.id} value={profile.id}>{profile.name}</option>
                  ))}
                </select>
              </div>
            )}
          </div>

          <div>
            <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">Description</label>
            <div style={{
              border: '1px solid var(--term-border)',
              borderRadius: '6px',
              background: 'var(--term-bg)',
              overflow: 'hidden',
              minHeight: '200px',
              maxHeight: '200px',
              paddingTop: '4px'
            }}>
              <CustomScrollbar style={{
                height: '50',
                minHeight: '200px',
                maxHeight: '200px'
              }}>
                <EditorContent editor={editor} />
              </CustomScrollbar>
            </div>
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
          <div style={{ display: 'flex', gap: '8px' }}>
            {isEditMode && onDelete && !showDeleteConfirm && (
              <button type="button" className="badge warn" onClick={() => setShowDeleteConfirm(true)}>Delete</button>
            )}
            {isEditMode && onDelete && showDeleteConfirm && (
              <>
                <span style={{ fontSize: '12px', color: 'var(--term-dim)', marginRight: '4px' }}>Confirm delete?</span>
                <button type="button" className="badge warn" onClick={() => { onDelete(editTodo!.id); onOpenChange(false); }}>Yes, Delete</button>
              </>
            )}
            <button type="button" className="badge" onClick={(e) => { e.preventDefault(); handleClear(); }}>Clear</button>
            <button type="button" className="badge" onClick={() => onOpenChange(false)}>Cancel</button>
          </div>
          <button type="button" className="badge success" onClick={(e) => { e.preventDefault(); handleSubmit(e as any); }}>{isEditMode ? 'Save' : 'Create'}</button>
        </div>
      </div>

      <TemplateSaveDialog
        open={showTemplateSaveDialog}
        onOpenChange={setShowTemplateSaveDialog}
        values={{
          contexts,
          projects,
          tags,
          priority: (priority as 'A' | 'B' | 'C') || undefined,
        }}
      />
    </div>
    </>
  );
}
