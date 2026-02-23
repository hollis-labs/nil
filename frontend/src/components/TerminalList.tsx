import * as React from "react";
import { useTermTheme } from "@/theme/ThemeProvider";
import { TermTheme } from "@/theme/theme";
import CustomScrollbar from "@/components/CustomScrollbar";
import { ChevronUp, ChevronDown, Check, Archive, Trash2, Edit3, Pin } from "lucide-react";

export type ItemRow = {
  id: number; title: string; priority?: string;
  due_at?: string; created_at?: string; threshold_at?: string;
  completed: boolean; archived: boolean;
  projects: string[]; contexts: string[]; tags: string[];
  notes_md?: string;
  section: string; // now, soon, anytime
  pinned?: boolean;
  type?: string; // 'todo' | 'note'
  inbox?: boolean;
};

type ViewMode = 'scope' | 'date';
type AppMode = 'todos' | 'notes';

function fmtDateLabel(iso?: string) {
  const d = iso ? new Date(iso) : new Date();
  return d.toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
}
function ymd(iso?: string) {
  const d = iso ? new Date(iso) : new Date();
  return d.toISOString().slice(0,10);
}

function getTagColor(index: number, theme: TermTheme): string {
  // Use theme colors if available, fallback to defaults
  const colors = [
    theme.tagColor1 || '#b4b8c0',
    theme.tagColor2 || '#999da5',
    theme.tagColor3 || '#7e8289'
  ];
  return colors[Math.min(index, 2)];
}

function getProjectColor(index: number, theme: TermTheme): string {
  // Use theme colors if available, fallback to calculated gradient
  if (theme.projectColor1 && theme.projectColor2 && theme.projectColor3) {
    const colors = [theme.projectColor1, theme.projectColor2, theme.projectColor3];
    return colors[Math.min(index, 2)];
  }
  // Fallback: calculate from accent
  const brightAccent = adjustBrightness(theme.accent, 15);
  const darkAccent = adjustBrightness(theme.accent, -25);
  const colors = [brightAccent, theme.accent, darkAccent];
  return colors[Math.min(index, 2)];
}

function adjustBrightness(hex: string, percent: number): string {
  const num = parseInt(hex.replace('#', ''), 16);
  const amt = Math.round(2.55 * percent);
  const R = Math.min(255, Math.max(0, (num >> 16) + amt));
  const G = Math.min(255, Math.max(0, ((num >> 8) & 0x00FF) + amt));
  const B = Math.min(255, Math.max(0, (num & 0x0000FF) + amt));
  return `#${((1 << 24) + (R << 16) + (G << 8) + B).toString(16).slice(1)}`;
}

type Props = {
  rows: ItemRow[];
  onToggle: (id: number, checked: boolean) => void;
  onOpenNotes: (row: ItemRow) => void;
  onMoveSection: (id: number, section: string) => void;
  onArchive: (id: number, archived: boolean) => void;
  onDelete: (id: number) => void;
  showCompleted: boolean;
  onEditItem: (row: ItemRow) => void;
  viewMode?: ViewMode;
  appMode?: AppMode;
  onOpenRadialMenu?: (todo: ItemRow, position: {x: number; y: number}) => void;
  closeRadialMenus?: boolean;
  hasActiveFilters?: boolean;
  animatingRow?: { id: number; action: string; phase?: 'collapsing' | 'expanding' } | null;
};

