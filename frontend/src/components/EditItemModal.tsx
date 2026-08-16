import * as React from "react";
import { ItemRow } from "./TerminalList";
import TagsAutocomplete from "./TagsAutocomplete";
import ProjectsAutocomplete from "./ProjectsAutocomplete";
import ContextsAutocomplete from "./ContextsAutocomplete";
import ItemTitleInput from "./ItemTitleInput";
import { useSettings } from "./SettingsModal";
import TemplatePickerCombobox from "./TemplatePickerCombobox";
import TemplateSaveDialog from "./TemplateSaveDialog";
import DeleteConfirmControl from "./DeleteConfirmControl";
import ClosePromptDialog from "./ClosePromptDialog";
import NotesEditorField from "./NotesEditorField";
import ExpandedNotesEditorModal from "./ExpandedNotesEditorModal";
import { TaskTemplate } from "@/lib/templates";
import { getActiveSession, getSessionProfiles, SessionProfile } from "@/lib/sessionContext";
import { useNotesEditor } from "@/lib/useNotesEditor";
import { useDirtyClose, EditItemFieldsSnapshot } from "@/lib/useDirtyClose";
import { Save, ToggleLeft, ToggleRight, Maximize2, Minimize2, FileText, CheckSquare } from "lucide-react";
import * as Backend from "../../wailsjs/go/main/App";

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSubmit: (line: string, extras: any) => void;
  onUpdate?: (todo: ItemRow) => void;
  // onSaveStay: save without closing (Cmd+S). Edit mode only; parent should
  // persist the item but leave the modal open. If omitted, Cmd+S falls back
  // to the normal save-and-close path.
  onSaveStay?: (todo: ItemRow) => Promise<void> | void;
  onDelete?: (id: number) => void;
  editItem?: ItemRow | null;
  defaultContexts?: string[];
  defaultProjects?: string[];
  defaultTags?: string[];
  isNoteMode?: boolean;
  onConvertType?: (todo: ItemRow) => void;
  onRefClick?: (id: number, refType: string) => void;
};

