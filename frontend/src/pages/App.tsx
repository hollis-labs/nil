import * as React from "react";
import TerminalList, { ItemRow } from "@/components/TerminalList";
import InboxView from "@/components/InboxView";
import VaultSwitcher from "@/components/VaultSwitcher";
import ChatPanel from "@/components/ChatPanel";
import KeyboardScope from "@/components/KeyboardScope";
import NotesModal from "@/components/NotesModal";
import EditItemModal from "@/components/EditItemModal";
import SettingsModal, { useSettings, SettingsProvider } from "@/components/SettingsModal";
import SessionContextModal from "@/components/SessionContextModal";
import SearchAutocomplete from "@/components/SearchAutocomplete";
import WelcomeDialog from "@/components/WelcomeDialog";
import HelpModal from "@/components/HelpModal";
import QuickSearchModal from "@/components/QuickSearchModal";
import AlphaWarning from "@/components/AlphaWarning";
import ConfirmDialog from "@/components/ConfirmDialog";
import PowerMenu from "@/components/PowerMenu";
import RadialMenuWrapper from "@/components/RadialMenuWrapper";
import MetaModal from "@/components/MetaModal";
import { CopyrightFooter } from "@/components/CopyrightFooter";
import { ThemeProvider } from "@/theme/ThemeProvider";
import { Settings, Plus, Search, Target, Power, HelpCircle, FileText, CheckSquare } from "lucide-react";
import { parseQuery } from "@/lib/query";
import { getActiveSession, setActiveSession, clearActiveSession } from "@/lib/sessionContext";

import * as Backend from "../../wailsjs/go/main/App";
import { config } from "../../wailsjs/go/models";
import { Quit } from "../../wailsjs/runtime/runtime";

type ViewMode = 'scope' | 'date';
type AppMode = 'todos' | 'notes' | 'inbox';

