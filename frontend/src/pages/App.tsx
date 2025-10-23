import * as React from "react";
import TerminalList, { TodoRow } from "@/components/TerminalList";
import KeyboardScope from "@/components/KeyboardScope";
import NotesModal from "@/components/NotesModal";
import EditTodoModal from "@/components/EditTodoModal";
import SettingsModal, { useSettings, SettingsProvider } from "@/components/SettingsModal";
import { CopyrightFooter } from "@/components/CopyrightFooter";
import CustomScrollbar from "@/components/CustomScrollbar";
import { ThemeProvider } from "@/theme/ThemeProvider";
import { SlidersVertical, Plus, Calendar, List } from "lucide-react";
import { parseQuery } from "@/lib/query";

import * as Backend from "../../wailsjs/go/main/App";

type ViewMode = 'scope' | 'date';

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
  const [viewMode, setViewMode] = React.useState<ViewMode>(() => {
    const saved = localStorage.getItem('planck.viewMode');
    return (saved as ViewMode) || settings.defaultView || 'scope';
  });
  
  React.useEffect(() => {
    localStorage.setItem('planck.viewMode', viewMode);
  }, [viewMode]);
  
  const defaultNewTodoFilters = React.useMemo(() => {
    const activeTab = settings.tabs.find(t => t.id === activeTabId);
    const mainParsed = parseQuery(query);
    const tabParsed = activeTab?.query ? parseQuery(activeTab.query) : null;
    
    return {
      contexts: [...mainParsed.contexts, ...(tabParsed?.contexts || [])],
      projects: [...mainParsed.projects, ...(tabParsed?.projects || [])],
      tags: [...mainParsed.tags, ...(tabParsed?.tags || [])]
    };
  }, [query, activeTabId, settings.tabs]);

  const runSearch = React.useCallback(async () => {
    try {
      const activeTab = settings.tabs.find(t => t.id === activeTabId);
      const mainParsed = parseQuery(query);
      const tabParsed = activeTab?.query ? parseQuery(activeTab.query) : null;

      const merged = {
        keywords: [...mainParsed.keywords],
        projects: [...mainParsed.projects, ...(tabParsed?.projects || [])],
        contexts: [...mainParsed.contexts, ...(tabParsed?.contexts || [])],
        tags: [...mainParsed.tags, ...(tabParsed?.tags || [])],
        priorities: [...(mainParsed.priority || []), ...(tabParsed?.priority || [])],
        statuses: [...(mainParsed.flags.status || []), ...(tabParsed?.flags.status || [])]
      };

      // Use explicit status filters if provided, otherwise let frontend handle filtering
      // Backend combines multiple statuses with AND (not OR), so we can only use single statuses
      const statusesToSearch = merged.statuses.length > 0
        ? merged.statuses
        : ['open']; // Always fetch at least open items

      const req: any = {
        query: merged.keywords.join(" "),
        page: 0,
        page_size: 500,
        sort_by: "created_at",
        sort_dir: "desc",
        projects: merged.projects,
        contexts: merged.contexts,
        tags: merged.tags,
        statuses: statusesToSearch,
        priorities: merged.priorities
      };
      const res = await Backend.Search(req);
      let allResults = Array.isArray(res) ? res : [];

      // If showCompleted is enabled and no explicit status filter, also fetch completed items
      if (settings.showCompleted && merged.statuses.length === 0) {
        const completedReq = { ...req, statuses: ['completed'] };
        const completedRes = await Backend.Search(completedReq);
        const completedResults = Array.isArray(completedRes) ? completedRes : [];
        // Merge and deduplicate by id
        const idsInResults = new Set(allResults.map(r => r.id));
        const newCompleted = completedResults.filter(r => !idsInResults.has(r.id));
        allResults = [...allResults, ...newCompleted];
      }

      setAllRows(allResults);
    } catch (err) {
      console.error("Search failed:", err);
      setAllRows([]);
    }
  }, [query, activeTabId, settings.tabs, settings.showCompleted]);

  React.useEffect(()=>{ runSearch(); }, [runSearch]);

  // Filter out archived items (unless explicitly searched for)
  // TerminalList handles showCompleted filtering internally
  const rows = React.useMemo(() => {
    // Don't filter if user explicitly searched for archived items
    const activeTab = settings.tabs.find(t => t.id === activeTabId);
    const mainParsed = parseQuery(query);
    const tabParsed = activeTab?.query ? parseQuery(activeTab.query) : null;
    const statuses = [...(mainParsed.flags.status || []), ...(tabParsed?.flags.status || [])];

    if (statuses.includes('archived')) {
      return allRows; // User wants archived items, show them
    }

    return allRows.filter(r => !r.archived);
  }, [allRows, query, activeTabId, settings.tabs]);

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
  
  async function handleArchive(id: number, archived: boolean) {
    await Backend.Archive(id, archived);
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
    <div style={{ height: '100vh', overflow: 'hidden', padding: '24px', display: 'flex', flexDirection: 'column', justifyContent: 'flex-start', alignItems: 'center', overscrollBehavior: 'none' }}>
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

        {/* Scope Tabs & View Mode Toggle */}
        <div style={{ marginBottom: '16px', display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
          {settings.tabs.length > 0 && settings.tabs.map(tab => (
            <button
              key={tab.id}
              className={`badge ${activeTabId === tab.id ? 'success' : ''}`}
              onClick={() => setActiveTabId(tab.id)}
              style={{ padding: '6px 12px', fontSize: '12px' }}
            >
              {tab.label}
            </button>
          ))}
          
          <div style={{ marginLeft: 'auto', display: 'flex', gap: '6px' }}>
            <button
              className={`badge ${viewMode === 'scope' ? 'info' : ''}`}
              onClick={() => setViewMode('scope')}
              style={{ padding: '6px 12px', fontSize: '12px', display: 'flex', alignItems: 'center', gap: '4px' }}
              title="Scope View (Now/Soon/Anytime)"
            >
              <List size={12} />
              Scope
            </button>
            <button
              className={`badge ${viewMode === 'date' ? 'info' : ''}`}
              onClick={() => setViewMode('date')}
              style={{ padding: '6px 12px', fontSize: '12px', display: 'flex', alignItems: 'center', gap: '4px' }}
              title="Date View (Due Dates)"
            >
              <Calendar size={12} />
              Date
            </button>
          </div>
        </div>

        <TerminalList rows={rows} onToggle={handleToggle} onOpenNotes={handleOpenNotes} onMoveSection={handleMoveSection} onArchive={handleArchive} onDelete={handleDeleteTodo} showCompleted={settings.showCompleted} onEditTodo={handleEditTodo} viewMode={viewMode} />
        
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
        defaultContexts={defaultNewTodoFilters.contexts}
        defaultProjects={defaultNewTodoFilters.projects}
        defaultTags={defaultNewTodoFilters.tags}
      />
    </div>
  );
}

export default function AppPage() {
  return (
    <SettingsProvider>
      <ThemeProvider>
        <Inner />
      </ThemeProvider>
    </SettingsProvider>
  );
}