export default function EditItemModal({ open, onOpenChange, onSubmit, onUpdate, onSaveStay, onDelete, editItem, defaultContexts = [], defaultProjects = [], defaultTags = [], isNoteMode = false, onConvertType, onRefClick }: Props) {
  const isEditMode = !!editItem;
  const { settings, setSettings } = useSettings();

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
  const [descriptionExpanded, setDescriptionExpanded] = React.useState(false);
  // Fullscreen toggle: when true the modal expands to fill the viewport.
  // Persists per session in component state only — intentionally not in
  // localStorage so each editor opening starts in the default size.
  const [isFullscreen, setIsFullscreen] = React.useState(false);
  // Renderer version stamped on fresh saves. Mirrors the Go-side
  // `notesHTMLVersion` constant; bump both together when the renderer (TipTap
  // extensions, ingest.DocToHTML mapping) changes in a way that would
  // produce different output for the same input.
  const NOTES_HTML_VERSION = 1;

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

  // TipTap editor instance + wikilink-click plumbing — extracted since it
  // doesn't depend on any of this modal's other state (see useNotesEditor).
  const { editor, editorContainerRef } = useNotesEditor(onRefClick);

  // Dirty-detection + close-behavior (Escape / Close / Cancel all route
  // through dirtyClose.requestClose()) — extracted into useDirtyClose().
  // onSaveAndClose is wired to the same doSave() used by the "Save & Close"
  // footer button, matching the original inline requestClose()'s 'always'
  // branch exactly. doSave is a hoisted function declaration further down in
  // this component, so referencing it here (before its textual declaration)
  // is safe.
  const dirtyClose = useDirtyClose({
    getCurrentSnapshot: (): EditItemFieldsSnapshot => ({
      line, priority, due, tags, contexts, projects,
      notes_doc: editor ? JSON.stringify(editor.getJSON()) : "",
    }),
    closeBehavior: settings.closeBehavior ?? 'ask',
    onOpenChange,
    onSaveAndClose: () => doSave(),
    onRememberBehavior: (behavior) => setSettings({ ...settings, closeBehavior: behavior }),
  });

  const prevOpenRef = React.useRef(open);

  React.useEffect(() => {
    // Only run this effect when modal is first opened (transition from closed to open)
    if (open && !prevOpenRef.current) {
      setShowDeleteConfirm(false);
      setDescriptionExpanded(false);
      dirtyClose.cancelClosePrompt();
      if (isEditMode && editItem) {
        setLine(editItem.title);
        setPriority(editItem.priority || "");
        setDue(editItem.due_at ? editItem.due_at.slice(0, 10) : "");
        setTags(editItem.tags || []);
        setContexts(editItem.contexts || []);
        setProjects(editItem.projects || []);
        setUseDefaults(false);
        if (editor) {
          // Load the stored PM JSON directly. Empty doc falls back to
          // setContent('') which TipTap renders as an empty paragraph.
          if (editItem.notes_doc) {
            try {
              editor.commands.setContent(JSON.parse(editItem.notes_doc));
            } catch {
              editor.commands.setContent("");
            }
          } else {
            editor.commands.setContent("");
          }
        }
        // Snapshot the editor's *normalized* doc as the initial-state baseline.
        // Reading getJSON() after setContent() captures whatever shape TipTap
        // produces, so isDirty() compares apples-to-apples and won't false-fire
        // on round-trip drift (the bug this whole migration fixes).
        const initialDocJSON = editor ? JSON.stringify(editor.getJSON()) : "";
        dirtyClose.captureInitial({
          line: editItem.title,
          priority: editItem.priority || "",
          due: editItem.due_at ? editItem.due_at.slice(0, 10) : "",
          tags: [...(editItem.tags || [])],
          contexts: [...(editItem.contexts || [])],
          projects: [...(editItem.projects || [])],
          notes_doc: initialDocJSON,
        });
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
        const initialDocJSON = editor ? JSON.stringify(editor.getJSON()) : "";
        dirtyClose.captureInitial({
          line: "",
          priority: activeSession?.priority || "",
          due: "",
          tags: [...mergedTags],
          contexts: [...mergedContexts],
          projects: [...mergedProjects],
          notes_doc: initialDocJSON,
        });
      }
    }
    prevOpenRef.current = open;
    // dirtyClose.cancelClosePrompt / captureInitial are useCallback-stabilized
    // in useDirtyClose (identity never changes), so listing them here doesn't
    // add re-runs beyond what this effect already does.
  }, [open, isEditMode, editItem, editor, defaultContexts, defaultProjects, defaultTags, dirtyClose.cancelClosePrompt, dirtyClose.captureInitial]);

  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        e.preventDefault();
        if (dirtyClose.showClosePrompt) { dirtyClose.cancelClosePrompt(); return; }
        requestClose();
        return;
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && open) {
        // Cmd+Enter: save AND close (existing behavior)
        e.preventDefault();
        handleSubmit(e as any);
        return;
      }
      if ((e.metaKey || e.ctrlKey) && (e.key === 's' || e.key === 'S') && open) {
        // Cmd+S: save in place (edit mode). Falls back to save+close in
        // create mode or when no onSaveStay handler is wired.
        e.preventDefault();
        if (isEditMode && onSaveStay) {
          doSaveStay();
        } else {
          handleSubmit(e as any);
        }
        return;
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [open, dirtyClose.showClosePrompt, dirtyClose.cancelClosePrompt, line, priority, due, tags, contexts, projects, editItem, onUpdate, onSaveStay, onSubmit, settings, isEditMode]);

  if (!open) return null;

  function doSave(forceInbox?: boolean) {
    // Send both: JSON is the source of truth, HTML is the write-time cache.
    // Backend stores both directly (no Go-side rendering), keeping TipTap as
    // the single source of HTML rendering for GUI-saved content.
    const notes_doc = editor ? JSON.stringify(editor.getJSON()) : "";
    const notes_html = editor?.getHTML() || "";
    let cleanTags = tags.map(t => t.replace(/^#/, ''));
    const cleanContexts = contexts.map(c => c.replace(/^@/, ''));
    const cleanProjects = projects.map(p => p.replace(/^\+/, ''));
    if (!isEditMode && cleanTags.length === 0 && cleanProjects.length === 0
        && settings.defaultTags && settings.defaultTags.length > 0) {
      cleanTags = [...settings.defaultTags];
    }
    if (isEditMode && editItem && onUpdate) {
      const updated: ItemRow = { ...editItem, title: line, tags: cleanTags, contexts: cleanContexts,
        projects: cleanProjects, notes_doc, notes_html, notes_html_version: NOTES_HTML_VERSION };
      if (priority) updated.priority = priority; else delete updated.priority;
      if (due) updated.due_at = due; else delete updated.due_at;
      onUpdate(updated);
    } else {
      const extras: any = { tags: cleanTags, contexts: cleanContexts, projects: cleanProjects };
      if (priority) extras.priority = priority;
      if (due) extras.due = due;
      if (notes_doc) {
        extras.notes_doc = notes_doc;
        extras.notes_html = notes_html;
        extras.notes_html_version = NOTES_HTML_VERSION;
      }
      if (forceInbox) extras.inbox = true;
      onSubmit(line, extras);
    }
  }

  // Save without closing. Edit mode only — refreshes the initial-state
  // snapshot after the save so isDirty() correctly reports clean afterward.
  async function doSaveStay() {
    if (!isEditMode || !editItem || !onSaveStay) return;
    const notes_doc = editor ? JSON.stringify(editor.getJSON()) : "";
    const notes_html = editor?.getHTML() || "";
    const cleanTags = tags.map(t => t.replace(/^#/, ''));
    const cleanContexts = contexts.map(c => c.replace(/^@/, ''));
    const cleanProjects = projects.map(p => p.replace(/^\+/, ''));
    const updated: ItemRow = {
      ...editItem,
      title: line,
      tags: cleanTags,
      contexts: cleanContexts,
      projects: cleanProjects,
      notes_doc,
      notes_html,
      notes_html_version: NOTES_HTML_VERSION,
    };
    if (priority) updated.priority = priority; else delete updated.priority;
    if (due) updated.due_at = due; else delete updated.due_at;
    try {
      await onSaveStay(updated);
      // Re-baseline so the just-saved state reads as clean for isDirty().
      dirtyClose.captureInitial({
        line,
        priority,
        due,
        tags: [...cleanTags],
        contexts: [...cleanContexts],
        projects: [...cleanProjects],
        notes_doc,
      });
    } catch (err) {
      console.error('Save (stay) failed:', err);
    }
  }

  // Thin wrapper so JSX call sites (Escape key, header "Close", footer
  // "Cancel") keep calling requestClose() exactly as before; the actual
  // isDirty/closeBehavior branching lives in useDirtyClose().
  function requestClose() {
    dirtyClose.requestClose();
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    doSave();
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

  // Create-mode only: routes the new item straight to Inbox (via doSave's
  // forceInbox flag, which sets extras.inbox = true) instead of wherever it
  // would otherwise land. The button rendering itself below is still gated
  // on `!isEditMode`; this handler just names the intent clearly.
  function handleSendToInbox(e: React.MouseEvent) {
    e.preventDefault();
    doSave(true);
  }

  return (
    <>
      <style>{`
        .tiptap-editor {
          padding: 8px 12px 12px 16px;
          outline: none;
          caret-color: var(--term-fg);
        }
        .tiptap-editor .ProseMirror {
          outline: none;
        }
        .tiptap-editor .ProseMirror-focused {
          outline: none;
        }
        .tiptap-editor .ProseMirror > * {
          margin: 0;
        }
        .tiptap-editor .ProseMirror > * + * {
          margin-top: 0.5em;
        }
        .tiptap-editor ul,
        .tiptap-editor ol {
          padding-left: 1.5em;
          margin: 0.5em 0;
        }
        .tiptap-editor ul li,
        .tiptap-editor ol li {
          margin: 0.05em 0;
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
        .tiptap-editor .ProseMirror > p:first-child {
          margin-top: 22px;
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
        .tiptap-editor .ProseMirror-gapcursor:after {
          display: none !important;
        }
        .tiptap-editor .ProseMirror-separator {
          display: none !important;
        }
        .tiptap-editor::selection,
        .tiptap-editor *::selection {
          background: rgba(96, 165, 250, 0.3);
        }
      `}</style>
      <div style={modalStyle}>
        <div className="terminal-card" style={{
        // Fullscreen mode fills the viewport; default mode is the original
        // 720x750 sizing but capped to the viewport so the footer can never
        // be pushed off-screen on short windows.
        width: isFullscreen ? '100vw' : '720px',
        maxWidth: isFullscreen ? '100vw' : '95vw',
        height: isFullscreen ? '100vh' : 'min(750px, calc(100vh - 40px))',
        maxHeight: isFullscreen ? '100vh' : 'calc(100vh - 40px)',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        position: 'relative',
        borderRadius: isFullscreen ? 0 : undefined,
      }}>
        {/* Fixed Header */}
        <div style={{ padding: '20px 20px 0 20px', flexShrink: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <div style={{ fontWeight: 600 }}>{isEditMode ? (isNoteMode ? 'Edit Note' : 'Edit Todo') : (isNoteMode ? 'Add Note' : 'Add Todo')}</div>
              {isEditMode && editItem && onConvertType && (
                <button
                  type="button"
                  className="badge"
                  onClick={() => onConvertType(editItem)}
                  style={{
                    padding: '4px 8px',
                    fontSize: '11px',
                    borderRadius: '4px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    opacity: 0.8
                  }}
                  title={isNoteMode ? 'Convert to Todo' : 'Convert to Note'}
                >
                  {isNoteMode ? <CheckSquare size={11} /> : <FileText size={11} />}
                  {isNoteMode ? '→ Todo' : '→ Note'}
                </button>
              )}
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <button
                type="button"
                className="badge"
                onClick={() => setIsFullscreen(v => !v)}
                style={{
                  padding: '6px 8px',
                  fontSize: '12px',
                  borderRadius: '6px',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                }}
                title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
              >
                {isFullscreen ? <Minimize2 size={13} /> : <Maximize2 size={13} />}
              </button>
              <button
                className="badge info"
                onClick={() => requestClose()}
                style={{
                  padding: '6px 9px',
                  fontSize: '12px',
                  borderRadius: '6px'
                }}
              >
                Close
              </button>
            </div>
          </div>
        </div>

        {/* Modal Body
            Scrolls vertically when content exceeds the available space between
            the fixed header and fixed footer. Native browser scrollbar is fine
            here — the previous CustomScrollbar caused a "jump to top" bug when
            TipTap re-rendered for markdown shortcuts (# for headings, - for
            bullets); native scroll behavior doesn't have that issue.
        */}
        <div style={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          overflowX: 'hidden',
          overflowAnchor: 'none',
          position: 'relative',
        }}>
          <div style={{
            padding: '8px 20px 16px 20px',
            overflowAnchor: 'none',
            position: 'relative',
          }}>
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '4px' }}>
              <div className="task-header-wrapper" style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                <span
                  className="badge warn task-lightning"
                  style={{
                    padding: '2px',
                    fontSize: '10px',
                    borderRadius: '3px',
                    display: 'inline-flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: '14px',
                    height: '14px',
                    lineHeight: '1'
                  }}
                >⚡</span>
                <label className="task-label" style={{ fontSize: '13px', opacity: 0.8, letterSpacing: '0.05em' }}>{isNoteMode ? 'NOTE' : 'TASK'}</label>
              </div>
              <div style={{
                display: 'flex',
                gap: '8px',
                alignItems: 'center'
              }}>
                {/* Priority/Date chips row - hidden in note mode */}
                {!isNoteMode && (
                <div style={{
                  display: 'flex',
                  gap: '0px',
                }}>
                  <button
                    type="button"
                    className={`badge ${priority === 'C' ? 'success' : ''}`}
                    onClick={() => setPriority(priority === 'C' ? '' : 'C')}
                    style={{
                      flex: '0 0 auto',
                      border: 'none',
                      borderRadius: '4px 0 0 4px',
                      fontSize: '11px',
                      padding: '4px 8px'
                    }}
                  >
                    LOW
                  </button>
                  <button
                    type="button"
                    className={`badge ${priority === 'B' ? 'info' : ''}`}
                    onClick={() => setPriority(priority === 'B' ? '' : 'B')}
                    style={{
                      flex: '0 0 auto',
                      border: 'none',
                      borderRadius: '0',
                      borderLeft: '1px solid var(--term-border)',
                      fontSize: '11px',
                      padding: '4px 8px'
                    }}
                  >
                    MED
                  </button>
                  <button
                    type="button"
                    className={`badge ${priority === 'A' ? 'warn' : ''}`}
                    onClick={() => setPriority(priority === 'A' ? '' : 'A')}
                    style={{
                      flex: '0 0 auto',
                      border: 'none',
                      borderRadius: '0',
                      borderLeft: '1px solid var(--term-border)',
                      fontSize: '11px',
                      padding: '4px 8px'
                    }}
                  >
                    HIGH
                  </button>
                  <button
                    type="button"
                    className={`badge ${priority === '' ? 'success' : ''}`}
                    onClick={() => setPriority('')}
                    style={{
                      flex: '0 0 auto',
                      border: 'none',
                      borderRadius: '0',
                      borderLeft: '1px solid var(--term-border)',
                      fontSize: '11px',
                      padding: '4px 8px'
                    }}
                  >
                    NA
                  </button>
                  <button
                    type="button"
                    className={`badge ${due ? 'info' : ''}`}
                    style={{
                      flex: '0 0 auto',
                      border: 'none',
                      borderRadius: '0 4px 4px 0',
                      borderLeft: '1px solid var(--term-border)',
                      fontSize: '11px',
                      padding: '4px 8px'
                    }}
                    onClick={(e) => {
                      e.stopPropagation();
                      if (due) {
                        setDue('');
                      } else {
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
                  >
                    {due ? `Due: ${due}` : 'DUE'}
                  </button>
                </div>
                )}

                <TemplatePickerCombobox
                  onSelect={handleTemplateSelect}
                  currentValues={{ contexts, projects, tags, ...(priority ? { priority } : {}) }}
                />
                <button
                  type="button"
                  className="badge"
                  onClick={() => setShowTemplateSaveDialog(true)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '4px',
                    fontSize: '13px',
                    padding: '8px 12px',
                    borderRadius: '6px',
                    borderColor: 'rgba(96, 165, 250, 0.5)',
                    opacity: 0.8,
                    height: '29px'
                  }}
                >
                  <Save size={12} />
                  Save Template
                </button>
              </div>
            </div>
            <ItemTitleInput
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
              <label style={{
                fontSize: '12px',
                marginBottom: '4px',
                display: 'block',
                opacity: 0.8
              }}>Projects</label>
              <ProjectsAutocomplete values={projects} onValuesChange={setProjects} placeholder="Add project..." />
            </div>

            <div>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block', opacity: 0.8 }}>Contexts</label>
              <ContextsAutocomplete values={contexts} onValuesChange={setContexts} placeholder="Add context..." />
            </div>

            <div>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block', opacity: 0.8 }}>Tags</label>
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
                  <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block', opacity: 0.8 }}>Session Context</label>
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
                    <div
                      onClick={toggleUseDefaults}
                      style={{ cursor: 'pointer', marginTop: '4px', flexShrink: 0 }}
                    >
                      {useDefaults ? (
                        <ToggleRight size={18} style={{ color: 'var(--term-accent)' }} />
                      ) : (
                        <ToggleLeft size={18} style={{ color: 'var(--term-dim)', opacity: 0.6 }} />
                      )}
                    </div>
                    <div style={{ fontSize: '11px', flex: 1, display: 'flex', flexWrap: 'wrap', gap: '6px', alignItems: 'center', marginTop: (useDefaults && (hasSessionContext || hasSearchFilters)) ? '0px' : '6px' }}>
                      {!hasSessionContext && !hasSearchFilters && <span className="text-dim">No active context</span>}
                      {useDefaults && (
                        <>
                          {displayContexts.map((ctx) => (
                            <span
                              key={`ctx-${ctx}`}
                              style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                padding: '4px 8px',
                                background: 'var(--term-panel)',
                                border: '1px solid var(--term-border)',
                                borderRadius: '4px',
                                fontSize: '12px',
                                color: 'var(--term-fg)'
                              }}
                            >
                              @{ctx}
                            </span>
                          ))}
                          {displayProjects.map((proj) => (
                            <span
                              key={`proj-${proj}`}
                              style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                padding: '4px 8px',
                                background: 'var(--term-panel)',
                                border: '1px solid var(--term-border)',
                                borderRadius: '4px',
                                fontSize: '12px',
                                color: 'var(--term-fg)'
                              }}
                            >
                              +{proj}
                            </span>
                          ))}
                          {displayTags.map((tag) => (
                            <span
                              key={`tag-${tag}`}
                              style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                padding: '4px 8px',
                                background: 'var(--term-panel)',
                                border: '1px solid var(--term-border)',
                                borderRadius: '4px',
                                fontSize: '12px',
                                color: 'var(--term-fg)'
                              }}
                            >
                              #{tag}
                            </span>
                          ))}
                          {activeSession?.priority && (
                            <span
                              style={{
                                display: 'inline-flex',
                                alignItems: 'center',
                                padding: '4px 8px',
                                background: 'var(--term-panel)',
                                border: '1px solid var(--term-border)',
                                borderRadius: '4px',
                                fontSize: '12px',
                                color: 'var(--term-fg)'
                              }}
                            >
                              Pri: {activeSession.priority}
                            </span>
                          )}
                        </>
                      )}
                      {!useDefaults && (hasSessionContext || hasSearchFilters) && <span className="text-dim">Context disabled</span>}
                    </div>
                  </div>
                </div>
              );
            })()}

            {isEditMode && (
              <div>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '4px', height: '16px' }}>
                  <label style={{ fontSize: '12px', lineHeight: '16px', opacity: 0.8 }}>Apply Context</label>
                  <div
                    style={{ display: 'flex', alignItems: 'center', gap: '8px', height: '16px', cursor: 'pointer' }}
                    onClick={() => setMergeContext(!mergeContext)}
                  >
                    {mergeContext ? (
                      <ToggleRight size={18} style={{ color: 'var(--term-accent)' }} />
                    ) : (
                      <ToggleLeft size={18} style={{ color: 'var(--term-dim)', opacity: 0.6 }} />
                    )}
                    <label style={{ fontSize: '11px', cursor: 'pointer', lineHeight: '16px', margin: 0, opacity: 0.8 }}>
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

          {!descriptionExpanded && (
            <NotesEditorField
              editor={editor}
              editorContainerRef={editorContainerRef}
              isFullscreen={isFullscreen}
              onExpand={() => setDescriptionExpanded(true)}
            />
          )}
        </form>
          </div>
        </div>

        {/* Fixed Footer — pinned to the bottom of the card via flexShrink: 0
            so a long body can scroll while the save button stays in view. */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '16px 20px 20px 20px',
          borderTop: '1px solid var(--term-border)',
          flexShrink: 0,
        }}>
          <div style={{ display: 'flex', gap: '8px' }}>
            <DeleteConfirmControl
              show={isEditMode && !!onDelete}
              confirming={showDeleteConfirm}
              onRequestConfirm={() => setShowDeleteConfirm(true)}
              onConfirmDelete={() => { onDelete!(editItem!.id); onOpenChange(false); }}
            />
            <button type="button" className="badge" onClick={(e) => { e.preventDefault(); handleClear(); }} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', background: 'var(--term-bg)', border: '1px solid var(--term-border)' }}>Clear</button>
            <button type="button" className="badge" onClick={() => requestClose()} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', background: 'var(--term-bg)', border: '1px solid var(--term-border)' }}>Cancel</button>
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            {!isEditMode && (
              <button type="button" className="badge" onClick={handleSendToInbox} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }} title="Create and send to Inbox for later review">→ Inbox</button>
            )}
            {isEditMode && onSaveStay && (
              <button
                type="button"
                className="badge"
                onClick={(e) => { e.preventDefault(); doSaveStay(); }}
                style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
                title="Save and keep editing (⌘S)"
              >
                Save
              </button>
            )}
            <button
              type="button"
              className="badge success"
              onClick={(e) => { e.preventDefault(); handleSubmit(e as any); }}
              style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
              title={isEditMode ? 'Save and close (⌘↵)' : 'Create (⌘↵)'}
            >
              {isEditMode ? (onSaveStay ? 'Save & Close' : 'Save') : 'Create'}
            </button>
          </div>
        </div>

        <ClosePromptDialog
          open={dirtyClose.showClosePrompt}
          pendingBehavior={dirtyClose.pendingBehavior}
          onSelectBehavior={dirtyClose.setPendingBehavior}
          onDiscard={dirtyClose.confirmDiscard}
          onSaveAndClose={dirtyClose.confirmSaveAndClose}
          onStay={dirtyClose.cancelClosePrompt}
        />
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

      <ExpandedNotesEditorModal
        open={descriptionExpanded}
        editor={editor}
        editorContainerRef={editorContainerRef}
        onClose={() => setDescriptionExpanded(false)}
        onSaveAndClose={(e) => {
          setDescriptionExpanded(false);
          handleSubmit(e as any);
        }}
      />
    </div>
    </>
  );
}
