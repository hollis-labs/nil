import * as React from "react";
import TerminalList, { TodoRow } from "@/components/TerminalList";
import KeyboardScope from "@/components/KeyboardScope";
import NotesModal from "@/components/NotesModal";
import EditTodoModal from "@/components/EditTodoModal";
import SettingsModal, { useSettings } from "@/components/SettingsModal";
import { CopyrightFooter } from "@/components/CopyrightFooter";
import { ThemeProvider } from "@/theme/ThemeProvider";
import { SlidersVertical, Plus } from "lucide-react";

import * as Backend from "../../wailsjs/go/main/App";

function Inner() {
  const [allRows, setAllRows] = React.useState<TodoRow[]>([]);
  const [query, setQuery] = React.useState("");
  const [notesOpen, setNotesOpen] = React.useState(false);
  const [notesTodo, setNotesTodo] = React.useState<any>(null);
  const [quickOpen, setQuickOpen] = React.useState(false);
  const [settingsOpen, setSettingsOpen] = React.useState(false);
  const [activeTabId, setActiveTabId] = React.useState<string>('1');
  const [editTodo, setEditTodo] = React.useState<TodoRow | null>(null);
  const { settings } = useSettings();

  const runSearch = React.useCallback(async () => {
    try {
      const req: any = { 
        query: query || "", 
        page: 0, 
        page_size: 500, 
        sort_by: "created_at", 
        sort_dir: "desc",
        projects: [],
        contexts: [],
        tags: [],
        statuses: [],
        priorities: []
      };
      const res = await Backend.Search(req);
      console.log("Search result:", res);
      setAllRows(Array.isArray(res) ? res : []);
    } catch (err) {
      console.error("Search failed:", err);
      setAllRows([]);
    }
  }, [query]);

  React.useEffect(()=>{ runSearch(); }, [runSearch]);

  // Filter rows based on active tab and settings
  const rows = React.useMemo(() => {
    let filtered = allRows;
    
    // Apply scope tab filter
    const activeTab = settings.tabs.find(t => t.id === activeTabId);
    if (activeTab) {
      if (activeTab.context) {
        filtered = filtered.filter(r => r.contexts.includes(activeTab.context!));
      }
      if (activeTab.project) {
        filtered = filtered.filter(r => r.projects.includes(activeTab.project!));
      }
    }
    
    // Apply show completed filter
    if (!settings.showCompleted) {
      filtered = filtered.filter(r => !r.completed);
    }
    
    return filtered;
  }, [allRows, activeTabId, settings]);

  async function handleToggle(id: number, checked: boolean) {
    await Backend.ToggleComplete(id, checked);
    runSearch();
  }
  
  async function handleMoveSection(id: number, section: string) {
    const todo = allRows.find(r => r.id === id);
    if (!todo) return;
    await Backend.UpdateTodo({ ...todo, section } as any);
    runSearch();
  }
  
  function handleOpenNotes(row: TodoRow) { setNotesTodo(row); setNotesOpen(true); }
  async function handleSaveNotes(md: string) {
    if (!notesTodo) return;
    await Backend.UpdateTodo({ ...notesTodo, notes_md: md } as any);
    setNotesOpen(false); setNotesTodo(null); runSearch();
  }
  async function handleQuickAdd(line: string, extras: any) {
    const created = await Backend.CreateTodoFromLine(line);
    const merged = {
      ...created,
      priority: extras.priority || created.priority,
      due_at: extras.due || created.due_at,
      threshold_at: extras.t || created.threshold_at,
      projects: extras.projects?.length ? extras.projects : created.projects,
      contexts: extras.contexts?.length ? extras.contexts : created.contexts,
      tags: extras.tags?.length ? extras.tags : created.tags,
      notes_md: extras.notes_md || created.notes_md,
    };
    await Backend.UpdateTodo(merged as any);
    setQuickOpen(false);
    runSearch();
  }
  
  function handleEditTodo(row: TodoRow) {
    setEditTodo(row);
    setQuickOpen(true);
  }
  
  async function handleUpdateTodo(todo: TodoRow) {
    const reparsed = await Backend.CreateTodoFromLine(todo.title);
    const merged = {
      ...todo,
      title: reparsed.title,
      projects: reparsed.projects?.length ? reparsed.projects : todo.projects,
      contexts: reparsed.contexts?.length ? reparsed.contexts : todo.contexts,
      tags: [...new Set([...(todo.tags || []), ...(reparsed.tags || [])])],
    };
    await Backend.UpdateTodo(merged as any);
    setQuickOpen(false);
    setEditTodo(null);
    runSearch();
  }
  
  async function handleDeleteTodo(id: number) {
    await Backend.DeleteTodo(id);
    setQuickOpen(false);
    setEditTodo(null);
    runSearch();
  }

  return (
    <div style={{ minHeight: '100vh', padding: '24px', display: 'flex', flexDirection: 'column', justifyContent: 'flex-start', alignItems: 'center' }}>
      <KeyboardScope onQuickAdd={()=>setQuickOpen(true)} />
      
      <div style={{ width: '100%', maxWidth: '800px' }}>
        {/* PLANCK Branding */}
        <div style={{ 
          marginBottom: '12px',
          display: 'flex',
          alignItems: 'baseline',
          gap: '6px'
        }}>
          <div style={{ 
            fontFamily: '"Courier New", Courier, monospace',
            fontSize: '14px',
            fontWeight: 800,
            letterSpacing: '0.12em',
            color: 'var(--term-accent)',
          }}>
            PLANCK
          </div>
          <div style={{ 
            fontSize: '10px',
            color: 'var(--term-dim)',
            fontFamily: 'serif',
            fontStyle: 'italic',
            opacity: 0.6,
            letterSpacing: '0.02em'
          }}>
            <span style={{ fontStyle: 'italic' }}>h</span> — the quantum of action
          </div>
        </div>
        {/* Search Bar */}
        <div style={{ marginBottom: '16px', display: 'flex', gap: '8px', alignItems: 'center' }}>
          <input
            style={{
              flex: 1,
              padding: '8px 12px',
              background: 'var(--term-panel)',
              border: '1px solid var(--term-border)',
              borderRadius: '6px',
              color: 'var(--term-fg)',
              fontSize: '13px'
            }}
            placeholder="Search… e.g. review #work +todoapp @home pri:A"
            value={query}
            onChange={(e)=>setQuery(e.target.value)}
            onKeyDown={(e)=>{ if(e.key==="Enter") runSearch(); }}
          />
          <button 
            style={{
              padding: '8px',
              background: 'var(--term-panel)',
              border: '1px solid var(--term-border)',
              borderRadius: '6px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'all 0.15s ease',
              height: '32px',
              width: '32px'
            }}
            onClick={()=>setQuickOpen(true)}
            title="New Todo (⌘N)"
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--term-accent)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--term-border)';
            }}
          >
            <Plus size={16} color="var(--term-fg)" />
          </button>
          <button 
            style={{
              padding: '8px',
              background: 'var(--term-panel)',
              border: '1px solid var(--term-border)',
              borderRadius: '6px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'all 0.15s ease',
              height: '32px',
              width: '32px'
            }}
            onClick={()=>setSettingsOpen(true)} 
            title="Settings"
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--term-accent)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--term-border)';
            }}
          >
            <SlidersVertical size={16} color="var(--term-fg)" />
          </button>
        </div>

        {/* Scope Tabs */}
        {settings.tabs.length > 0 && (
          <div style={{ marginBottom: '16px', display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
            {settings.tabs.map(tab => (
              <button
                key={tab.id}
                className={`badge ${activeTabId === tab.id ? 'success' : ''}`}
                onClick={() => setActiveTabId(tab.id)}
                style={{ padding: '6px 12px', fontSize: '12px' }}
              >
                {tab.label}
              </button>
            ))}
          </div>
        )}

        <TerminalList rows={rows} onToggle={handleToggle} onOpenNotes={handleOpenNotes} onMoveSection={handleMoveSection} showCompleted={settings.showCompleted} onEditTodo={handleEditTodo} />
        
        <CopyrightFooter version="1.0.0" buildDate={new Date().toISOString().slice(0, 10)} />
      </div>

      <SettingsModal open={settingsOpen} onOpenChange={setSettingsOpen} />
      <NotesModal open={notesOpen} onOpenChange={setNotesOpen} todo={notesTodo} onSave={handleSaveNotes} />
      <EditTodoModal 
        open={quickOpen} 
        onOpenChange={(v) => { setQuickOpen(v); if (!v) setEditTodo(null); }} 
        onSubmit={handleQuickAdd}
        onUpdate={handleUpdateTodo}
        onDelete={handleDeleteTodo}
        editTodo={editTodo}
      />
    </div>
  );
}

export default function AppPage() {
  return (
    <ThemeProvider>
      <link rel="stylesheet" href="/src/theme/theme.css" />
      <Inner />
    </ThemeProvider>
  );
}