export default function TerminalList({ rows, onToggle, onOpenNotes, onMoveSection, onArchive, onDelete, showCompleted, onEditItem, viewMode = 'scope', appMode = 'todos', onOpenRadialMenu, closeRadialMenus = false, hasActiveFilters = false, animatingRow = null }: Props) {
  const { theme } = useTermTheme();
  const [draggedId, setDraggedId] = React.useState<number | null>(null);
  const [collapsedSections, setCollapsedSections] = React.useState<Record<string, boolean>>({
    done: false
  });
  const [hoveredRow, setHoveredRow] = React.useState<number | null>(null);
  const [deleteConfirm, setDeleteConfirm] = React.useState<{ id: number; position: { x: number; y: number } } | null>(null);
  const longPressTimer = React.useRef<NodeJS.Timeout | null>(null);
  const deleteTriggered = React.useRef<boolean>(false);

  React.useEffect(() => {
    if (closeRadialMenus) {
      setHoveredRow(null);
    }
  }, [closeRadialMenus]);

  const toggleSection = (section: string) => {
    setCollapsedSections(prev => ({ ...prev, [section]: !prev[section] }));
  };

  const sections = React.useMemo(() => {
    if (!rows || !Array.isArray(rows)) return { now: [], soon: [], anytime: [], done: [] };

    const today = ymd();
    const now: ItemRow[] = [];
    const soon: ItemRow[] = [];
    const anytime: ItemRow[] = [];
    const done: ItemRow[] = [];

    for (const r of rows) {
      if (r.completed) {
        done.push(r);
        continue;
      }

      const section = r.section || 'anytime';
      if (section === 'now') now.push(r);
      else if (section === 'soon') soon.push(r);
      else anytime.push(r);
    }

    const sortFn = (a: ItemRow, b: ItemRow) => {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
      return (a.priority||'Z').localeCompare(b.priority||'Z') ||
        (b.created_at||'').localeCompare(a.created_at||'');
    };

    [now, soon, anytime, done].forEach(list => list.sort(sortFn));

    return { now, soon, anytime, done };
  }, [rows]);

  const dateGroups = React.useMemo(() => {
    const map: Record<string, ItemRow[]> = {};
    const withDates = rows.filter(r => !r.completed && r.due_at);

    for (const r of withDates) {
      const key = ymd(r.due_at);
      (map[key] ||= []).push(r);
    }

    Object.values(map).forEach(list => list.sort((a,b) => {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
      return (a.priority||'Z').localeCompare(b.priority||'Z') ||
        (b.created_at||'').localeCompare(a.created_at||'');
    }));

    return Object.entries(map).sort((a,b)=>a[0].localeCompare(b[0]));
  }, [rows]);

  const noteSections = React.useMemo(() => {
    if (appMode !== 'notes') return { pinned: [], notes: [] };
    const pinned: ItemRow[] = [];
    const notes: ItemRow[] = [];
    for (const r of rows) {
      if (r.pinned) pinned.push(r);
      else notes.push(r);
    }
    // Sort newest first
    const sortFn = (a: ItemRow, b: ItemRow) =>
      (b.created_at || '').localeCompare(a.created_at || '');
    pinned.sort(sortFn);
    notes.sort(sortFn);
    return { pinned, notes };
  }, [rows, appMode]);

  const handleDragStart = (e: React.DragEvent, id: number) => {
    setDraggedId(id);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  };

  const handleDrop = (e: React.DragEvent, section: string) => {
    e.preventDefault();
    if (draggedId !== null) {
      onMoveSection(draggedId, section);
      setDraggedId(null);
    }
  };

  const renderSection = (title: string, items: ItemRow[], sectionKey: string, canCollapse = true) => {
    const isCollapsed = collapsedSections[sectionKey];

    return (
        <div
            onDragOver={handleDragOver}
            onDrop={(e) => handleDrop(e, sectionKey)}
            style={{ marginBottom: '20px' }}
        >
          <div
              className="date-header"
              style={{
                cursor: canCollapse ? 'pointer' : 'default',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                userSelect: 'none',
                marginRight: '25px'
              }}
              onClick={() => canCollapse && toggleSection(sectionKey)}
          >
            {canCollapse && <span>{isCollapsed ? '▸' : '▾'}</span>}
            <span>{title}</span>
            <span style={{ fontSize: '0.75em', opacity: 0.6 }}>({items.length})</span>
          </div>
          {!isCollapsed && (
              <>
                {items.length === 0 ? (
                    <div style={{ padding: '12px', textAlign: 'center', color: 'var(--term-dim)', fontSize: '12px' }}>
                      Drop items here
                    </div>
                ) : (
                    items.map((r, idx) => (
                        <div
                            key={r.id}
                            className={`list-row ${animatingRow?.id === r.id && animatingRow.phase === 'collapsing' ? 'row-animating' : ''} ${animatingRow?.id === r.id && animatingRow.phase === 'expanding' ? 'row-expanding' : ''}`}
                            draggable
                            onDragStart={(e) => handleDragStart(e, r.id)}
                            onMouseEnter={() => {
                              setHoveredRow(r.id);
                            }}
                            onMouseLeave={() => {
                              setHoveredRow(null);
                              if (longPressTimer.current) {
                                clearTimeout(longPressTimer.current);
                                longPressTimer.current = null;
                              }
                            }}
                            onMouseDown={(e) => {
                              deleteTriggered.current = false;
                              longPressTimer.current = setTimeout(() => {
                                deleteTriggered.current = true;
                                if (onOpenRadialMenu) {
                                  onOpenRadialMenu(r, { x: e.clientX, y: e.clientY });
                                }
                              }, 1000);
                            }}
                            onMouseUp={() => {
                              if (longPressTimer.current) {
                                clearTimeout(longPressTimer.current);
                                longPressTimer.current = null;
                              }
                            }}
                            onTouchStart={(e) => {
                              deleteTriggered.current = false;
                              const touch = e.touches[0];
                              longPressTimer.current = setTimeout(() => {
                                deleteTriggered.current = true;
                                if (onOpenRadialMenu) {
                                  onOpenRadialMenu(r, { x: touch.clientX, y: touch.clientY });
                                }
                              }, 1000);
                            }}
                            onTouchEnd={() => {
                              if (longPressTimer.current) {
                                clearTimeout(longPressTimer.current);
                                longPressTimer.current = null;
                              }
                            }}
                            onClick={(e) => {
                              // Don't open edit if delete confirmation is showing
                              if (deleteConfirm) {
                                e.preventDefault();
                                e.stopPropagation();
                                return;
                              }
                              
                              const target = e.target as HTMLElement;
                              const isCheckboxClick = target.closest('.checkbox');

                              if (!isCheckboxClick) {
                                e.preventDefault();
                                setHoveredRow(null);
                                onEditItem(r);
                              }
                            }}
                             style={{
                               cursor: r.completed ? 'grab' : 'pointer',
                               padding: '12px 20px',
                               marginRight: '25px',
                               display: 'flex',
                               gap: '12px',
                               position: 'relative',
                               borderTop: idx === 0 ? '1px solid var(--term-border)' : 'none',
                               borderBottom: idx === items.length - 1 ? 'none' : '1px solid var(--term-border)'
                             }}
                         >
                          {(r as any).pinned && (
                            <div style={{
                              position: 'absolute',
                              left: '4px',
                              top: '18px',
                              display: 'flex',
                              alignItems: 'center'
                            }}>
                              <Pin size={12} style={{ color: 'var(--term-border)', fill: 'var(--term-border)' }} />
                            </div>
                          )}
                          <div
                            onClick={(e) => {
                              e.stopPropagation();
                              onToggle(r.id, !r.completed);
                            }}
                            style={{
                              width: '18px',
                              height: '18px',
                              minWidth: '18px',
                              border: r.completed ? '1px solid var(--term-border)' : '1px solid rgba(0, 0, 0, 0.4)',
                              borderRadius: '4px',
                              display: 'flex',
                              alignItems: 'flex-start',
                              justifyContent: 'center',
                              background: r.completed ? 'transparent' : 'var(--term-border)',
                              cursor: 'pointer',
                              marginTop: '2px',
                              flexShrink: 0
                            }}
                          >
                            {r.completed && <Check size={14} style={{ color: 'var(--term-info)' }} />}
                          </div>
                          <div
                              style={{ minWidth: 0, opacity: r.completed ? 0.5 : 1, flex: 1, display: 'flex', alignItems: 'flex-start', gap: '8px' }}
                          >
                            <div style={{ flex: 1, minWidth: 0 }}>
                              <div className={`title ${r.title.includes("http") ? "underlined" : ""}`} style={{ textDecoration: r.completed ? 'line-through' : 'none', fontSize: '13px' }}>
                                {r.title}{" "}
                                <span style={{ color: 'var(--term-info)', opacity: 0.8, fontSize: '0.95em' }}>
                          {r.contexts.map((c: string) => `@${c}`).join(" ")}
                        </span>
                                {r.priority === "A" && <span style={{ color: 'var(--term-success)', marginLeft: '6px' }}>★</span>}
                                {r.priority === "B" && <span style={{ color: 'var(--term-warn)', marginLeft: '6px' }}>⁙</span>}
                                {r.priority === "C" && <span style={{ color: 'var(--term-info)', marginLeft: '6px' }}>•</span>}
                              </div>
                              <div className="meta" style={{ fontSize: '11px' }}>
                                {r.projects.map((p: string, i: number) => <span key={"p"+p} style={{ color: getProjectColor(i, theme) }}>+{p}</span>)}
                                {r.tags.map((t: string, i: number) => <span key={"t"+t} style={{ color: getTagColor(i, theme) }}>#{t}</span>)}
                              </div>
                            </div>
                            {!r.completed && r.due_at && (
                              <span className="badge info" style={{ 
                                padding: '3px 6px', 
                                fontSize: '10px', 
                                borderRadius: '4px', 
                                opacity: 0.8,
                                flexShrink: 0,
                                alignSelf: 'flex-start'
                              }}>
                                {new Date(r.due_at).toLocaleDateString()}
                              </span>
                            )}
                          </div>
                          {animatingRow?.id === r.id && (
                            <div className="row-overlay" />
                          )}
                        </div>
                    ))
                )}
              </>
          )}
        </div>
    );
  };

  if (!rows || rows.length === 0) {
    return (
        <div className="terminal-card" style={{
          padding: '20px',
          flex: 1,
          minHeight: '300px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center'
        }}>
          {hasActiveFilters ? (
            <div style={{ 
              textAlign: 'center', 
              color: 'var(--term-dim)', 
              fontSize: '13px',
              maxWidth: '500px',
              lineHeight: '1.6'
            }}>
              <div style={{ marginBottom: '12px', fontSize: '14px', whiteSpace: 'nowrap' }}>
                This combination of filters did not produce any results.
              </div>
              <div style={{ marginBottom: '12px', opacity: 0.8 }}>
                Please expand your filter criteria.
              </div>
              <div style={{ 
                margin: '20px 0',
                fontSize: '13px',
                letterSpacing: '1px',
                opacity: 0.5
              }}>
                — OR —
              </div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px', fontSize: '13px' }}>
                <span>Press</span>
                <kbd style={{
                  padding: '4px 8px',
                  background: 'var(--term-bg)',
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  fontSize: '12px',
                  fontFamily: 'monospace',
                  fontWeight: 500
                }}>⌘</kbd>
                <span>+</span>
                <kbd style={{
                  padding: '4px 8px',
                  background: 'var(--term-bg)',
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  fontSize: '12px',
                  fontFamily: 'monospace',
                  fontWeight: 500
                }}>N</kbd>
                <span>to create a new {appMode === 'notes' ? 'note' : 'todo'}</span>
              </div>
            </div>
          ) : (
            <div style={{ textAlign: 'center', color: 'var(--term-dim)', fontSize: '14px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}>
              <span>No {appMode === 'notes' ? 'notes' : 'todos'} yet. Press</span>
              <kbd style={{
                padding: '4px 8px',
                background: 'var(--term-bg)',
                border: '1px solid var(--term-border)',
                borderRadius: '6px',
                fontSize: '13px',
                fontFamily: 'monospace',
                fontWeight: 500
              }}>⌘</kbd>
              <span>+</span>
              <kbd style={{
                padding: '4px 8px',
                background: 'var(--term-bg)',
                border: '1px solid var(--term-border)',
                borderRadius: '6px',
                fontSize: '13px',
                fontFamily: 'monospace',
                fontWeight: 500
              }}>N</kbd>
              <span>to create one.</span>
            </div>
          )}
        </div>
    );
  }

  const total = rows.length;
  const done = rows.filter(r=>r.completed).length;
  const pending = total - done;
  const pct = Math.round((done/Math.max(total,1))*100);

  // Helper to strip HTML tags for note preview
  function stripHtml(html: string): string {
    const tmp = document.createElement('div');
    tmp.innerHTML = html;
    return tmp.textContent || tmp.innerText || '';
  }

  // Render a single note row (no checkbox, no priority, shows preview)
  const renderNoteRow = (r: ItemRow, idx: number, listLen: number) => (
    <div
      key={r.id}
      className={`list-row ${animatingRow?.id === r.id && animatingRow.phase === 'collapsing' ? 'row-animating' : ''} ${animatingRow?.id === r.id && animatingRow.phase === 'expanding' ? 'row-expanding' : ''}`}
      onMouseEnter={() => setHoveredRow(r.id)}
      onMouseLeave={() => {
        setHoveredRow(null);
        if (longPressTimer.current) {
          clearTimeout(longPressTimer.current);
          longPressTimer.current = null;
        }
      }}
      onMouseDown={(e) => {
        deleteTriggered.current = false;
        longPressTimer.current = setTimeout(() => {
          deleteTriggered.current = true;
          if (onOpenRadialMenu) {
            onOpenRadialMenu(r, { x: e.clientX, y: e.clientY });
          }
        }, 1000);
      }}
      onMouseUp={() => {
        if (longPressTimer.current) {
          clearTimeout(longPressTimer.current);
          longPressTimer.current = null;
        }
      }}
      onClick={(e) => {
        if (deleteConfirm) { e.preventDefault(); e.stopPropagation(); return; }
        e.preventDefault();
        setHoveredRow(null);
        onEditItem(r);
      }}
      style={{
        cursor: 'pointer',
        padding: '12px 20px',
        marginRight: '25px',
        display: 'flex',
        gap: '12px',
        position: 'relative',
        borderTop: idx === 0 ? '1px solid var(--term-border)' : 'none',
        borderBottom: idx === listLen - 1 ? 'none' : '1px solid var(--term-border)'
      }}
    >
      {r.pinned && (
        <div style={{
          position: 'absolute',
          left: '4px',
          top: '18px',
          display: 'flex',
          alignItems: 'center'
        }}>
          <Pin size={12} style={{ color: 'var(--term-border)', fill: 'var(--term-border)' }} />
        </div>
      )}
      <div style={{ minWidth: 0, flex: 1 }}>
        <div className="title" style={{ fontSize: '13px' }}>
          {r.title}{" "}
          <span style={{ color: 'var(--term-info)', opacity: 0.8, fontSize: '0.95em' }}>
            {r.contexts.map((c: string) => `@${c}`).join(" ")}
          </span>
        </div>
        {r.notes_md && (
          <div style={{
            fontSize: '11px',
            color: 'var(--term-dim)',
            opacity: 0.7,
            marginTop: '2px',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            maxWidth: '500px'
          }}>
            {stripHtml(r.notes_md).slice(0, 80)}
          </div>
        )}
        <div className="meta" style={{ fontSize: '11px' }}>
          {r.projects.map((p: string, i: number) => <span key={"p"+p} style={{ color: getProjectColor(i, theme) }}>+{p}</span>)}
          {r.tags.map((t: string, i: number) => <span key={"t"+t} style={{ color: getTagColor(i, theme) }}>#{t}</span>)}
        </div>
      </div>
      {animatingRow?.id === r.id && (
        <div className="row-overlay" />
      )}
    </div>
  );

  // Notes view mode
  if (appMode === 'notes') {
    return (
      <>
        <style>{`
          @keyframes rowCollapse {
            0% { max-height: 100px; opacity: 1; transform: scaleY(1); }
            50% { max-height: 100px; opacity: 0.5; }
            100% { max-height: 0; opacity: 0; transform: scaleY(0); margin: 0; padding: 0; }
          }
          @keyframes rowExpand {
            0% { max-height: 0; opacity: 0; transform: scaleY(0); margin: 0; padding: 0; }
            50% { max-height: 100px; opacity: 0.5; }
            100% { max-height: 100px; opacity: 1; transform: scaleY(1); }
          }
          .row-animating { animation: rowCollapse 0.24s ease-out forwards; transform-origin: center; overflow: hidden; }
          .row-expanding { animation: rowExpand 0.24s ease-out forwards; transform-origin: center; overflow: hidden; }
          .row-overlay { position: absolute; inset: 0; background: rgba(0, 0, 0, 0.15); backdrop-filter: blur(1px); z-index: 10; pointer-events: none; }
        `}</style>
        <div className="terminal-card" style={{ flex: 1, minHeight: 0, padding: '20px', position: 'relative', display: 'flex', flexDirection: 'column' }}>
          <div style={{ flex: 1, minHeight: 0 }}>
            <CustomScrollbar style={{ height: '100%' }}>
              <div style={{ paddingBottom: '20px' }}>
                {noteSections.pinned.length > 0 && (
                  <div style={{ marginBottom: '20px' }}>
                    <div className="date-header" style={{
                      display: 'flex', alignItems: 'center', gap: '8px', userSelect: 'none', marginRight: '25px'
                    }}>
                      <span>Pinned</span>
                      <span style={{ fontSize: '0.75em', opacity: 0.6 }}>({noteSections.pinned.length})</span>
                    </div>
                    {noteSections.pinned.map((r, idx) => renderNoteRow(r, idx, noteSections.pinned.length))}
                  </div>
                )}
                <div style={{ marginBottom: '20px' }}>
                  <div className="date-header" style={{
                    display: 'flex', alignItems: 'center', gap: '8px', userSelect: 'none', marginRight: '25px'
                  }}>
                    <span>Notes</span>
                    <span style={{ fontSize: '0.75em', opacity: 0.6 }}>({noteSections.notes.length})</span>
                  </div>
                  {noteSections.notes.length === 0 ? (
                    <div style={{ padding: '12px', textAlign: 'center', color: 'var(--term-dim)', fontSize: '12px' }}>
                      No notes yet
                    </div>
                  ) : (
                    noteSections.notes.map((r, idx) => renderNoteRow(r, idx, noteSections.notes.length))
                  )}
                </div>
              </div>
            </CustomScrollbar>
          </div>

          <div className="summary" style={{
            background: 'var(--term-panel)',
            padding: '16px 20px 0 0',
            boxShadow: '0 -4px 6px -1px rgba(0, 0, 0, 0.1)',
            marginTop: '20px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center'
          }}>
            <span>{rows.length} note{rows.length !== 1 ? 's' : ''}</span>
          </div>
        </div>
      </>
    );
  }

  if (viewMode === 'date') {
    return (
        <div className="terminal-card" style={{ flex: 1, minHeight: 0, padding: '20px', position: 'relative', display: 'flex', flexDirection: 'column' }}>
          <div style={{ flex: 1, minHeight: 0 }}>
            <CustomScrollbar style={{ height: '100%' }}>
            <div style={{ paddingBottom: '20px' }}>
              {dateGroups.length === 0 ? (
                  <div style={{ textAlign: 'center', color: 'var(--term-dim)', padding: '40px 20px' }}>
                    No todos with due dates.
                  </div>
              ) : (
                  dateGroups.map(([key, list], gi) => (
                      <div key={key} style={{ marginBottom: '20px' }}>
                        <div className="date-header" style={{ marginRight: '25px' }}>
                          {fmtDateLabel(key)}
                        </div>
                        {list.map((r, idx) => (
                            <div
                                key={r.id}
                                className="list-row"
                            onMouseEnter={() => {
                              setHoveredRow(r.id);
                            }}
                                onMouseLeave={() => setHoveredRow(null)}
                                onClick={(e) => {
                              const target = e.target as HTMLElement;
                              const isCheckboxClick = target.closest('.checkbox');

                              if (!isCheckboxClick) {
                                e.preventDefault();
                                setHoveredRow(null);
                                onEditItem(r);
                              }
                            }}
                            style={{
                              cursor: r.completed ? 'default' : 'pointer',
                                  padding: '12px 20px',
                                  marginRight: '25px',
                                  display: 'flex',
                                  gap: '12px',
                                  position: 'relative',
                                  borderTop: idx === 0 ? '1px solid var(--term-border)' : 'none',
                                  borderBottom: idx === list.length - 1 ? 'none' : '1px solid var(--term-border)'
                                }}
                            >
                              {(r as any).pinned && (
                                <div style={{
                                  position: 'absolute',
                                  left: '4px',
                                  top: '18px',
                                  display: 'flex',
                                  alignItems: 'center'
                                }}>
                                  <Pin size={12} style={{ color: 'var(--term-border)', fill: 'var(--term-border)' }} />
                                </div>
                              )}
                              <div
                                onClick={(e) => {
                                  e.stopPropagation();
                                  onToggle(r.id, !r.completed);
                                }}
                                style={{
                                  width: '18px',
                                  height: '18px',
                                  minWidth: '18px',
                                  border: r.completed ? '1px solid var(--term-border)' : '1px solid rgba(0, 0, 0, 0.4)',
                                  borderRadius: '4px',
                                  display: 'flex',
                                  alignItems: 'center',
                                  justifyContent: 'center',
                                  background: r.completed ? 'transparent' : 'var(--term-border)',
                                  cursor: 'pointer'
                                }}
                              >
                                {r.completed && <Check size={14} style={{ color: 'var(--term-info)' }} />}
                              </div>
                              <div
                                  style={{ minWidth: 0, opacity: r.completed ? 0.5 : 1, flex: 1 }}
                              >
                                <div className={`title ${r.title.includes("http") ? "underlined" : ""}`}>
                                  {r.title}{" "}
                                  <span style={{ color: 'var(--term-info)', opacity: 0.8, fontSize: '0.95em' }}>
                          {r.contexts.map((c: string) => `@${c}`).join(" ")}
                        </span>
                                  {r.priority === "A" && <span style={{ color: 'var(--term-warn)', marginLeft: '6px' }}>★</span>}
                                  {r.priority === "B" && <span style={{ color: 'var(--term-info)', marginLeft: '6px' }}>⁙</span>}
                                  {r.priority === "C" && <span style={{ color: 'var(--term-accent)', marginLeft: '6px' }}>•</span>}
                                </div>
                                <div className="meta">
                                  {r.projects.map((p: string, i: number) => <span key={"p"+p} style={{ color: getProjectColor(i, theme) }}>+{p}</span>)}
                                  {r.tags.map((t: string, i: number) => <span key={"t"+t} style={{ color: getTagColor(i, theme) }}>#{t}</span>)}
                                </div>
                              </div>
                            </div>
                        ))}
                        {gi < dateGroups.length - 1 && <div className="separator" />}
                      </div>
                  ))
              )}
            </div>
            </CustomScrollbar>
          </div>

          <div className="summary" style={{
            background: 'var(--term-panel)',
            padding: '16px 20px 0 0',
            boxShadow: '0 -4px 6px -1px rgba(0, 0, 0, 0.1)',
            marginTop: '20px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center'
          }}>
          <span>
            {pct}% complete · <span style={{ color: 'var(--term-success)' }}>{done} done</span> · <span style={{ color: 'var(--term-info)' }}>{pending} pending</span>
          </span>
          </div>
        </div>
    );
  }

  return (
      <>
        <style>{`
          @keyframes rowCollapse {
            0% {
              max-height: 100px;
              opacity: 1;
              transform: scaleY(1);
            }
            50% {
              max-height: 100px;
              opacity: 0.5;
            }
            100% {
              max-height: 0;
              opacity: 0;
              transform: scaleY(0);
              margin: 0;
              padding: 0;
            }
          }
          
          @keyframes rowExpand {
            0% {
              max-height: 0;
              opacity: 0;
              transform: scaleY(0);
              margin: 0;
              padding: 0;
            }
            50% {
              max-height: 100px;
              opacity: 0.5;
            }
            100% {
              max-height: 100px;
              opacity: 1;
              transform: scaleY(1);
            }
          }
          
          .row-animating {
            animation: rowCollapse 0.24s ease-out forwards;
            transform-origin: center;
            overflow: hidden;
          }
          
          .row-expanding {
            animation: rowExpand 0.24s ease-out forwards;
            transform-origin: center;
            overflow: hidden;
          }
          
          .row-overlay {
            position: absolute;
            inset: 0;
            background: rgba(0, 0, 0, 0.15);
            backdrop-filter: blur(1px);
            z-index: 10;
            pointer-events: none;
          }
        `}</style>
        <div className="terminal-card" style={{
          flex: 1,
          minHeight: 0,
          padding: '20px',
          position: 'relative',
          display: 'flex',
          flexDirection: 'column'
        }}>
          <div style={{ flex: 1, minHeight: 0 }}>
            <CustomScrollbar style={{ height: '100%' }}>
              <div style={{ paddingBottom: '20px' }}>
                {renderSection('Now', sections.now, 'now', true)}
                {renderSection('Soon', sections.soon, 'soon', true)}
                {renderSection('Anytime', sections.anytime, 'anytime', true)}
                {showCompleted && sections.done.length > 0 && renderSection('Done', sections.done, 'done', true)}
              </div>
            </CustomScrollbar>
          </div>

          <div className="summary" style={{
            background: 'var(--term-panel)',
            padding: '16px 20px 0 0',
            boxShadow: '0 -4px 6px -1px rgba(0, 0, 0, 0.1)',
            marginTop: '20px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center'
          }}>
          <span>
            {pct}% complete · <span style={{ color: 'var(--term-success)' }}>{done} done</span> · <span style={{ color: 'var(--term-info)' }}>{pending} pending</span>
          </span>
          </div>
        </div>
        
        {deleteConfirm && (
          <div
            style={{
              position: 'fixed',
              top: deleteConfirm.position.y,
              left: deleteConfirm.position.x,
              zIndex: 1100,
              backgroundColor: 'var(--term-bg)',
              border: '1px solid var(--term-border)',
              borderRadius: '6px',
              padding: '16px',
              boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.3)',
              display: 'flex',
              gap: '12px',
              flexDirection: 'column',
              minWidth: '200px',
            }}
          >
            <div style={{ fontSize: '14px', marginBottom: '4px' }}>Delete this todo?</div>
            <div style={{ display: 'flex', gap: '8px' }}>
              <button
                autoFocus
                onClick={() => {
                  setDeleteConfirm(null);
                  deleteTriggered.current = false;
                }}
                style={{
                  padding: '10px 16px',
                  fontSize: '13px',
                  borderRadius: '6px',
                  backgroundColor: 'var(--term-bg)',
                  color: 'var(--term-fg)',
                  border: '1px solid var(--term-border)',
                  cursor: 'pointer',
                }}
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  if (deleteConfirm) {
                    onDelete(deleteConfirm.id);
                    setDeleteConfirm(null);
                    deleteTriggered.current = false;
                  }
                }}
                style={{
                  padding: '10px 16px',
                  fontSize: '13px',
                  borderRadius: '6px',
                  backgroundColor: '#dc2626',
                  color: 'white',
                  border: 'none',
                  cursor: 'pointer',
                }}
              >
                Delete
              </button>
            </div>
          </div>
        )}
      </>
  );
}