function Inner() {
  const [allRows, setAllRows] = React.useState<ItemRow[]>([]);
  const [query, setQuery] = React.useState("");
  const [notesOpen, setNotesOpen] = React.useState(false);
  const [notesItem, setNotesItem] = React.useState<any>(null);
  const [quickOpen, setQuickOpen] = React.useState(false);
  const [settingsOpen, setSettingsOpen] = React.useState(false);
  const [settingsInitialTab, setSettingsInitialTab] = React.useState<'general' | 'tabs' | 'data' | undefined>(undefined);
  const [sessionContextOpen, setSessionContextOpen] = React.useState(false);
  const [activeTabId, setActiveTabId] = React.useState<string>('1');
  const [editItem, setEditItem] = React.useState<ItemRow | null>(null);
  const [sessionFilterCount, setSessionFilterCount] = React.useState(0);
  const [sessionAsFilter, setSessionAsFilter] = React.useState(false);
  const [showWelcome, setShowWelcome] = React.useState(false);
  const [showDemoPrompt, setShowDemoPrompt] = React.useState(false);
  const [hasDemoData, setHasDemoData] = React.useState(false);
  const [helpOpen, setHelpOpen] = React.useState(false);
  const [showAlphaWarning, setShowAlphaWarning] = React.useState(false);
  const [confirmRemoveDemo, setConfirmRemoveDemo] = React.useState(false);
  const [showPowerMenu, setShowPowerMenu] = React.useState(false);
  const [radialMenuItem, setRadialMenuItem] = React.useState<{todo: ItemRow; position: {x: number; y: number}} | null>(null);
  const [animatingRow, setAnimatingRow] = React.useState<{ id: number; action: string; phase?: 'collapsing' | 'expanding' } | null>(null);
  const [metaModalItem, setMetaModalItem] = React.useState<ItemRow | null>(null);
  const [inboxCount, setInboxCount] = React.useState(0);
  const [vaults, setVaults] = React.useState<config.Vault[]>([]);
  const [activeVault, setActiveVault] = React.useState<config.Vault | null>(null);
  const [showVaultSwitcher, setShowVaultSwitcher] = React.useState(false);
  const [quickSearchOpen, setQuickSearchOpen] = React.useState(false);
  const [chatOpen, setChatOpen] = React.useState(false);
  const [prevAppMode, setPrevAppMode] = React.useState<'todos' | 'notes'>('todos');
  const appModeLPTimer = React.useRef<NodeJS.Timeout | null>(null);
  const appModeLPFired = React.useRef(false);
  const { settings } = useSettings();
  const [viewMode] = React.useState<ViewMode>(() => {
    const saved = localStorage.getItem('nil.viewMode');
    return (saved as ViewMode) || settings.defaultView || 'scope';
  });
  const [appMode, setAppMode] = React.useState<AppMode>(() => {
    const saved = localStorage.getItem('nil.appMode');
    const mode = (saved as AppMode) || 'todos';
    // Never restore inbox mode from localStorage
    return mode === 'inbox' ? 'todos' : mode;
  });
  const [inputMode, setInputMode] = React.useState<'search' | 'add'>(() => {
    return settings.defaultInputMode || 'add';
  });

  React.useEffect(() => {
    localStorage.setItem('nil.viewMode', viewMode);
  }, [viewMode]);

  React.useEffect(() => {
    // Don't persist inbox mode
    if (appMode !== 'inbox') {
      localStorage.setItem('nil.appMode', appMode);
    }
    // When switching modes, ensure activeTabId points to a tab in the new mode
    if (appMode !== 'inbox') {
      const currentTab = settings.tabs.find(t => t.id === activeTabId);
      if (!currentTab || (currentTab.appMode || 'todos') !== appMode) {
        const firstMatch = settings.tabs.find(t => (t.appMode || 'todos') === appMode);
        if (firstMatch) {
          setActiveTabId(firstMatch.id);
        }
      }
    }
  }, [appMode, settings.tabs]);

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

    // If session has useAsFilterTab enabled but no actual filters, clear it
    if (session.useAsFilterTab && count === 0) {
      setActiveSession({ ...session, useAsFilterTab: false });
      setSessionFilterCount(0);
      setSessionAsFilter(false);
      return;
    }

    setSessionFilterCount(count);
    setSessionAsFilter(session.useAsFilterTab || false);
  }, []);

  const refreshInboxCount = React.useCallback(async () => {
    try {
      const count = await Backend.GetInboxCount();
      setInboxCount(count);
    } catch (err) {
      console.error('Failed to get inbox count:', err);
    }
  }, []);

  const loadVaults = React.useCallback(async () => {
    try {
      const [allVaults, active] = await Promise.all([
        Backend.GetVaults(),
        Backend.GetActiveVault(),
      ]);
      setVaults(allVaults ?? []);
      setActiveVault(active ?? null);
    } catch (err) {
      console.error('Failed to load vaults:', err);
    }
  }, []);

  // Ref so handleVaultSwitch can call the latest runSearch without a forward-reference TDZ error
  const runSearchRef = React.useRef<() => void>(() => {});

  const handleVaultSwitch = React.useCallback(async (vault: config.Vault) => {
    setShowVaultSwitcher(false);
    try {
      await Backend.SwitchVault(vault.id);
      setActiveVault(vault);
      runSearchRef.current();
      refreshInboxCount();
    } catch (err) {
      console.error('Failed to switch vault:', err);
    }
  }, [refreshInboxCount]);

  // Cmd+Shift+V / Ctrl+Shift+V → vault switcher; Cmd+, / Ctrl+, → settings
  // Cmd+Shift+C / Ctrl+Shift+C → chat panel
  React.useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'v') {
        e.preventDefault();
        setShowVaultSwitcher(v => !v);
      }
      if ((e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === ',') {
        e.preventDefault();
        setSettingsOpen(v => !v);
      }
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'c') {
        e.preventDefault();
        setChatOpen(v => !v);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  function openInbox() {
    if (appMode !== 'inbox') setPrevAppMode(appMode as 'todos' | 'notes');
    setAppMode('inbox');
  }

  function closeInbox() {
    setAppMode(prevAppMode);
    refreshInboxCount();
  }

  React.useEffect(() => {
    // Validate and clean session on mount
    const session = getActiveSession();
    console.log('[App Mount] Session on startup:', session);
    if (session) {
      const count =
        (session.contexts?.length || 0) +
        (session.projects?.length || 0) +
        (session.tags?.length || 0) +
        (session.priority ? 1 : 0);

      console.log('[App Mount] Session filter count:', count, 'useAsFilterTab:', session.useAsFilterTab);

      // If session has useAsFilterTab enabled but no actual filters, disable it immediately
      if (session.useAsFilterTab && count === 0) {
        console.log('[App Mount] Disabling empty useAsFilterTab');
        setActiveSession({ ...session, useAsFilterTab: false });
      }

      // If session exists but has empty arrays, clear it entirely
      if (count === 0 && !session.useAsFilterTab) {
        console.log('[App Mount] Clearing empty session');
        clearActiveSession();
      }
    }
    updateSessionFilterCount();
    refreshInboxCount();
    loadVaults();
  }, [updateSessionFilterCount, refreshInboxCount, loadVaults]);

  React.useEffect(() => {
    // Check if database is set up
    Backend.NeedsSetup().then((needs: boolean) => {
      console.log('NeedsSetup:', needs);
      if (needs) {
        // Fresh install - always show alpha warning first
        console.log('Showing alpha warning');
        setShowAlphaWarning(true);
      } else {
        // Database exists, check demo data status
        console.log('Database exists, checking demo data');
        (Backend as any).HasDemoData?.().then((has: boolean) => {
          setHasDemoData(has);
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
    console.log('[runSearch] Starting search...');
    try {
      const session = getActiveSession();
      console.log('[runSearch] Session:', session);
      const sessionAsFilter = session?.useAsFilterTab ? session : null;
      console.log('[runSearch] sessionAsFilter:', sessionAsFilter);

      const activeTab = sessionAsFilter ? null : settings.tabs.find(t => t.id === activeTabId);
      const mainParsed = parseQuery(query);
      const tabParsed = activeTab?.query ? parseQuery(activeTab.query) : null;
      console.log('[runSearch] mainParsed:', mainParsed, 'tabParsed:', tabParsed);

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
      const isNotesMode = appMode === 'notes';
      const statusesToSearch = isNotesMode
        ? [] // Notes don't have completion state, fetch all
        : (merged.statuses.length > 0 ? merged.statuses : ['open']);

      const req: any = {
        query: merged.keywords.join(" "),
        page: 0,
        page_size: 500,
        sort_by: "created_at",
        sort_dir: "desc",
        statuses: statusesToSearch,
        type: isNotesMode ? 'note' : 'todo'
      };

      // Only include filter arrays if they have values
      if (merged.projects.length > 0) req.projects = merged.projects;
      if (merged.contexts.length > 0) req.contexts = merged.contexts;
      if (merged.tags.length > 0) req.tags = merged.tags;
      if (merged.priorities.length > 0) req.priorities = merged.priorities;

      console.log('[Search] Request being sent:', req);
      const res = await Backend.Search(req);
      let allResults = Array.isArray(res) ? res : [];

      // If showCompleted is enabled and no explicit status filter, also fetch completed items (not for notes)
      if (!isNotesMode && settings.showCompleted && merged.statuses.length === 0) {
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
      // Keep inbox count in sync after any search
      Backend.GetInboxCount().then(setInboxCount).catch(() => {});
    } catch (err) {
      console.error("Search failed:", err);
      setAllRows([]);
    }
  }, [query, activeTabId, settings.tabs, settings.showCompleted, appMode]);

  // Keep ref in sync so vault switch (defined earlier) always calls the latest runSearch
  runSearchRef.current = runSearch;

  React.useEffect(() => {
    console.log('[useEffect] Running search (inputMode does not affect search)');
    runSearch();
  }, [runSearch]);

  // Re-run search when active tab changes
  React.useEffect(() => {
    runSearch();
  }, [activeTabId, runSearch]);

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

  const animateAction = (id: number, action: string, callback: () => Promise<void>, shouldExpand = false) => {
    setAnimatingRow({ id, action, phase: 'collapsing' });
    setTimeout(async () => {
      await callback();
      if (shouldExpand) {
        // After collapse, trigger expand animation at new location
        setTimeout(() => {
          setAnimatingRow({ id, action, phase: 'expanding' });
          setTimeout(() => {
            setAnimatingRow(null);
          }, 240); // Same speed as collapse
        }, 50); // Small delay to ensure row is in new position
      } else {
        setAnimatingRow(null);
      }
    }, 240); // Collapse duration
  };

  async function handleToggle(id: number, checked: boolean) {
    if (checked) {
      animateAction(id, 'Item Completed!', async () => {
        await Backend.ToggleComplete(id, checked);
        runSearch();
      }, true); // Expand in Done section
    } else {
      await Backend.ToggleComplete(id, checked);
      runSearch();
    }
  }

  async function handleMoveSection(id: number, section: string) {
    const todo = allRows.find(r => r.id === id);
    if (!todo) return;
    const sectionName = section === 'now' ? 'Now' : section === 'soon' ? 'Soon' : 'Anytime';
    animateAction(id, `Moved to ${sectionName}!`, async () => {
      await Backend.UpdateItem({ ...todo, section } as any);
      runSearch();
    }, true); // Expand in new section
  }

  async function handleArchive(id: number, archived: boolean) {
    if (archived) {
      animateAction(id, 'Item Archived!', async () => {
        await Backend.Archive(id, archived);
        runSearch();
      });
    } else {
      await Backend.Archive(id, archived);
      runSearch();
    }
  }

  function handleOpenNotes(row: ItemRow) { setNotesItem(row); setNotesOpen(true); }
  async function handleRefClick(id: number, _refType: string) {
    try {
      const todo = await Backend.GetItem(id) as any;
      if (!todo) return;
      const row: ItemRow = {
        id: todo.id,
        title: todo.title,
        priority: todo.priority,
        due_at: todo.due_at,
        created_at: todo.created_at,
        threshold_at: todo.threshold_at,
        completed: todo.completed,
        archived: todo.archived,
        projects: todo.projects || [],
        contexts: todo.contexts || [],
        tags: todo.tags || [],
        notes_md: todo.notes_md,
        section: todo.section || "anytime",
        pinned: todo.pinned,
        type: todo.type,
      };
      setNotesOpen(false);
      setNotesItem(null);
      setEditItem(row);
      setQuickOpen(true);
    } catch (err) {
      console.error("Failed to open linked item:", err);
    }
  }
  async function handleSaveNotes(md: string) {
    if (!notesItem) return;
    await Backend.UpdateItem({ ...notesItem, notes_md: md } as any);
    setNotesOpen(false); setNotesItem(null); runSearch();
  }
  async function handleQuickAdd(line: string, extras: any) {
    const created = await Backend.CreateItemFromLine(line);
    const merged = {
      ...created,
      priority: extras.priority || created.priority,
      due_at: extras.due || created.due_at,
      threshold_at: extras.t || created.threshold_at,
      projects: extras.projects?.length ? extras.projects : created.projects,
      contexts: extras.contexts?.length ? extras.contexts : created.contexts,
      tags: extras.tags?.length ? extras.tags : created.tags,
      notes_md: extras.notes_md || created.notes_md,
      inbox: extras.inbox ? true : created.inbox,
    };
    await Backend.UpdateItem(merged as any);
    setQuickOpen(false);
    runSearch();
  }

  async function handleQuickAddNote(line: string, extras: any) {
    const created = await Backend.CreateNoteFromLine(line);
    const merged = {
      ...created,
      projects: extras.projects?.length ? extras.projects : created.projects,
      contexts: extras.contexts?.length ? extras.contexts : created.contexts,
      tags: extras.tags?.length ? extras.tags : created.tags,
      notes_md: extras.notes_md || created.notes_md,
      inbox: extras.inbox ? true : created.inbox,
    };
    await Backend.UpdateItem(merged as any);
    setQuickOpen(false);
    runSearch();
  }

  function handleEditItem(row: ItemRow) {
    setEditItem(row);
    setQuickOpen(true);
  }

  async function handleUpdateItem(todo: ItemRow) {
    // Just update the existing todo - don't create a new one!
    await Backend.UpdateItem(todo as any);
    setQuickOpen(false);
    setEditItem(null);
    runSearch();
  }

  async function handleCloneItem(todo: ItemRow) {
    const isNote = todo.type === 'note';
    const created = isNote
      ? await Backend.CreateNoteFromLine(todo.title)
      : await Backend.CreateItemFromLine(todo.title);
    const cloned = {
      ...created,
      priority: todo.priority,
      due_at: todo.due_at,
      threshold_at: todo.threshold_at,
      projects: todo.projects,
      contexts: todo.contexts,
      tags: todo.tags,
      notes_md: todo.notes_md,
      section: todo.section,
    };
    await Backend.UpdateItem(cloned as any);
    await runSearch();

    // Find the newly created todo and open it in edit modal
    const allTodos = await Backend.Search({
      query: '',
      page: 0,
      page_size: 500,
      sort_by: 'created_at',
      sort_dir: 'desc',
      statuses: ['open']
    } as any);
    const newTodo = allTodos.find((t: any) => t.id === created.id);
    if (newTodo) {
      setEditItem(newTodo);
      setQuickOpen(true);
    }
  }

  async function handleConvertType(todo: ItemRow) {
    const newType = todo.type === 'note' ? 'todo' : 'note';
    const label = newType === 'note' ? 'Converted to Note!' : 'Converted to Todo!';
    animateAction(todo.id, label, async () => {
      await Backend.UpdateItem({ ...todo, type: newType } as any);
      runSearch();
    }, false);
  }

  async function handleMetaSave(id: number, updates: Partial<ItemRow>) {
    const todo = allRows.find(r => r.id === id);
    if (!todo) return;
    await Backend.UpdateItem({ ...todo, ...updates } as any);
    setMetaModalItem(null);
    runSearch();
  }

  async function handlePin(id: number, pinned: boolean) {
    const todo = allRows.find(r => r.id === id);
    if (!todo) return;
    const itemName = appMode === 'notes' ? 'Note' : 'Todo';
    const message = pinned ? `${itemName} Pinned!` : `${itemName} Unpinned!`;
    animateAction(id, message, async () => {
      await Backend.UpdateItem({ ...todo, pinned } as any);
      runSearch();
    }, true); // Expand at new position (top if pinned, original if unpinned)
  }

  async function handleInputSubmit() {
    console.log('[InputSubmit] Mode:', inputMode, 'Query:', query, 'AppMode:', appMode);
    if (inputMode === 'add' && !query.trim()) {
      // Empty quick add — create an untitled inbox item
      const baseMode = appMode === 'inbox' ? prevAppMode : appMode;
      const created = baseMode === 'notes'
        ? await Backend.CreateNoteFromLine('')
        : await Backend.CreateItemFromLine('');
      console.log('[InputSubmit] Created inbox item:', created);
      await refreshInboxCount();
      return;
    }
    if (inputMode === 'add' && query.trim()) {
      console.log('[InputSubmit] Quick add mode - creating', appMode === 'notes' ? 'note' : 'todo');
      // Quick add mode
      const session = getActiveSession();
      const created = appMode === 'notes'
        ? await Backend.CreateNoteFromLine(query)
        : await Backend.CreateItemFromLine(query);
      const merged = {
        ...created,
        priority: session?.priority || created.priority,
        projects: session?.projects?.length ? session.projects : created.projects,
        contexts: session?.contexts?.length ? session.contexts : created.contexts,
        tags: session?.tags?.length ? session.tags : created.tags,
      };
      await Backend.UpdateItem(merged as any);
      setQuery('');
      // Force refresh the entire list
      try {
        const isNotesMode = appMode === 'notes';
        const req: any = {
          query: '',
          page: 0,
          page_size: 500,
          sort_by: 'created_at',
          sort_dir: 'desc',
          statuses: isNotesMode ? [] : ['open'],
          type: isNotesMode ? 'note' : 'todo'
        };
        let allResults = await Backend.Search(req);
        if (!isNotesMode && settings.showCompleted) {
          const completedReq = { ...req, statuses: ['completed'] };
          const completedRes = await Backend.Search(completedReq);
          allResults = [...allResults, ...completedRes];
        }
        setAllRows(Array.isArray(allResults) ? allResults : []);
      } catch (err) {
        console.error('Failed to refresh after add:', err);
      }
    } else {
      console.log('[InputSubmit] Search mode - running search');
      // Search mode
      runSearch();
    }
  }

  async function handleDeleteTodo(id: number) {
    const deleteMsg = appMode === 'notes' ? 'Note Deleted!' : 'Item Deleted!';
    animateAction(id, deleteMsg, async () => {
      await Backend.DeleteItem(id);
      setQuickOpen(false);
      setEditItem(null);
      runSearch();
    });
  }

  function handleClearAll() {
    if (appMode === 'inbox') {
      closeInbox();
      return;
    }
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
      background: 'var(--term-bg)',
      border: '1px solid var(--term-border)',
    }}>
      <KeyboardScope
        onQuickAdd={()=>setQuickOpen(true)}
        onEscape={handleClearAll}
        onQuickAddNote={() => {
          setAppMode('notes');
          setQuickOpen(true);
        }}
        onToggleAppMode={() => setAppMode(prev => prev === 'todos' ? 'notes' : 'todos')}
        onOpenInbox={openInbox}
        onOpenQuickSearch={() => setQuickSearchOpen(true)}
      />

      <div style={{
        width: '100%',
        flex: 1,
        minHeight: 0,
        padding: '10px',
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column'
      }} >
        {/* NIL Branding - Draggable */}
        <div style={{ position: 'relative' }}>
          <div
            style={{
              padding: '16px 20px',
              paddingTop: '26px',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              cursor: 'grab',
              userSelect: 'none',
              // @ts-ignore
              '--wails-draggable': 'drag',
              WebkitAppRegion: 'drag'
            } as any}
          >
            <div className="planck-header-wrapper" style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
              <span
                className="badge warn planck-lightning"
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
              <div className="planck-label" style={{
                fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
                fontSize: '14px',
                fontWeight: 800,
                letterSpacing: '0.12em',
                color: 'var(--term-info)',
              }}>
                NIL
              </div>
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
          </div>

          {hasDemoData && (
            <button
              className="badge warn"
              onClick={(e) => {
                e.stopPropagation();
                setConfirmRemoveDemo(true);
              }}
              style={{
                position: 'absolute',
                right: '18px',
                top: '24px',
                fontSize: '10px',
                padding: '4px 8px',
                borderRadius: '4px',
                cursor: 'pointer',
                zIndex: 50,
                // @ts-ignore
                WebkitAppRegion: 'no-drag'
              } as any}
            >
              Remove Tutorial
            </button>
          )}

          {/* Vault indicator — right-aligned with the search bar buttons below */}
          {activeVault && (
            <button
              onClick={(e) => { e.stopPropagation(); setShowVaultSwitcher(true); }}
              title={`Vault: ${activeVault.name} — Click or ⌘⇧V to switch`}
              style={{
                position: 'absolute',
                right: 10,
                top: 24,
                padding: '2px 8px',
                fontSize: '10px',
                fontFamily: 'monospace',
                background: 'var(--term-panel)',
                border: '1px solid var(--term-border)',
                borderRadius: '3px',
                color: 'var(--term-dim)',
                cursor: 'pointer',
                // @ts-ignore
                WebkitAppRegion: 'no-drag',
              } as any}
            >
              {activeVault.name}
            </button>
          )}
        </div>

        <div
          className="custom-scrollbar"
          style={{
            flex: 1,
            scrollbar: 'none',
            overflow: 'hidden',
            padding: '10px',
            // @ts-ignore
            WebkitAppRegion: 'no-drag'
          } as any}>
        {/* Search Bar */}
        <div style={{ marginBottom: '16px', display: 'flex', gap: '8px', alignItems: 'center' }}>
          <SearchAutocomplete
            value={query}
            onChange={setQuery}
            onSearch={handleInputSubmit}
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
            onClick={() => {
              const newMode = inputMode === 'search' ? 'add' : 'search';
              setInputMode(newMode);
              localStorage.setItem('nil.inputMode', newMode);
            }}
            title={inputMode === 'add' ? 'Quick Add Mode (Click to Search)' : 'Search Mode (Click to Quick Add)'}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--term-accent)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--term-border)';
            }}
          >
            {inputMode === 'add' ? (
              <Plus size={16} color="var(--term-fg)" />
            ) : (
              <Search size={16} color="var(--term-fg)" />
            )}
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
          {/* Inbox button — always leftmost */}
          <button
            className={`badge ${inboxCount > 0 ? 'warn' : ''}`}
            onClick={openInbox}
            title="Inbox — untitled captures"
            style={{
              padding: '4px 8px',
              fontSize: '11px',
              borderRadius: '6px',
              border: 'none',
              display: 'flex',
              alignItems: 'center',
              gap: '4px',
              opacity: appMode === 'inbox' ? 1 : (inboxCount > 0 ? 1 : 0.45),
              outline: appMode === 'inbox' ? '2px solid var(--term-accent)' : 'none',
            }}
          >
            Inbox{inboxCount > 0 && (
              <span style={{
                fontWeight: 'bold',
                background: 'rgba(0,0,0,0.25)',
                borderRadius: '3px',
                padding: '0 4px',
                fontSize: '10px',
              }}>
                {inboxCount}
              </span>
            )}
          </button>

          {/* Back button — only visible when in inbox mode, sits right of Inbox */}
          {appMode === 'inbox' && (
            <button
              className="badge info"
              onClick={closeInbox}
              title="Back (Esc)"
              style={{
                padding: '4px 8px',
                fontSize: '11px',
                borderRadius: '6px',
                border: 'none',
                display: 'flex',
                alignItems: 'center',
                gap: '4px',
              }}
            >
              ← Back
            </button>
          )}

          {sessionAsFilter && (
            <button
              className="badge warn"
              style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', display: 'flex', alignItems: 'center', gap: '6px', fontWeight: 'bold' }}
              title="Session Context Filter Active"
            >
              <Target size={12} />
              Session Filter
            </button>
          )}
          {appMode !== 'inbox' && (() => {
            const filteredTabs = settings.tabs.filter(t => (t.appMode || 'todos') === appMode);
            // Fallback: synthesize a read-only "All" tab if no tabs exist for this mode
            const displayTabs = filteredTabs.length > 0 ? filteredTabs : [{ id: '__all__', label: 'All', query: '', appMode: appMode as 'todos' | 'notes' }];
            return displayTabs.length > 0 && (
              <div style={{display: 'flex', gap: '0px'}}>
                {displayTabs.map((tab, index) => (
                  <button
                  type="button"
                  key={tab.id}
                   className={`badge ${!sessionAsFilter && activeTabId === tab.id ? 'warn' : 'fg'}`}
                   onClick={() => {
                     if (!sessionAsFilter) {
                       setActiveTabId(tab.id);
                     }
                   }}
                   style={{
                     flex: '0 0 auto',
                     fontSize: '11px',
                     padding: '4px 8px',
                     border: 'none',
                     borderRadius: index === 0 ? '4px 0 0 4px' : (index === displayTabs.length - 1 ? '0 4px 4px 0' : '0'),
                     borderLeft: index > 0 ? '1px solid var(--term-bgAlt)' : 'none',
                     opacity: sessionAsFilter ? 0.4 : 0.8,
                     cursor: sessionAsFilter ? 'not-allowed' : 'pointer'
                   }}
                   disabled={sessionAsFilter}
              >
                {tab.label}
              </button>
          ))}
        </div>
            );
          })()}

          <div style={{marginLeft: 'auto', display: 'flex', gap: '4px'}}>
            {/* Scope/Date toggle hidden — view mode UI to be redesigned */}
            <button
                className="badge info"
                draggable
                onDragStart={(e) => e.preventDefault()}
                onContextMenu={(e) => e.preventDefault()}
                onMouseDown={() => {
                  appModeLPFired.current = false;
                  appModeLPTimer.current = setTimeout(() => {
                    appModeLPFired.current = true;
                    setSettingsInitialTab('tabs');
                    setSettingsOpen(true);
                  }, 1000);
                }}
                onMouseUp={() => {
                  if (appModeLPTimer.current) {
                    clearTimeout(appModeLPTimer.current);
                    appModeLPTimer.current = null;
                  }
                }}
                onMouseLeave={() => {
                  if (appModeLPTimer.current) {
                    clearTimeout(appModeLPTimer.current);
                    appModeLPTimer.current = null;
                  }
                }}
                onClick={() => {
                  if (!appModeLPFired.current) {
                    setAppMode(appMode === 'todos' ? 'notes' : 'todos');
                  }
                  appModeLPFired.current = false;
                }}
                style={{
                  padding: '8px 12px',
                  fontSize: '13px',
                  borderRadius: '6px',
                  border: 'none',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  cursor: 'pointer',
                  userSelect: 'none'
                }}
                title={appMode === 'todos' ? 'Switch to Notes (long-press for tab settings)' : 'Switch to Items (long-press for tab settings)'}
            >
              {appMode === 'todos' ? (
                <>
                  <CheckSquare size={12} />
                  Items
                </>
              ) : (
                <>
                  <FileText size={12} />
                  Notes
                </>
              )}
            </button>
          </div>
        </div>

        <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
          {appMode === 'inbox' ? (
            <InboxView
              onClose={closeInbox}
              onEdit={(row) => { setEditItem(row); setQuickOpen(true); }}
              onProcessed={refreshInboxCount}
            />
          ) : (
            <TerminalList
              rows={rows}
              onToggle={handleToggle}
              onOpenNotes={handleOpenNotes}
              onMoveSection={handleMoveSection}
              onArchive={handleArchive}
              onDelete={handleDeleteTodo}
              showCompleted={settings.showCompleted}
              onEditItem={handleEditItem}
              viewMode={viewMode}
              appMode={appMode as 'todos' | 'notes'}
              onOpenRadialMenu={(todo, position) => setRadialMenuItem({todo, position})}
              closeRadialMenus={quickOpen || notesOpen || settingsOpen || sessionContextOpen || editItem !== null || radialMenuItem !== null || metaModalItem !== null}
              hasActiveFilters={query.trim().length > 0 || sessionAsFilter || (settings.tabs.find(t => t.id === activeTabId)?.query?.trim().length || 0) > 0}
              animatingRow={animatingRow}
            />
          )}
        </div>

        {/* Persistent footer — always visible regardless of content state */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          paddingTop: '8px',
        }}>
          <CopyrightFooter version="1.0.0" buildDate={new Date().toISOString().slice(0, 10)} />
          <div style={{ display: 'flex', gap: '6px' }}>
            {(['help', 'settings', 'power'] as const).map(btn => (
              <button
                key={btn}
                style={{
                  padding: '6px',
                  background: 'var(--term-panel)',
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  transition: 'border-color 0.15s ease',
                  height: '28px',
                  width: '28px',
                }}
                title={btn === 'help' ? 'Help & Guide' : btn === 'settings' ? 'Settings (⌘,)' : 'Quit NIL'}
                onMouseEnter={(e) => { e.currentTarget.style.borderColor = 'var(--term-accent)'; }}
                onMouseLeave={(e) => { e.currentTarget.style.borderColor = 'var(--term-border)'; }}
                onMouseDown={btn === 'power' ? (e) => { e.stopPropagation(); e.preventDefault(); } : undefined}
                onClick={
                  btn === 'help' ? () => setHelpOpen(true) :
                  btn === 'settings' ? () => setSettingsOpen(true) :
                  (e) => { e.stopPropagation(); e.preventDefault(); setShowPowerMenu(true); }
                }
              >
                {btn === 'help' && <HelpCircle size={14} color="var(--term-fg)" />}
                {btn === 'settings' && <Settings size={14} color="var(--term-fg)" />}
                {btn === 'power' && <Power size={14} color="var(--term-fg)" />}
              </button>
            ))}
          </div>
        </div>
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
          onComplete={async () => {
            setShowWelcome(false);
            // Check if database is empty (no todos)
            try {
              const req: any = {
                query: '',
                page: 0,
                page_size: 1,
                sort_by: 'created_at',
                sort_dir: 'desc',
                keywords: [],
                projects: [],
                contexts: [],
                tags: [],
                statuses: ['open'],
                priorities: []
              };
              const result = await Backend.Search(req);
              if (result && Array.isArray(result) && result.length === 0) {
                // Database is empty, show demo prompt
                setShowDemoPrompt(true);
              }
            } catch (err) {
              console.error('Error checking database:', err);
              // On error, show demo prompt to be safe
              setShowDemoPrompt(true);
            }
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
            <h2 style={{ margin: '0 0 16px 0', color: 'var(--term-info)', fontSize: '20px' }}>
              Welcome to Nil!
            </h2>
            <p style={{ margin: '0 0 24px 0', color: 'var(--term-fg)', lineHeight: '1.6', opacity: 0.9 }}>
              Would you like to add <strong>tutorial</strong> todos? They'll <strong>teach you</strong> todo.txt syntax and show off all the features.
            </p>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button
                className="badge"
                style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
                onClick={() => {
                  setShowDemoPrompt(false);
                  runSearch();
                }}
              >
                No Thanks
              </button>
              <button
                className="badge success"
                style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
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
      <SettingsModal open={settingsOpen} onOpenChange={(v) => { setSettingsOpen(v); if (!v) { setSettingsInitialTab(undefined); runSearch(); loadVaults(); } }} {...(settingsInitialTab ? { initialTab: settingsInitialTab } : {})} />

      <ConfirmDialog
        open={confirmRemoveDemo}
        title="Remove Tutorial Data"
        message="Are you sure you want to remove all tutorial todos? This action cannot be undone."
        confirmText="Remove"
        cancelText="Cancel"
        onConfirm={async () => {
          setConfirmRemoveDemo(false);
          try {
            await Backend.RemoveDemoData();
            setHasDemoData(false);
            runSearch();
          } catch (err) {
            console.error('Failed to remove demo data:', err);
          }
        }}
        onCancel={() => setConfirmRemoveDemo(false)}
      />

      <PowerMenu
        open={showPowerMenu}
        onQuit={() => {
          setShowPowerMenu(false);
          Quit();
        }}
        onRestart={async () => {
          setShowPowerMenu(false);
          try {
            await Backend.Restart();
          } catch (err) {
            console.error('Restart failed:', err);
          }
        }}
        onCancel={() => setShowPowerMenu(false)}
      />
      <SessionContextModal
        open={sessionContextOpen}
        onOpenChange={setSessionContextOpen}
        onSessionChanged={() => {
          updateSessionFilterCount();
          runSearch();
        }}
      />
      <NotesModal open={notesOpen} onOpenChange={setNotesOpen} todo={notesItem} onSave={handleSaveNotes} onRefClick={handleRefClick} />
      <EditItemModal
        open={quickOpen}
        onOpenChange={(v) => { setQuickOpen(v); if (!v) setEditItem(null); }}
        onSubmit={appMode === 'notes' ? handleQuickAddNote : handleQuickAdd}
        onUpdate={handleUpdateItem}
        onDelete={handleDeleteTodo}
        editItem={editItem}
        defaultContexts={defaultNewTodoFilters.contexts}
        defaultProjects={defaultNewTodoFilters.projects}
        defaultTags={defaultNewTodoFilters.tags}
        isNoteMode={editItem ? editItem.type === 'note' : appMode === 'notes'}
        onConvertType={(todo) => {
          handleConvertType(todo);
          setQuickOpen(false);
          setEditItem(null);
        }}
        onRefClick={handleRefClick}
      />
      {radialMenuItem && (
        <RadialMenuWrapper
          todo={radialMenuItem.todo}
          position={radialMenuItem.position}
          onMoveSection={handleMoveSection}
          onDelete={handleDeleteTodo}
          onArchive={handleArchive}
          onToggle={handleToggle}
          onEdit={(todo) => {
            setRadialMenuItem(null);
            handleEditItem(todo);
          }}
          onClone={(todo) => {
            setRadialMenuItem(null);
            handleCloneItem(todo);
          }}
          onMeta={(todo) => {
            setRadialMenuItem(null);
            setMetaModalItem(todo);
          }}
          onPin={handlePin}
          onConvertType={(todo) => {
            setRadialMenuItem(null);
            handleConvertType(todo);
          }}
          onClose={() => setRadialMenuItem(null)}
        />
      )}
      <MetaModal
        open={metaModalItem !== null}
        todo={metaModalItem}
        onOpenChange={(open) => !open && setMetaModalItem(null)}
        onSave={handleMetaSave}
      />
      <QuickSearchModal
        open={quickSearchOpen}
        onOpenChange={setQuickSearchOpen}
        onOpenItem={(row) => {
          setQuickSearchOpen(false);
          if (row.type === 'note') {
            setNotesItem(row);
            setNotesOpen(true);
          } else {
            setEditItem(row);
            setQuickOpen(true);
          }
        }}
      />
      <VaultSwitcher
        open={showVaultSwitcher}
        vaults={vaults}
        activeVaultId={activeVault?.id ?? ''}
        onSwitch={handleVaultSwitch}
        onClose={() => setShowVaultSwitcher(false)}
      />
      {/* Chat panel — Cmd+Shift+C / Ctrl+Shift+C */}
      <ChatPanel open={chatOpen} onClose={() => setChatOpen(false)} />
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
