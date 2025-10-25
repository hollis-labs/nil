import * as React from "react";
import { useTermTheme } from "@/theme/ThemeProvider";
import { TermTheme } from "@/theme/theme";
import CustomScrollbar from "@/components/CustomScrollbar";
import { ChevronUp, ChevronDown, Check, Archive, Trash2, Edit3 } from "lucide-react";

export type TodoRow = {
  id: number; title: string; priority?: string;
  due_at?: string; created_at?: string; threshold_at?: string;
  completed: boolean; archived: boolean;
  projects: string[]; contexts: string[]; tags: string[];
  notes_md?: string;
  section: string; // now, soon, anytime
};

type ViewMode = 'scope' | 'date';

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
  rows: TodoRow[];
  onToggle: (id: number, checked: boolean) => void;
  onOpenNotes: (row: TodoRow) => void;
  onMoveSection: (id: number, section: string) => void;
  onArchive: (id: number, archived: boolean) => void;
  onDelete: (id: number) => void;
  showCompleted: boolean;
  onEditTodo: (row: TodoRow) => void;
  viewMode?: ViewMode;
  closeRadialMenus?: boolean;
  settingsButton?: React.ReactNode;
};

export default function TerminalList({ rows, onToggle, onOpenNotes, onMoveSection, onArchive, onDelete, showCompleted, onEditTodo, viewMode = 'scope', closeRadialMenus = false, settingsButton }: Props) {
  const { theme } = useTermTheme();
  const [draggedId, setDraggedId] = React.useState<number | null>(null);
  const [collapsedSections, setCollapsedSections] = React.useState<Record<string, boolean>>({
    done: false
  });
  const [isRadialNavOpen, setIsRadialNavOpen] = React.useState<number | null>(null);
  const [hoveredRow, setHoveredRow] = React.useState<number | null>(null);
  const modalOverlayRef = React.useRef<HTMLDivElement>(null);
  const [deleteConfirm, setDeleteConfirm] = React.useState<{ id: number; position: { x: number; y: number } } | null>(null);
  const longPressTimer = React.useRef<NodeJS.Timeout | null>(null);
  const deleteTriggered = React.useRef<boolean>(false);

  React.useEffect(() => {
    if (isRadialNavOpen !== null && modalOverlayRef.current) {
      modalOverlayRef.current.focus();
    }
  }, [isRadialNavOpen]);

  React.useEffect(() => {
    if (closeRadialMenus) {
      setIsRadialNavOpen(null);
      setHoveredRow(null);
    }
  }, [closeRadialMenus]);

  const toggleSection = (section: string) => {
    setCollapsedSections(prev => ({ ...prev, [section]: !prev[section] }));
  };

  const sections = React.useMemo(() => {
    if (!rows || !Array.isArray(rows)) return { now: [], soon: [], anytime: [], done: [] };

    const today = ymd();
    const now: TodoRow[] = [];
    const soon: TodoRow[] = [];
    const anytime: TodoRow[] = [];
    const done: TodoRow[] = [];

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

    const sortFn = (a: TodoRow, b: TodoRow) =>
        (a.priority||'Z').localeCompare(b.priority||'Z') ||
        (b.created_at||'').localeCompare(a.created_at||'');

    [now, soon, anytime, done].forEach(list => list.sort(sortFn));

    return { now, soon, anytime, done };
  }, [rows]);

  const dateGroups = React.useMemo(() => {
    const map: Record<string, TodoRow[]> = {};
    const withDates = rows.filter(r => !r.completed && r.due_at);

    for (const r of withDates) {
      const key = ymd(r.due_at);
      (map[key] ||= []).push(r);
    }

    Object.values(map).forEach(list => list.sort((a,b) =>
        (a.priority||'Z').localeCompare(b.priority||'Z') ||
        (b.created_at||'').localeCompare(a.created_at||'')
    ));

    return Object.entries(map).sort((a,b)=>a[0].localeCompare(b[0]));
  }, [rows]);

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

  const DirectionalMenu = ({ row }: { row: TodoRow }) => {
    const section = row.section || 'anytime';
    const isOpen = isRadialNavOpen === row.id;
    const isHovered = hoveredRow === row.id;
    const autoCloseTimerRef = React.useRef<NodeJS.Timeout | null>(null);

    const menuItems: any[] = [];

    if (section === 'anytime') {
      menuItems.push({ dir: 'up', label: 'N', section: 'now', delay: 0 });
      menuItems.push({ dir: 'up', label: 'S', section: 'soon', delay: 1 });
    } else if (section === 'soon') {
      menuItems.push({ dir: 'up', label: 'N', section: 'now', delay: 0 });
      menuItems.push({ dir: 'down', label: 'A', section: 'anytime', delay: 0 });
    } else if (section === 'now') {
      menuItems.push({ dir: 'down', label: 'S', section: 'soon', delay: 0 });
      menuItems.push({ dir: 'down', label: 'A', section: 'anytime', delay: 1 });
    }

    menuItems.push({ dir: 'left', label: <Edit3 size={12} />, action: 'edit', delay: 0 });
    menuItems.push({ dir: 'left', label: <Check size={12} />, action: 'complete', delay: 1 });
    menuItems.push({ dir: 'left', label: <Trash2 size={12} />, action: 'delete', delay: 2 });
    menuItems.push({ dir: 'left', label: <Archive size={12} />, action: 'archive', delay: 3 });

    const menuRef = React.useRef<HTMLDivElement>(null);



    React.useEffect(() => {
      if (!isOpen) {
        if (autoCloseTimerRef.current) {
          clearTimeout(autoCloseTimerRef.current);
          autoCloseTimerRef.current = null;
        }
        return;
      }

      const startAutoCloseTimer = () => {
        if (autoCloseTimerRef.current) {
          clearTimeout(autoCloseTimerRef.current);
        }
        autoCloseTimerRef.current = setTimeout(() => {
          setIsRadialNavOpen(null);
        }, 3000);
      };

      const resetTimer = () => {
        startAutoCloseTimer();
      };

      startAutoCloseTimer();

      const menuElement = menuRef.current;
      if (menuElement) {
        menuElement.addEventListener('mousemove', resetTimer);
        menuElement.addEventListener('click', resetTimer);
      }

      return () => {
        if (autoCloseTimerRef.current) {
          clearTimeout(autoCloseTimerRef.current);
        }
        if (menuElement) {
          menuElement.removeEventListener('mousemove', resetTimer);
          menuElement.removeEventListener('click', resetTimer);
        }
      };
    }, [isOpen]);

    const handleAction = (item: any) => {
      if (item.action === 'complete') {
        onToggle(row.id, true);
      } else if (item.action === 'archive') {
        onArchive(row.id, true);
      } else if (item.action === 'delete') {
        onDelete(row.id);
      } else if (item.action === 'edit') {
        onEditTodo(row);
      } else if (item.section) {
        onMoveSection(row.id, item.section);
      }
      setIsRadialNavOpen(null);
    };

    const getPosition = (dir: string, index: number) => {
      const spacing = 35;
      const firstOffset = 32;
      const offset = firstOffset + (index * spacing);
      if (dir === 'up') return { bottom: `${offset}px`, left: '50%', transform: 'translateX(-50%)' };
      if (dir === 'down') return { top: `${offset}px`, left: '50%', transform: 'translateX(-50%)' };
      if (dir === 'left') return { right: `${offset}px`, top: '50%', transform: 'translateY(-50%)' };
      return {};
    };



    return (
        <div
            data-radial-menu="true"
            style={{
              position: 'absolute',
              right: '19px',
              top: 'calc(50% - 2px)',
              transform: 'translateY(-50%)',
              display: 'flex',
              alignItems: 'center',
              zIndex: 1100,
              opacity: (!isHovered && !isOpen) ? 0 : 1,
              pointerEvents: (!isHovered && !isOpen) || (isRadialNavOpen !== null && isRadialNavOpen !== row.id) ? 'none' : 'auto',
              transition: 'opacity 0.15s ease'
            }}
            ref={menuRef}
        >


          <button
              onClick={(e) => {
                e.stopPropagation();
                setIsRadialNavOpen(isOpen ? null : row.id);
              }}
              style={{
                background: 'var(--term-bgAlt)',
                border: '1px solid var(--term-info)',
                borderRadius: '50%',
                cursor: 'pointer',
                width: '24px',
                height: '24px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'var(--term-info)',
                transition: 'all 0.2s ease',
                boxShadow: isOpen ? '0 1px 3px rgba(0, 0, 0, 0.5), 0 0 4px 0 rgba(96, 165, 250, 0.4)' : '0 0 0 0 var(--term-info), 0 0 4px 0 rgba(96, 165, 250, 0.15)',
                fontSize: '16px',
                fontWeight: 'bold',
                position: 'relative',
                zIndex: 1001
              }}

              title="Quick actions"
          >
            ⋮
          </button>

          {isOpen && (() => {
            const grouped: Record<string, any[]> = {};
            menuItems.forEach(item => {
              if (!grouped[item.dir]) grouped[item.dir] = [];
              grouped[item.dir].push(item);
            });



            return (
                <>
                  {menuItems.map((item, idx) => {
                    const indexInDir = grouped[item.dir].indexOf(item);

                    return (
                        <React.Fragment key={`${item.dir}-${item.section || item.action}`}>
                          <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleAction(item);
                              }}
                              style={{
                                position: 'absolute',
                                ...getPosition(item.dir, indexInDir),
                                background: 'var(--term-bgAlt)',
                                border: '1px solid var(--term-info)',
                                borderRadius: '50%',
                                width: '24px',
                                height: '24px',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                color: 'var(--term-info)',
                                fontSize: '10px',
                                fontWeight: 'bold',
                                cursor: 'pointer',
                                transition: 'all 0.3s ease',
                                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.5), 0 0 0 0 rgba(96, 165, 250, 0)',
                                zIndex: 1000
                              }}
                              onMouseEnter={(e) => {
                                e.currentTarget.style.background = 'var(--term-bg)';
                                e.currentTarget.style.color = 'var(--term-fg)';
                                e.currentTarget.style.boxShadow = '0 1px 3px rgba(0, 0, 0, 0.5), 0 0 4px 1px rgba(96, 165, 250, 0.5)';
                              }}
                              onMouseLeave={(e) => {
                                e.currentTarget.style.background = 'var(--term-bgAlt)';
                                e.currentTarget.style.boxShadow = '0 1px 3px rgba(0, 0, 0, 0.5), 0 0 0 0 rgba(96, 165, 250, 0)';
                              }}
                          >
                            {item.label}
                          </button>
                        </React.Fragment>
                    );
                  })}
                </>
            );
          })()}
        </div>
    );
  };

  const renderSection = (title: string, items: TodoRow[], sectionKey: string, canCollapse = true) => {
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
                            className="list-row"
                            draggable
                            onDragStart={(e) => handleDragStart(e, r.id)}
                            onMouseEnter={() => {
                              if (isRadialNavOpen === null) {
                                setHoveredRow(r.id);
                              }
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
                                setDeleteConfirm({ id: r.id, position: { x: e.clientX, y: e.clientY } });
                              }, 1500);
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
                                setDeleteConfirm({ id: r.id, position: { x: touch.clientX, y: touch.clientY } });
                              }, 1500);
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
                              const isRadialMenuClick = target.closest('[data-radial-menu="true"]');
                              const isCheckboxClick = target.closest('.checkbox');

                              if (!isRadialMenuClick && !isCheckboxClick && isRadialNavOpen === null) {
                                e.preventDefault();
                                setHoveredRow(null);
                                onEditTodo(r);
                              }
                            }}
                            style={{
                              cursor: r.completed ? 'grab' : 'pointer',
                              opacity: isRadialNavOpen !== null && isRadialNavOpen !== r.id ? 0.5 : 1,
                              padding: '12px 20px',
                              marginRight: '25px',
                              display: 'flex',
                              gap: '12px',
                              position: 'relative',
                              borderTop: idx === 0 ? '1px solid var(--term-border)' : 'none',
                              borderBottom: idx === items.length - 1 ? 'none' : '1px solid var(--term-border)'
                            }}
                        >
                          <button
                              className={`checkbox ${r.completed ? "checked" : ""}`}
                              onClick={(e) => {
                                e.stopPropagation();
                                onToggle(r.id, !r.completed);
                              }}
                              aria-label={r.completed ? "Uncheck" : "Check"}
                              style={{ cursor: 'pointer', border: 'none', padding: 0, background: 'transparent' }}
                          >
                            <div className={`checkbox ${r.completed ? "checked" : ""}`}>
                              <div className="mark" />
                            </div>
                          </button>
                          <div
                              style={{ minWidth: 0, opacity: r.completed ? 0.5 : 1, flex: 1 }}
                          >
                            <div className={`title ${r.title.includes("http") ? "underlined" : ""}`} style={{ textDecoration: r.completed ? 'line-through' : 'none' }}>
                              {r.title}{" "}
                              <span style={{ color: 'var(--term-info)', opacity: 0.8, fontSize: '0.95em' }}>
                        {r.contexts.map((c: string) => `@${c}`).join(" ")}
                      </span>
                              {r.priority === "A" && <span style={{ color: 'var(--term-success)', marginLeft: '6px' }}>★</span>}
                              {r.priority === "B" && <span style={{ color: 'var(--term-warn)', marginLeft: '6px' }}>⁙</span>}
                              {r.priority === "C" && <span style={{ color: 'var(--term-info)', marginLeft: '6px' }}>•</span>}
                              {!r.completed && r.due_at && <span className="badge info" style={{ marginLeft: '6px', padding: '4px 8px', fontSize: '11px', borderRadius: '6px', opacity: 0.8 }}>{new Date(r.due_at).toLocaleDateString()}</span>}
                            </div>
                            <div className="meta">
                              {r.projects.map((p: string, i: number) => <span key={"p"+p} style={{ color: getProjectColor(i, theme) }}>+{p}</span>)}
                              {r.tags.map((t: string, i: number) => <span key={"t"+t} style={{ color: getTagColor(i, theme) }}>#{t}</span>)}
                            </div>
                          </div>
                          {!r.completed && <DirectionalMenu row={r} />}
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
          height: '450px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center'
        }}>
          <div style={{ textAlign: 'center', color: 'var(--term-dim)', fontSize: '14px' }}>
            No todos yet. Press <kbd style={{
            padding: '2px 6px',
            background: 'var(--term-panel)',
            border: '1px solid var(--term-border)',
            borderRadius: '4px',
            fontSize: '11px',
            fontFamily: 'monospace'
          }}>⌘N</kbd> to create one.
          </div>
        </div>
    );
  }

  const total = rows.length;
  const done = rows.filter(r=>r.completed).length;
  const pending = total - done;
  const pct = Math.round((done/Math.max(total,1))*100);

  if (viewMode === 'date') {
    return (
        <div className="terminal-card" style={{ padding: '20px', position: 'relative', display: 'flex', flexDirection: 'column' }}>
          <CustomScrollbar style={{
            height: '450px'
          }}>
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
                                  if (isRadialNavOpen === null) {
                                    setHoveredRow(r.id);
                                  }
                                }}
                                onMouseLeave={() => setHoveredRow(null)}
                                onClick={(e) => {
                                  const target = e.target as HTMLElement;
                                  const isRadialMenuClick = target.closest('[data-radial-menu="true"]');
                                  const isCheckboxClick = target.closest('.checkbox');

                                  if (!isRadialMenuClick && !isCheckboxClick && isRadialNavOpen === null) {
                                    e.preventDefault();
                                    setHoveredRow(null);
                                    onEditTodo(r);
                                  }
                                }}
                                style={{
                                  cursor: r.completed ? 'default' : 'pointer',
                                  opacity: isRadialNavOpen !== null && isRadialNavOpen !== r.id ? 0.5 : 1,
                                  padding: '12px 20px',
                                  marginRight: '25px',
                                  display: 'flex',
                                  gap: '12px',
                                  position: 'relative',
                                  borderTop: idx === 0 ? '1px solid var(--term-border)' : 'none',
                                  borderBottom: idx === list.length - 1 ? 'none' : '1px solid var(--term-border)'
                                }}
                            >
                              <button
                                  className={`checkbox ${r.completed ? "checked" : ""}`}
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    onToggle(r.id, !r.completed);
                                  }}
                                  aria-label={r.completed ? "Uncheck" : "Check"}
                                  style={{ cursor: 'pointer', border: 'none', padding: 0, background: 'transparent' }}
                              >
                                <div className={`checkbox ${r.completed ? "checked" : ""}`}>
                                  <div className="mark" />
                                </div>
                              </button>
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
                              {!r.completed && <DirectionalMenu row={r} />}
                            </div>
                        ))}
                        {gi < dateGroups.length - 1 && <div className="separator" />}
                      </div>
                  ))
              )}
            </div>
          </CustomScrollbar>

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
            {settingsButton && <div>{settingsButton}</div>}
          </div>
        </div>
    );
  }

  return (
      <>
        {isRadialNavOpen !== null && (
            <div
                ref={modalOverlayRef}
                onClick={() => setIsRadialNavOpen(null)}
                onKeyDown={(e) => {
                  if (e.key === 'Escape') {
                    setIsRadialNavOpen(null);
                  }
                }}
                style={{
                  position: 'fixed',
                  top: 0,
                  left: 0,
                  right: 0,
                  bottom: 0,
                  zIndex: 1099,
                  background: 'transparent',
                  cursor: 'default',
                  outline: 'none'
                }}
                tabIndex={0}
            />
        )}
        <div className="terminal-card" style={{
          padding: '20px',
          position: 'relative',
          display: 'flex',
          flexDirection: 'column'
        }}>
          <CustomScrollbar style={{
            height: '450px'
          }}>
            <div style={{ paddingBottom: '20px' }}>
              {renderSection('Now', sections.now, 'now', true)}
              {renderSection('Soon', sections.soon, 'soon', true)}
              {renderSection('Anytime', sections.anytime, 'anytime', true)}
              {showCompleted && sections.done.length > 0 && renderSection('Done', sections.done, 'done', true)}
            </div>
          </CustomScrollbar>

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
            {settingsButton && <div>{settingsButton}</div>}
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
