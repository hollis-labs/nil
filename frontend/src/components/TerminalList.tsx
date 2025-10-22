import * as React from "react";
import { useTermTheme } from "@/theme/ThemeProvider";

export type TodoRow = {
  id: number; title: string; priority?: string;
  due_at?: string; created_at?: string;
  completed: boolean; archived: boolean;
  projects: string[]; contexts: string[]; tags: string[];
  notes_md?: string;
  section: string; // now, soon, anytime
};

function fmtDateLabel(iso?: string) {
  const d = iso ? new Date(iso) : new Date();
  return d.toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
}
function ymd(iso?: string) {
  const d = iso ? new Date(iso) : new Date();
  return d.toISOString().slice(0,10);
}

type Props = {
  rows: TodoRow[];
  onToggle: (id: number, checked: boolean) => void;
  onOpenNotes: (row: TodoRow) => void;
  onMoveSection: (id: number, section: string) => void;
  showCompleted: boolean;
  onEditTodo: (row: TodoRow) => void;
};

export default function TerminalList({ rows, onToggle, onOpenNotes, onMoveSection, showCompleted, onEditTodo }: Props) {
  const [draggedId, setDraggedId] = React.useState<number | null>(null);

  const sections = React.useMemo(() => {
    if (!rows || !Array.isArray(rows)) return { now: [], soon: [], anytime: [] };
    
    const now: TodoRow[] = [];
    const soon: TodoRow[] = [];
    const anytime: TodoRow[] = [];
    
    for (const r of rows) {
      // Skip completed items unless showCompleted is true
      if (r.completed && !showCompleted) continue;
      
      const section = r.section || 'anytime';
      if (section === 'now') now.push(r);
      else if (section === 'soon') soon.push(r);
      else anytime.push(r);
    }
    
    // Sort by priority within each section
    const sortFn = (a: TodoRow, b: TodoRow) => 
      (a.priority||'Z').localeCompare(b.priority||'Z') || 
      (b.created_at||'').localeCompare(a.created_at||'');
    
    now.sort(sortFn);
    soon.sort(sortFn);
    anytime.sort(sortFn);
    
    return { now, soon, anytime };
  }, [rows, showCompleted]);

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

  const renderSection = (title: string, items: TodoRow[], sectionKey: string) => (
    <div
      onDragOver={handleDragOver}
      onDrop={(e) => handleDrop(e, sectionKey)}
      style={{ marginBottom: '20px' }}
    >
      <div className="date-header" style={{ textDecoration: 'underline', textUnderlineOffset: '4px' }}>
        {title}
      </div>
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
                {r.projects.map((p: string) => <span key={"p"+p} className="text-dim">+{p}</span>)}
                {r.tags.map((t: string) => <span key={"t"+t} className="text-dim">#{t}</span>)}
              </div>
            </div>
          </div>
        ))
      )}
    </div>
  );

  if (!rows || rows.length === 0) {
    return (
      <div className="terminal-card" style={{ padding: '20px' }}>
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

  return (
    <div className="terminal-card" style={{ padding: '20px', position: 'relative', display: 'flex', flexDirection: 'column' }}>
      <div style={{ 
        maxHeight: '450px', 
        minHeight: '200px',
        overflowY: 'auto',
        paddingBottom: '60px',
        marginBottom: '-40px'
      }} className="todo-list-scroll">
        {renderSection('Now', sections.now, 'now')}
        {renderSection('Soon', sections.soon, 'soon')}
        {renderSection('Anytime', sections.anytime, 'anytime')}
      </div>
      
      <div className="summary" style={{ 
        position: 'sticky',
        bottom: 0,
        background: 'var(--term-panel)',
        padding: '16px 0 0',
        zIndex: 10,
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
