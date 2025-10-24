import * as React from "react";
import TerminalList, { TodoRow } from "@/components/TerminalList";
import KeyboardScope from "@/components/KeyboardScope";
import NotesModal from "@/components/NotesModal";
import EditTodoModal from "@/components/EditTodoModal";
import SettingsModal, { useSettings, SettingsProvider } from "@/components/SettingsModal";
import SessionContextModal from "@/components/SessionContextModal";
import SearchAutocomplete from "@/components/SearchAutocomplete";
import WelcomeDialog from "@/components/WelcomeDialog";
import HelpModal from "@/components/HelpModal";
import AlphaWarning from "@/components/AlphaWarning";
import { CopyrightFooter } from "@/components/CopyrightFooter";
import CustomScrollbar from "@/components/CustomScrollbar";
import { ThemeProvider } from "@/theme/ThemeProvider";
import { Settings, Plus, Calendar, List, Target, Power, HelpCircle } from "lucide-react";
import { parseQuery } from "@/lib/query";
import { getActiveSession, setActiveSession } from "@/lib/sessionContext";

import * as Backend from "../../wailsjs/go/main/App";
import { Quit } from "../../wailsjs/runtime/runtime";

type ViewMode = 'scope' | 'date';

function Inner() {
  const [allRows, setAllRows] = React.useState<TodoRow[]>([]);
  const [query, setQuery] = React.useState("");
  const [notesOpen, setNotesOpen] = React.useState(false);
  const [notesTodo, setNotesTodo] = React.useState<any>(null);
  const [quickOpen, setQuickOpen] = React.useState(false);
  const [settingsOpen, setSettingsOpen] = React.useState(false);
  const [sessionContextOpen, setSessionContextOpen] = React.useState(false);
  const [activeTabId, setActiveTabId] = React.useState<string>('1');
  const [editTodo, setEditTodo] = React.useState<TodoRow | null>(null);
  const [sessionFilterCount, setSessionFilterCount] = React.useState(0);
  const [sessionAsFilter, setSessionAsFilter] = React.useState(false);
  const [showWelcome, setShowWelcome] = React.useState(false);
  const [showDemoPrompt, setShowDemoPrompt] = React.useState(false);
  const [hasDemoData, setHasDemoData] = React.useState(false);
  const [helpOpen, setHelpOpen] = React.useState(false);
  const [showAlphaWarning, setShowAlphaWarning] = React.useState(false);
  const { settings } = useSettings();
  const [viewMode, setViewMode] = React.useState<ViewMode>(() => {
    const saved = localStorage.getItem('planck.viewMode');
    return (saved as ViewMode) || settings.defaultView || 'scope';
  });

  React.useEffect(() => {
    localStorage.setItem('planck.viewMode', viewMode);
  }, [viewMode]);

  // Update session filter count when component mounts or session changes
  const updateSessionFilterCount = React.useCallback(() => {
    const session = getActiveSession();
    if (!session) {
      setSessionFilterCount(0);
      setSessionAsFilter(false);
      return;
    }
    const count =
      (session.contexts?.length || 0) +
      (session.projects?.length || 0) +
      (session.tags?.length || 0) +
      (session.priority ? 1 : 0);
    setSessionFilterCount(count);
    setSessionAsFilter(session.useAsFilterTab || false);
  }, []);

  React.useEffect(() => {
    updateSessionFilterCount();
  }, [updateSessionFilterCount]);

  React.useEffect(() => {
    // Check if database is set up
    Backend.NeedsSetup().then((needs: boolean) => {
      console.log('NeedsSetup:', needs);
      if (needs) {
        // Fresh install - always show alpha warning first
        console.log('Showing alpha warning');
        setShowAlphaWarning(true);
      } else {
        // Database exists, check demo data
        console.log('Database exists, checking demo data');
        (Backend as any).HasDemoData?.().then((has: boolean) => {
          setHasDemoData(has);
          if (!has) {
            setShowDemoPrompt(true);
          }
        });
      }
    }).catch(err => {
      console.error('Error checking setup:', err);
      // On error, assume fresh install
      setShowAlphaWarning(true);
    });
  }, []);

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
      const session = getActiveSession();
      const sessionAsFilter = session?.useAsFilterTab ? session : null;

      const activeTab = sessionAsFilter ? null : settings.tabs.find(t => t.id === activeTabId);
      const mainParsed = parseQuery(query);
      const tabParsed = activeTab?.query ? parseQuery(activeTab.query) : null;

      const merged = {
        keywords: [...mainParsed.keywords],
        projects: [
          ...mainParsed.projects, 
          ...(tabParsed?.projects || []),
          ...(sessionAsFilter?.projects || [])
        ],
        contexts: [
          ...mainParsed.contexts, 
          ...(tabParsed?.contexts || []),
          ...(sessionAsFilter?.contexts || [])
        ],
        tags: [
          ...mainParsed.tags, 
          ...(tabParsed?.tags || []),
          ...(sessionAsFilter?.tags || [])
        ],
        priorities: [
          ...(mainParsed.priority || []), 
          ...(tabParsed?.priority || []),
          ...(sessionAsFilter?.priority ? [sessionAsFilter.priority] : [])
        ],
        statuses: [...(mainParsed.flags.status || []), ...(tabParsed?.flags.status || [])],
        negativeKeywords: [...mainParsed.negativeKeywords, ...(tabParsed?.negativeKeywords || [])],
        negativeProjects: [...mainParsed.negativeProjects, ...(tabParsed?.negativeProjects || [])],
        negativeContexts: [...mainParsed.negativeContexts, ...(tabParsed?.negativeContexts || [])],
        negativeTags: [...mainParsed.negativeTags, ...(tabParsed?.negativeTags || [])]
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

      // Client-side filtering for negative keywords
      if (merged.negativeKeywords.length > 0 || merged.negativeProjects.length > 0 || 
          merged.negativeContexts.length > 0 || merged.negativeTags.length > 0) {
        allResults = allResults.filter(todo => {
          // Check negative keywords
          for (const kw of merged.negativeKeywords) {
            const kwLower = kw.toLowerCase();
            if (todo.title?.toLowerCase().includes(kwLower) || 
                todo.notes_md?.toLowerCase().includes(kwLower)) {
              return false;
            }
          }
          // Check negative projects
          if (merged.negativeProjects.length > 0 && todo.projects) {
            const todoProjects = todo.projects.map((p: any) => p.toLowerCase());
            if (merged.negativeProjects.some(np => todoProjects.includes(np.toLowerCase()))) {
              return false;
            }
          }
          // Check negative contexts
          if (merged.negativeContexts.length > 0 && todo.contexts) {
            const todoContexts = todo.contexts.map((c: any) => c.toLowerCase());
            if (merged.negativeContexts.some(nc => todoContexts.includes(nc.toLowerCase()))) {
              return false;
            }
          }
          // Check negative tags
          if (merged.negativeTags.length > 0 && todo.tags) {
            const todoTags = todo.tags.map((t: any) => t.toLowerCase());
            if (merged.negativeTags.some(nt => todoTags.includes(nt.toLowerCase()))) {
              return false;
            }
          }
          return true;
        });
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

  function handleClearAll() {
    if (!quickOpen && !notesOpen && !settingsOpen && !sessionContextOpen) {
      setQuery("");
      const session = getActiveSession();
      if (session?.useAsFilterTab) {
        setActiveSession({ ...session, useAsFilterTab: false });
        updateSessionFilterCount();
        runSearch();
      }
    }
  }

  return (
    <div style={{ 
      width: '100vw',
      height: '100vh', 
      overflow: 'hidden', 
      display: 'flex', 
      flexDirection: 'column',
      padding: '10px',
      boxSizing: 'border-box'
    }}>
      <KeyboardScope onQuickAdd={()=>setQuickOpen(true)} onEscape={handleClearAll} />

      <div style={{ 
        width: '100%',
        height: '100%',
        background: 'var(--term-bg)',
        borderRadius: '12px',
        boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4), 0 2px 8px rgba(0, 0, 0, 0.2)',
        border: '1px solid rgba(255, 255, 255, 0.1)',
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column'
      }}>
        {/* PLANCK Branding - Draggable */}
        <div 
          style={{
            padding: '16px 20px',
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            cursor: 'grab',
            userSelect: 'none',
            borderBottom: '1px solid var(--term-border)',
            // @ts-ignore
            '--wails-draggable': 'drag',
            WebkitAppRegion: 'drag'
          } as any}
        >
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
            <span style={{ fontStyle: 'italic' }}>(h)</span> — the quantum of action
          </div>
          
          {hasDemoData && (
            <button
              className="badge warn"
              onClick={async (e) => {
                e.stopPropagation();
                e.preventDefault();
                if (confirm('Remove all tutorial todos?')) {
                  try {
                    await (Backend as any).RemoveDemoData?.();
                    setHasDemoData(false);
                    runSearch();
                  } catch (err) {
                    console.error('Failed to remove demo data:', err);
                    alert('Failed to remove demo data');
                  }
                }
              }}
              style={{
                marginLeft: 'auto',
                fontSize: '11px',
                padding: '4px 8px',
                cursor: 'pointer',
                // @ts-ignore
                WebkitAppRegion: 'no-drag'
              } as any}
            >
              Remove Tutorial
            </button>
          )}
        </div>
        
        <div style={{ 
          flex: 1, 
          overflow: 'auto', 
          padding: '20px',
          // @ts-ignore
          WebkitAppRegion: 'no-drag'
        } as any}>
        {/* Search Bar */}
        <div style={{ marginBottom: '16px', display: 'flex', gap: '8px', alignItems: 'center' }}>
          <SearchAutocomplete
            value={query}
            onChange={setQuery}
            onSearch={runSearch}
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
              background: sessionAsFilter ? 'var(--term-accent)' : 'var(--term-panel)',
              border: sessionAsFilter ? '1px solid var(--term-accent)' : '1px solid var(--term-border)',
              borderRadius: '6px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'all 0.15s ease',
              height: '32px',
              width: '32px',
              position: 'relative'
            }}
            onClick={()=>setSessionContextOpen(true)}
            title={sessionAsFilter ? "Session Context (Active Filter)" : "Session Context"}
            onMouseEnter={(e) => {
              if (!sessionAsFilter) {
                e.currentTarget.style.borderColor = 'var(--term-accent)';
              }
            }}
            onMouseLeave={(e) => {
              if (!sessionAsFilter) {
                e.currentTarget.style.borderColor = 'var(--term-border)';
              }
            }}
          >
            <Target size={16} color={sessionAsFilter ? '#000' : 'var(--term-fg)'} />
            {sessionFilterCount > 0 && (
              <span style={{
                position: 'absolute',
                top: '-4px',
                right: '-4px',
                background: 'var(--term-accent)',
                color: '#000',
                borderRadius: '50%',
                width: '16px',
                height: '16px',
                fontSize: '10px',
                fontWeight: 'bold',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                border: '1px solid var(--term-bg)'
              }}>
                {sessionFilterCount}
              </span>
            )}
          </button>
        </div>

        {/* Scope Tabs & View Mode Toggle */}
        <div style={{ marginBottom: '16px', display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
          {sessionAsFilter && (
            <button
              className="badge warn"
              style={{ padding: '6px 12px', fontSize: '12px', display: 'flex', alignItems: 'center', gap: '6px', fontWeight: 'bold' }}
              title="Session Context Filter Active"
            >
              <Target size={12} />
              Session Filter
            </button>
          )}
          {settings.tabs.length > 0 && settings.tabs.map(tab => (
            <button
              key={tab.id}
              className={`badge ${!sessionAsFilter && activeTabId === tab.id ? 'success' : ''}`}
              onClick={() => !sessionAsFilter && setActiveTabId(tab.id)}
              style={{ 
                padding: '6px 12px', 
                fontSize: '12px',
                opacity: sessionAsFilter ? 0.4 : 1,
                cursor: sessionAsFilter ? 'not-allowed' : 'pointer'
              }}
              disabled={sessionAsFilter}
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

        <TerminalList
          rows={rows}
          onToggle={handleToggle}
          onOpenNotes={handleOpenNotes}
          onMoveSection={handleMoveSection}
          onArchive={handleArchive}
          onDelete={handleDeleteTodo}
          showCompleted={settings.showCompleted}
          onEditTodo={handleEditTodo}
          viewMode={viewMode}
          closeRadialMenus={quickOpen || notesOpen || settingsOpen || sessionContextOpen || editTodo !== null}
          settingsButton={
            <div style={{ display: 'flex', gap: '8px' }}>
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
                  width: '32px',
                  position: 'relative',
                  top: '6px'
                }}
                onClick={()=>setHelpOpen(true)}
                title="Help & Guide"
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = 'var(--term-accent)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = 'var(--term-border)';
                }}
              >
                <HelpCircle size={16} color="var(--term-fg)" />
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
                  width: '32px',
                  position: 'relative',
                  top: '6px'
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
                <Settings size={16} color="var(--term-fg)" />
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
                  width: '32px',
                  position: 'relative',
                  top: '6px'
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  if (confirm('Quit PLANCK?')) {
                    Quit();
                  }
                }}
                title="Quit PLANCK"
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = 'var(--term-accent)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = 'var(--term-border)';
                }}
              >
                <Power size={16} color="var(--term-fg)" />
              </button>
            </div>
          }
        />

          <CopyrightFooter version="1.0.0" buildDate={new Date().toISOString().slice(0, 10)} />
        </div>
      </div>

      <AlphaWarning
        open={showAlphaWarning}
        onComplete={() => {
          console.log('Alpha warning completed');
          setShowAlphaWarning(false);
          // After warning, show welcome dialog for database setup
          setShowWelcome(true);
        }}
      />
      
      {!showAlphaWarning && (
        <WelcomeDialog 
          open={showWelcome} 
          onComplete={() => { 
            setShowWelcome(false);
            setShowDemoPrompt(true);
          }} 
        />
      )}
      
      {/* Demo Data Prompt */}
      {showDemoPrompt && (
        <div style={{
          position: 'fixed',
          inset: 0,
          background: 'rgba(0, 0, 0, 0.85)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 10000
        }}>
          <div style={{
            background: 'var(--term-bg)',
            border: '1px solid var(--term-border)',
            borderRadius: '12px',
            padding: '32px',
            maxWidth: '500px',
            boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)'
          }}>
            <h2 style={{ margin: '0 0 16px 0', color: 'var(--term-accent)', fontSize: '20px' }}>
              Welcome to Planck!
            </h2>
            <p style={{ margin: '0 0 24px 0', color: 'var(--term-fg)', lineHeight: '1.6' }}>
              Would you like to add tutorial todos? They'll teach you todo.txt syntax and show off all the features.
            </p>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button
                className="badge"
                onClick={() => {
                  setShowDemoPrompt(false);
                  runSearch();
                }}
              >
                No Thanks
              </button>
              <button
                className="badge success"
                onClick={async () => {
                  try {
                    await (Backend as any).SeedDemoData?.();
                    setHasDemoData(true);
                    setShowDemoPrompt(false);
                    runSearch();
                  } catch (err) {
                    console.error('Failed to seed demo data:', err);
                    alert('Failed to add demo data');
                  }
                }}
              >
                Yes, Add Tutorial
              </button>
            </div>
          </div>
        </div>
      )}
      <HelpModal open={helpOpen} onOpenChange={setHelpOpen} />
      <SettingsModal open={settingsOpen} onOpenChange={setSettingsOpen} />
      <SessionContextModal
        open={sessionContextOpen}
        onOpenChange={setSessionContextOpen}
        onSessionChanged={() => {
          updateSessionFilterCount();
          runSearch();
        }}
      />
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
