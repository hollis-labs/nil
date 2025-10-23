import * as React from "react";
import { useTermTheme } from "@/theme/ThemeProvider";
import { TermTheme } from "@/theme/theme";
import CustomScrollbar from "@/components/CustomScrollbar";

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
  showCompleted: boolean;
  onEditTodo: (row: TodoRow) => void;
  viewMode?: ViewMode;
};

export default function TerminalList({ rows, onToggle, onOpenNotes, onMoveSection, showCompleted, onEditTodo, viewMode = 'scope' }: Props) {
  const { theme } = useTermTheme();
  const [draggedId, setDraggedId] = React.useState<number | null>(null);
  const [collapsedSections, setCollapsedSections] = React.useState<Record<string, boolean>>({
    done: true
  });

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
            textDecoration: 'underline', 
            textUnderlineOffset: '4px',
            cursor: canCollapse ? 'pointer' : 'default',
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            userSelect: 'none'
          }}
          onClick={() => canCollapse && toggleSection(sectionKey)}
        >
          {canCollapse && <span>{isCollapsed ? '▸' : '▾'}</span>}
          <span>{title}</span>
          <span style={{ fontSize: '0.75em', opacity: 0.6 }}>({items.length})</span>
        </div>
        {!isCollapsed && (
          <>
            <div className="separator" />
            {items.length === 0 ? (
              <div style={{ padding: '12px', textAlign: 'center', color: 'var(--term-dim)', fontSize: '12px' }}>
                Drop items here
              </div>
            ) : (
              items.map((r) => (
                <div
                  key={r.id}
                  className="list-row"
                  draggable
                  onDragStart={(e) => handleDragStart(e, r.id)}
                  style={{ cursor: 'grab' }}
                >
                  <button
                    className={`checkbox ${r.completed ? "checked" : ""}`}
                    onClick={() => onToggle(r.id, !r.completed)}
                    aria-label={r.completed ? "Uncheck" : "Check"}
                    style={{ cursor: 'pointer', border: 'none', padding: 0, background: 'transparent' }}
                  >
                    <div className={`checkbox ${r.completed ? "checked" : ""}`}>
                      <div className="mark" />
                    </div>
                  </button>
                  <div 
                    style={{ minWidth: 0, opacity: r.completed ? 0.5 : 1, cursor: 'pointer', flex: 1 }}
                    onClick={() => onEditTodo(r)}
                  >
                    <div className={`title ${r.title.includes("http") ? "underlined" : ""}`} style={{ textDecoration: r.completed ? 'line-through' : 'none' }}>
                      {r.title}{" "}
                      <span style={{ color: 'var(--term-info)', opacity: 0.8, fontSize: '0.95em' }}>
                        {r.contexts.map((c: string) => `@${c}`).join(" ")}
                      </span>{" "}
                      {r.priority === "A" && <span className="badge warn">★</span>}
                      {r.priority === "B" && <span className="badge info">•</span>}
                      {!r.completed && r.due_at && <span className="badge info">{new Date(r.due_at).toLocaleDateString()}</span>}
                    </div>
                    <div className="meta">
                      {r.projects.map((p: string, i: number) => <span key={"p"+p} style={{ color: getProjectColor(i, theme) }}>+{p}</span>)}
                      {r.tags.map((t: string, i: number) => <span key={"t"+t} style={{ color: getTagColor(i, theme) }}>#{t}</span>)}
                    </div>
                  </div>
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
      <div className="terminal-card" style={{ padding: '20px', height: '450px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
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
                <div className="date-header" style={{ textDecoration: 'underline', textUnderlineOffset: '4px' }}>
                  {fmtDateLabel(key)}
                </div>
                <div className="separator" />
                {list.map((r) => (
                  <div
                    key={r.id}
                    className="list-row"
                    style={{ cursor: 'pointer' }}
                  >
                    <button
                      className={`checkbox ${r.completed ? "checked" : ""}`}
                      onClick={() => onToggle(r.id, !r.completed)}
                      aria-label={r.completed ? "Uncheck" : "Check"}
                      style={{ cursor: 'pointer', border: 'none', padding: 0, background: 'transparent' }}
                    >
                      <div className={`checkbox ${r.completed ? "checked" : ""}`}>
                        <div className="mark" />
                      </div>
                    </button>
                    <div 
                      style={{ minWidth: 0, opacity: r.completed ? 0.5 : 1, cursor: 'pointer', flex: 1 }}
                      onClick={() => onEditTodo(r)}
                    >
                      <div className={`title ${r.title.includes("http") ? "underlined" : ""}`}>
                        {r.title}{" "}
                        <span style={{ color: 'var(--term-info)', opacity: 0.8, fontSize: '0.95em' }}>
                          {r.contexts.map((c: string) => `@${c}`).join(" ")}
                        </span>{" "}
                        {r.priority === "A" && <span className="badge warn">★</span>}
                        {r.priority === "B" && <span className="badge info">•</span>}
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
        
        <div className="summary" style={{ 
          background: 'var(--term-panel)',
          padding: '16px 0 0',
          borderTop: '1px solid var(--term-border)',
          marginTop: '20px'
        }}>
          <span>
            {pct}% complete · <span className="badge success">{done} done</span> · <span className="badge info">{pending} pending</span>
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className="terminal-card" style={{ padding: '20px', position: 'relative', display: 'flex', flexDirection: 'column' }}>
      <CustomScrollbar style={{ 
        height: '450px'
      }}>
        <div style={{ paddingBottom: '20px' }}>
        {renderSection('Now', sections.now, 'now', true)}
        {renderSection('Soon', sections.soon, 'soon', true)}
        {renderSection('Anytime', sections.anytime, 'anytime', true)}
        {sections.done.length > 0 && renderSection('Done', sections.done, 'done', true)}
        </div>
      </CustomScrollbar>
      
      <div className="summary" style={{ 
        background: 'var(--term-panel)',
        padding: '16px 0 0',
        borderTop: '1px solid var(--term-border)',
        marginTop: '20px'
      }}>
        <span>
          {pct}% complete · <span className="badge success">{done} done</span> · <span className="badge info">{pending} pending</span>
        </span>
      </div>
    </div>
  );
}
