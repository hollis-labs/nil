import * as React from "react";
import { useTermTheme } from "@/theme/ThemeProvider";
import { themePresets, TermTheme } from "@/theme/theme";
import CustomScrollbar from "@/components/CustomScrollbar";
import ConfirmDialog from "@/components/ConfirmDialog";
import * as Backend from "../../wailsjs/go/main/App";
import { config } from "../../wailsjs/go/models";
import { Check } from "lucide-react";

type Settings = {
  showCompleted: boolean;
  tabs: ScopeTab[];
  dbPath?: string;
  defaultView?: 'scope' | 'date';
  defaultTags?: string[];
  defaultInputMode?: 'search' | 'add';
  closeBehavior?: 'never' | 'always' | 'ask';
};

type ScopeTab = {
  id: string;
  label: string;
  query: string;
  appMode?: 'todos' | 'notes'; // defaults to 'todos'
};

type SettingsTab = 'general' | 'tabs' | 'data' | 'vaults'; // 'theme' hidden until Tailwind migration

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  initialTab?: SettingsTab;
};

const defaultSettings: Settings = {
  showCompleted: false,
  tabs: [
    { id: '1', label: 'All', query: '', appMode: 'todos' },
    { id: '2', label: 'All', query: '', appMode: 'notes' },
  ],
  dbPath: './data',
  defaultView: 'scope',
  defaultTags: [],
  defaultInputMode: 'add',
  closeBehavior: 'ask',
};

const SettingsContext = React.createContext<{
  settings: Settings;
  setSettings: (s: Settings) => void;
} | null>(null);

export function SettingsProvider({ children }: { children: React.ReactNode }) {
  const [settings, setSettingsState] = React.useState<Settings>(() => {
    // Migrate old key on first load
    const oldData = localStorage.getItem('todo.settings');
    if (oldData && !localStorage.getItem('nanite.settings')) {
      localStorage.setItem('nanite.settings', oldData);
      localStorage.removeItem('todo.settings');
    }

    const saved = localStorage.getItem('nanite.settings');
    let parsed: Settings = saved ? JSON.parse(saved) : defaultSettings;

    // Inject notes-mode All tab if missing (one-time migration)
    if (!localStorage.getItem('nanite.settings.migrated.notesTab')) {
      const hasNoteTab = parsed.tabs.some(t => (t.appMode || 'todos') === 'notes');
      if (!hasNoteTab) {
        parsed = {
          ...parsed,
          tabs: [...parsed.tabs, { id: 'notes-all', label: 'All', query: '', appMode: 'notes' }],
        };
        localStorage.setItem('nanite.settings', JSON.stringify(parsed));
      }
      localStorage.setItem('nanite.settings.migrated.notesTab', '1');
    }

    return parsed;
  });

  const setSettings = React.useCallback((s: Settings) => {
    setSettingsState(s);
    localStorage.setItem('nanite.settings', JSON.stringify(s));
  }, []);

  return (
    <SettingsContext.Provider value={{ settings, setSettings }}>
      {children}
    </SettingsContext.Provider>
  );
}

export function useSettings() {
  const context = React.useContext(SettingsContext);
  if (!context) {
    throw new Error('useSettings must be used within SettingsProvider');
  }
  return context;
}

export default function SettingsModal({ open, onOpenChange, initialTab }: Props) {
  const { settings, setSettings } = useSettings();
  const [local, setLocal] = React.useState(settings);
  const [activeTab, setActiveTab] = React.useState<SettingsTab>('general');
  const { theme, setTheme } = useTermTheme();
  const [localTheme, setLocalTheme] = React.useState(() => ({ ...themePresets.default, ...theme }));
  const [customTheme, setCustomTheme] = React.useState(() => ({ ...themePresets.default, ...theme }));
  const [selectedPreset, setSelectedPreset] = React.useState<string>('custom');
  const [draggedTabId, setDraggedTabId] = React.useState<string | null>(null);
  const [databasePath, setDatabasePath] = React.useState<string>('');
  const [changingDbPath, setChangingDbPath] = React.useState(false);
  const [showRestartPrompt, setShowRestartPrompt] = React.useState(false);
  const [apiConfig, setApiConfig] = React.useState<{ enabled: boolean; port: number; api_key: string } | null>(null);
  const [apiKeyCopied, setApiKeyCopied] = React.useState(false);
  const [vaults, setVaults] = React.useState<config.Vault[]>([]);
  const [activeVaultId, setActiveVaultId] = React.useState<string>('');
  const [editingVaultId, setEditingVaultId] = React.useState<string | null>(null);
  const [editingVaultName, setEditingVaultName] = React.useState('');
  const [newVaultName, setNewVaultName] = React.useState('');
  const [newVaultDir, setNewVaultDir] = React.useState('');
  const [showNewVaultForm, setShowNewVaultForm] = React.useState(false);
  const [confirmDeleteVaultId, setConfirmDeleteVaultId] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (open) {
      setLocal(settings);
      setActiveTab(initialTab || 'general');
      // Ensure theme has all properties by merging with default
      const fullTheme = { ...themePresets.default, ...theme };
      console.log('[Settings] Full theme properties:', Object.keys(fullTheme));
      console.log('[Settings] Scrollbar properties:', {
        scrollbarBg: fullTheme.scrollbarBg,
        scrollbarThumb: fullTheme.scrollbarThumb,
        scrollbarThumbHover: fullTheme.scrollbarThumbHover
      });
      setLocalTheme(fullTheme);
      // Check if current theme matches a preset
      const matchingPreset = Object.entries(themePresets).find(([_, preset]) =>
        JSON.stringify(preset) === JSON.stringify(theme)
      );
      setSelectedPreset(matchingPreset ? matchingPreset[0] : 'custom');
      // If it's custom, save it so we don't lose it
      if (!matchingPreset) {
        setCustomTheme(fullTheme);
      }

      // Load database path
      (Backend as any).GetDatabasePath?.().then((path: string) => {
        setDatabasePath(path);
      }).catch((err: any) => {
        console.error('Failed to load database path:', err);
        setDatabasePath('./data'); // fallback
      });

      // Load API config
      (Backend as any).GetAPIConfig?.().then((cfg: { enabled: boolean; port: number; api_key: string }) => {
        setApiConfig(cfg);
      }).catch((err: any) => {
        console.error('Failed to load API config:', err);
      });

      // Load vaults
      Promise.all([
        Backend.GetVaults(),
        Backend.GetActiveVault(),
      ]).then(([allVaults, active]) => {
        setVaults(allVaults ?? []);
        setActiveVaultId(active?.id ?? '');
      }).catch((err: any) => {
        console.error('Failed to load vaults:', err);
      });
    }
  }, [open, settings, theme]);

  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onOpenChange(false);
      }
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [open, onOpenChange]);

  if (!open) return null;

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
    padding: '6px 10px',
    background: 'var(--term-bg)',
    border: '1px solid var(--term-border)',
    borderRadius: '4px',
    color: 'var(--term-fg)',
    fontSize: '13px'
  };

  const handleSave = () => {
    setSettings(local);
    // Theme tab hidden until Tailwind migration; keep setTheme call guarded
    // if (activeTab === 'theme') { setTheme(localTheme as TermTheme); }
    onOpenChange(false);
  };

  const addTab = () => {
    const newTab: ScopeTab = {
      id: Date.now().toString(),
      label: '',
      query: ''
    };
    setLocal({ ...local, tabs: [...local.tabs, newTab] });
  };

  const updateTab = (id: string, updates: Partial<ScopeTab>) => {
    setLocal({
      ...local,
      tabs: local.tabs.map(t => t.id === id ? { ...t, ...updates } : t)
    });
  };

  const deleteTab = (id: string) => {
    if (local.tabs.length <= 1) return;
    setLocal({ ...local, tabs: local.tabs.filter(t => t.id !== id) });
  };

  const handleDragStart = (e: React.DragEvent, tabId: string) => {
    setDraggedTabId(tabId);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  };

  const handleDrop = (e: React.DragEvent, targetTabId: string) => {
    e.preventDefault();
    if (!draggedTabId || draggedTabId === targetTabId) return;

    const draggedIndex = local.tabs.findIndex(t => t.id === draggedTabId);
    const targetIndex = local.tabs.findIndex(t => t.id === targetTabId);

    const newTabs = [...local.tabs];
    const [removed] = newTabs.splice(draggedIndex, 1);
    newTabs.splice(targetIndex, 0, removed);

    setLocal({ ...local, tabs: newTabs });
    setDraggedTabId(null);
  };

  const handleDragEnd = () => {
    setDraggedTabId(null);
  };

  const TabButton = ({ name, label }: { name: SettingsTab, label: string }) => (
    <button
      className={`badge ${activeTab === name ? 'success' : ''}`}
      onClick={() => setActiveTab(name)}
      style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
    >
      {label}
    </button>
  );

  return (
    <>
      <div style={modalStyle} onClick={() => onOpenChange(false)}>
        <div
          className="terminal-card"
          style={{
            width: '700px',
            maxWidth: '95vw',
            height: '650px',
            display: 'flex',
            flexDirection: 'column'
          }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Fixed Header */}
          <div style={{ padding: '20px 20px 0 20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
              <div style={{ fontWeight: 600, fontSize: '16px' }}>Settings</div>
              <button className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Close</button>
            </div>

            {/* Tab Navigation */}
            <div style={{ display: 'flex', gap: '0px', marginBottom: '20px', paddingBottom: '12px', borderBottom: '1px solid var(--term-border)' }}>
              <button
                onClick={() => setActiveTab('general')}
                className={`badge ${activeTab === 'general' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '4px 0 0 4px',
                  border: 'none',
                  opacity: activeTab === 'general' ? 1 : 0.8
                }}
              >
                General
              </button>
              <button
                onClick={() => setActiveTab('tabs')}
                className={`badge ${activeTab === 'tabs' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'tabs' ? 1 : 0.8
                }}
              >
                Scope Tabs
              </button>
              <button
                onClick={() => setActiveTab('data')}
                className={`badge ${activeTab === 'data' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'data' ? 1 : 0.8
                }}
              >
                Data
              </button>
              <button
                onClick={() => setActiveTab('vaults')}
                className={`badge ${activeTab === 'vaults' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0 4px 4px 0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'vaults' ? 1 : 0.8
                }}
              >
                Vaults
              </button>
              {/* HIDDEN until Tailwind migration:
              <button
                onClick={() => setActiveTab('theme')}
                className={`badge ${activeTab === 'theme' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0 4px 4px 0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'theme' ? 1 : 0.8
                }}
              >
                Theme
              </button>
              */}
            </div>
          </div>

          {/* Scrollable Body */}
          <CustomScrollbar style={{
            flex: 1,
            minHeight: 0
          }}>
            <div style={{ padding: '0 20px' }}>
              {/* General Settings */}
              {activeTab === 'general' && (
            <div>
              <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Display Options</h3>
              <div 
                style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 0', cursor: 'pointer' }}
                onClick={() => setLocal({ ...local, showCompleted: !local.showCompleted })}
              >
                <div style={{
                  width: '18px',
                  height: '18px',
                  border: '1px solid var(--term-border)',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: local.showCompleted ? 'var(--term-accent)' : 'transparent',
                  transition: 'all 0.2s ease'
                }}>
                  {local.showCompleted && <Check size={14} style={{ color: '#000' }} />}
                </div>
                <label style={{ cursor: 'pointer', fontSize: '13px' }}>
                  Show completed items by default
                </label>
              </div>
              <div style={{ marginTop: '12px', fontSize: '11px' }} className="text-dim">
                When enabled, completed todos will appear in the section lists (grayed out).
              </div>

              <div style={{ marginTop: '20px' }}>
                <label style={{ fontSize: '13px', display: 'block', marginBottom: '8px' }}>
                  Default View
                </label>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, defaultView: 'scope' })}>
                    <div style={{
                      width: '16px',
                      height: '16px',
                      borderRadius: '50%',
                      border: '1px solid var(--term-border)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: local.defaultView === 'scope' ? 'var(--term-accent)' : 'transparent'
                    }}>
                      {local.defaultView === 'scope' && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
                    </div>
                    <label style={{ cursor: 'pointer', fontSize: '12px' }}>Scope</label>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, defaultView: 'date' })}>
                    <div style={{
                      width: '16px',
                      height: '16px',
                      borderRadius: '50%',
                      border: '1px solid var(--term-border)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: local.defaultView === 'date' ? 'var(--term-accent)' : 'transparent'
                    }}>
                      {local.defaultView === 'date' && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
                    </div>
                    <label style={{ cursor: 'pointer', fontSize: '12px' }}>Date</label>
                  </div>
                </div>
                <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
                  Scope organizes by Now/Soon/Anytime priority. Date groups by due date. Choose which view to show on app load.
                </div>
              </div>

              <div style={{ marginTop: '20px' }}>
                <label style={{ fontSize: '13px', display: 'block', marginBottom: '8px' }}>
                  Default Input Mode
                </label>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, defaultInputMode: 'search' })}>
                    <div style={{
                      width: '16px',
                      height: '16px',
                      borderRadius: '50%',
                      border: '1px solid var(--term-border)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: local.defaultInputMode === 'search' ? 'var(--term-accent)' : 'transparent'
                    }}>
                      {local.defaultInputMode === 'search' && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
                    </div>
                    <label style={{ cursor: 'pointer', fontSize: '12px' }}>Search</label>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, defaultInputMode: 'add' })}>
                    <div style={{
                      width: '16px',
                      height: '16px',
                      borderRadius: '50%',
                      border: '1px solid var(--term-border)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: local.defaultInputMode === 'add' ? 'var(--term-accent)' : 'transparent'
                    }}>
                      {local.defaultInputMode === 'add' && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
                    </div>
                    <label style={{ cursor: 'pointer', fontSize: '12px' }}>Add</label>
                  </div>
                </div>
                <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
                  Search mode filters todos as you type. Add mode creates a new todo when you press Enter. Toggle with Tab key anytime.
                </div>
              </div>

              <div style={{ marginTop: '20px' }}>
                <label style={{ fontSize: '13px', display: 'block', marginBottom: '8px' }}>
                  Close &amp; Save Behavior
                </label>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  {(['ask', 'always', 'never'] as const).map(opt => (
                    <div key={opt} style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, closeBehavior: opt })}>
                      <div style={{
                        width: '16px',
                        height: '16px',
                        borderRadius: '50%',
                        border: '1px solid var(--term-border)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: (local.closeBehavior ?? 'ask') === opt ? 'var(--term-accent)' : 'transparent'
                      }}>
                        {(local.closeBehavior ?? 'ask') === opt && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
                      </div>
                      <label style={{ cursor: 'pointer', fontSize: '12px' }}>
                        {opt === 'ask' ? 'Ask each time' : opt === 'always' ? 'Always save' : 'Never save'}
                      </label>
                    </div>
                  ))}
                </div>
                <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
                  When closing a todo or note with unsaved changes: save automatically, discard, or be prompted.
                </div>
              </div>

              <div style={{ marginTop: '20px' }}>
                <label style={{ fontSize: '13px', display: 'block', marginBottom: '8px' }}>
                  Default Tags
                </label>
                <input
                  type="text"
                  style={inputStyle}
                  value={(local.defaultTags || []).join(', ')}
                  onChange={(e) => {
                    const tags = e.target.value.split(',').map(t => t.trim().replace(/^#/, '')).filter(t => t);
                    setLocal({ ...local, defaultTags: tags });
                  }}
                  placeholder="e.g., inbox, review (comma-separated)"
                />
                <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
                  These tags will be automatically added to new todos when no tags or projects are specified
                </div>
              </div>

              <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px', marginTop: '24px' }}>Database Location</h3>
              <div style={{ marginBottom: '8px' }}>
                <label style={{ fontSize: '12px', display: 'block', marginBottom: '6px' }} className="text-dim">
                  Current Database Path
                </label>
                {!changingDbPath ? (
                  <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    <div style={{
                      flex: 1,
                      padding: '8px 12px',
                      background: 'var(--term-panel)',
                      border: '1px solid var(--term-border)',
                      borderRadius: '6px',
                      fontFamily: 'monospace',
                      fontSize: '12px',
                      color: 'var(--term-fg)',
                      opacity: 0.8
                    }}>
                      {databasePath || 'Loading...'}
                    </div>
                    <button
                      type="button"
                      className="badge info"
                      onClick={() => setChangingDbPath(true)}
                      style={{ padding: '8px 12px', fontSize: '12px', borderRadius: '6px', whiteSpace: 'nowrap', height: '44px' }}
                    >
                      Change Location
                    </button>
                  </div>
                ) : (
                  <div>
                    <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
                      <input
                        type="text"
                        style={{
                          flex: 1,
                          padding: '8px 12px',
                          background: 'var(--term-bg)',
                          border: '1px solid var(--term-border)',
                          borderRadius: '6px',
                          color: 'var(--term-fg)',
                          fontSize: '12px',
                          fontFamily: 'monospace'
                        }}
                        placeholder="Paste new database directory path..."
                        defaultValue={databasePath}
                        id="new-db-path-input"
                      />
                    </div>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <button
                        type="button"
                        className="badge success"
                        onClick={async () => {
                          const input = document.getElementById('new-db-path-input') as HTMLInputElement;
                          const newPath = input?.value?.trim();
                          if (newPath) {
                            try {
                              await (Backend as any).SetDatabasePath(newPath);
                              setDatabasePath(newPath);
                              setChangingDbPath(false);
                              setShowRestartPrompt(true);
                            } catch (err: any) {
                              alert(`Error: ${err.message || err}`);
                            }
                          }
                        }}
                        style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
                      >
                        Save
                      </button>
                      <button
                        type="button"
                        className="badge"
                        onClick={() => setChangingDbPath(false)}
                        style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
                      >
                        Cancel
                      </button>
                    </div>
                    <div style={{ fontSize: '11px', marginTop: '8px', color: 'var(--term-dim)' }}>
                      Paste the full path to the directory where you want to store the database.
                      <br />
                      For iCloud: <code style={{ fontSize: '10px' }}>~/Library/Mobile Documents/com~apple~CloudDocs/Planck</code>
                    </div>
                  </div>
                )}
              </div>

              <details style={{ marginTop: '12px', marginBottom: '24px' }}>
                <summary style={{
                  fontSize: '12px',
                  cursor: 'pointer',
                  padding: '8px',
                  background: 'var(--term-panel)',
                  borderRadius: '4px',
                  border: '1px solid var(--term-border)',
                  marginBottom: '16px'
                }}>
                  💡 Cloud Sync Setup (Dropbox, iCloud, Google Drive)
                </summary>
                <div style={{
                  fontSize: '11px',
                  padding: '12px',
                  background: 'var(--term-panel)',
                  borderRadius: '4px',
                  border: '1px solid var(--term-border)',
                  lineHeight: '1.6'
                }} className="text-dim">
                  <p style={{ marginBottom: '8px' }}>To sync your todos across multiple computers:</p>
                  <ol style={{ marginLeft: '16px', marginBottom: '8px' }}>
                    <li>Quit the app completely</li>
                    <li>Move the <code>data</code> folder to your cloud sync location</li>
                    <li>Create a symbolic link pointing to the new location</li>
                    <li>Restart the app</li>
                  </ol>
                  <p style={{ marginTop: '8px', padding: '6px', background: 'var(--term-bg)', borderRadius: '3px', fontFamily: 'monospace', fontSize: '10px' }}>
                    Example: ln -s ~/Dropbox/planck-data ./data
                  </p>
                  <p style={{ marginTop: '8px', color: 'var(--term-warn)' }}>
                    ⚠️ Never open the app on multiple computers simultaneously before sync completes!
                  </p>
                </div>
              </details>
            </div>
          )}

          {/* Scope Tabs Settings */}
          {activeTab === 'tabs' && (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                <h3 style={{ fontSize: '14px', fontWeight: 600 }}>Scope Filter Tabs</h3>
                <button
                  className="badge success"
                  onClick={addTab}
                  style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
                >
                  + Add Tab
                </button>
              </div>

              <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
                Create custom scope filters using search syntax: @context +project #tag pri:A
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {local.tabs.map((tab, idx) => (
                  <div
                    key={tab.id}
                    draggable
                    onDragStart={(e) => handleDragStart(e, tab.id)}
                    onDragOver={handleDragOver}
                    onDrop={(e) => handleDrop(e, tab.id)}
                    onDragEnd={handleDragEnd}
                    style={{
                      padding: '12px',
                      background: draggedTabId === tab.id ? 'var(--term-panel)' : 'var(--term-bg)',
                      border: `1px solid ${draggedTabId === tab.id ? 'var(--term-accent)' : 'var(--term-border)'}`,
                      borderRadius: '6px',
                      cursor: 'move',
                      opacity: draggedTabId === tab.id ? 0.5 : 1,
                      transition: 'all 0.2s ease'
                    }}
                  >
                    <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
                      <div style={{ display: 'flex', alignItems: 'flex-end', paddingBottom: '2px' }}>
                        <div style={{
                          cursor: 'grab',
                          padding: '4px',
                          color: 'var(--term-dim)',
                          fontSize: '16px',
                          lineHeight: '1',
                          userSelect: 'none'
                        }} title="Drag to reorder">
                          ⋮⋮
                        </div>
                      </div>
                      <div style={{ flex: 1 }}>
                        <label style={{ fontSize: '11px', display: 'block', marginBottom: '4px' }} className="text-dim">
                          Tab Label
                        </label>
                        <input
                          style={inputStyle}
                          value={tab.label}
                          onChange={(e) => updateTab(tab.id, { label: e.target.value })}
                          placeholder="e.g., Work, Home, Personal"
                        />
                      </div>
                      <div style={{ flex: 2 }}>
                        <label style={{ fontSize: '11px', display: 'block', marginBottom: '4px' }} className="text-dim">
                          Query
                        </label>
                        <input
                          style={inputStyle}
                          value={tab.query || ''}
                          onChange={(e) => updateTab(tab.id, { query: e.target.value })}
                          placeholder="e.g., @work +myproject pri:A"
                        />
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'flex-end' }}>
                        <label style={{ fontSize: '11px', display: 'block', marginBottom: '4px' }} className="text-dim">
                          Type
                        </label>
                        <button
                          type="button"
                          className={`badge ${(tab.appMode || 'todos') === 'notes' ? 'info' : 'success'}`}
                          onClick={() => updateTab(tab.id, { appMode: (tab.appMode || 'todos') === 'todos' ? 'notes' : 'todos' })}
                          style={{
                            padding: '6px 10px',
                            fontSize: '11px',
                            borderRadius: '4px',
                            border: 'none',
                            whiteSpace: 'nowrap',
                            cursor: 'pointer'
                          }}
                          title={`Currently: ${(tab.appMode || 'todos') === 'todos' ? 'Todos' : 'Notes'}. Click to toggle.`}
                        >
                          {(tab.appMode || 'todos') === 'todos' ? 'Todos' : 'Notes'}
                        </button>
                      </div>
                        {local.tabs.length > 1 && (
                        <div style={{ display: 'flex', alignItems: 'flex-end' }}>
                          <button
                            className="badge warn"
                            onClick={() => deleteTab(tab.id)}
                            style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
                          >
                            ✕
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Vaults Settings */}
          {activeTab === 'vaults' && (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                <h3 style={{ fontSize: '14px', fontWeight: 600 }}>Vaults</h3>
                <button
                  className="badge success"
                  onClick={() => setShowNewVaultForm(v => !v)}
                  style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
                >
                  {showNewVaultForm ? 'Cancel' : '+ New Vault'}
                </button>
              </div>
              <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
                Each vault is a separate SQLite database. Use ⌘⇧V to quickly switch between them.
              </div>

              {/* New vault form */}
              {showNewVaultForm && (
                <div style={{
                  padding: '12px',
                  background: 'var(--term-panel)',
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  marginBottom: '16px',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '8px'
                }}>
                  <input
                    type="text"
                    placeholder="Vault name (e.g. Work, RPG World, Research)"
                    value={newVaultName}
                    onChange={e => setNewVaultName(e.target.value)}
                    style={{
                      padding: '6px 10px',
                      background: 'var(--term-bg)',
                      border: '1px solid var(--term-border)',
                      borderRadius: '4px',
                      color: 'var(--term-fg)',
                      fontSize: '13px'
                    }}
                  />
                  <input
                    type="text"
                    placeholder="Directory path (e.g. /Users/you/Vaults/Work)"
                    value={newVaultDir}
                    onChange={e => setNewVaultDir(e.target.value)}
                    style={{
                      padding: '6px 10px',
                      background: 'var(--term-bg)',
                      border: '1px solid var(--term-border)',
                      borderRadius: '4px',
                      color: 'var(--term-fg)',
                      fontSize: '13px',
                      fontFamily: 'monospace'
                    }}
                  />
                  <button
                    className="badge success"
                    disabled={!newVaultName.trim() || !newVaultDir.trim()}
                    onClick={async () => {
                      try {
                        const created = await Backend.CreateVault(newVaultName.trim(), newVaultDir.trim());
                        if (created) setVaults(prev => [...prev, created]);
                        setNewVaultName('');
                        setNewVaultDir('');
                        setShowNewVaultForm(false);
                      } catch (err: any) {
                        alert(`Failed to create vault: ${err.message || err}`);
                      }
                    }}
                    style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', alignSelf: 'flex-start' }}
                  >
                    Create Vault
                  </button>
                </div>
              )}

              {/* Vault list */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {vaults.map(vault => {
                  const isActive = vault.id === activeVaultId;
                  const isEditing = editingVaultId === vault.id;
                  return (
                    <div
                      key={vault.id}
                      style={{
                        padding: '10px 12px',
                        background: isActive ? 'rgba(var(--term-accent-rgb, 122, 162, 247), 0.06)' : 'var(--term-panel)',
                        border: `1px solid ${isActive ? 'var(--term-accent)' : 'var(--term-border)'}`,
                        borderRadius: '6px',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px'
                      }}
                    >
                      {/* Name (editable inline) */}
                      <div style={{ flex: 1 }}>
                        {isEditing ? (
                          <input
                            autoFocus
                            value={editingVaultName}
                            onChange={e => setEditingVaultName(e.target.value)}
                            onKeyDown={async e => {
                              if (e.key === 'Enter') {
                                try {
                                  await Backend.RenameVault(vault.id, editingVaultName.trim());
                                  setVaults(prev => prev.map(v => v.id === vault.id ? { ...v, name: editingVaultName.trim() } : v));
                                } catch (err: any) {
                                  alert(`Failed to rename: ${err.message || err}`);
                                }
                                setEditingVaultId(null);
                              } else if (e.key === 'Escape') {
                                setEditingVaultId(null);
                              }
                            }}
                            onBlur={() => setEditingVaultId(null)}
                            style={{
                              padding: '2px 6px',
                              background: 'var(--term-bg)',
                              border: '1px solid var(--term-accent)',
                              borderRadius: '3px',
                              color: 'var(--term-fg)',
                              fontSize: '13px',
                              width: '100%'
                            }}
                          />
                        ) : (
                          <div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                              <span style={{ fontSize: '13px', fontWeight: isActive ? 600 : 400 }}>
                                {vault.name}
                              </span>
                              {isActive && (
                                <span className="badge success" style={{ fontSize: '10px', padding: '2px 5px', borderRadius: '3px' }}>active</span>
                              )}
                            </div>
                            <div style={{ fontSize: '11px', color: 'var(--term-dim)', fontFamily: 'monospace', marginTop: '2px' }}>
                              {vault.path}
                            </div>
                          </div>
                        )}
                      </div>

                      {/* Actions */}
                      <div style={{ display: 'flex', gap: '4px', flexShrink: 0 }}>
                        {!isActive && (
                          <button
                            className="badge info"
                            onClick={async () => {
                              try {
                                await Backend.SwitchVault(vault.id);
                                setActiveVaultId(vault.id);
                              } catch (err: any) {
                                alert(`Failed to switch: ${err.message || err}`);
                              }
                            }}
                            style={{ padding: '4px 8px', fontSize: '11px', borderRadius: '4px' }}
                          >
                            Switch To
                          </button>
                        )}
                        <button
                          className="badge"
                          onClick={() => {
                            setEditingVaultId(vault.id);
                            setEditingVaultName(vault.name);
                          }}
                          style={{ padding: '4px 8px', fontSize: '11px', borderRadius: '4px' }}
                        >
                          Rename
                        </button>
                        {vaults.length > 1 && !isActive && (
                          <button
                            className="badge warn"
                            onClick={() => setConfirmDeleteVaultId(vault.id)}
                            style={{ padding: '4px 8px', fontSize: '11px', borderRadius: '4px' }}
                          >
                            Delete
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>

              <div style={{ marginTop: '16px', fontSize: '11px', lineHeight: '1.6' }} className="text-dim">
                Deleting a vault removes it from the registry only — the database files on disk are preserved.
              </div>
            </div>
          )}

          {/* Theme Settings — HIDDEN until Tailwind migration */}
          {false && (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                <h3 style={{ fontSize: '14px', fontWeight: 600 }}>Color Theme</h3>
                <div style={{ display: 'flex', gap: '6px' }}>
                  <button
                    className="badge warn"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      alert('Clearing theme cache...');
                      console.log('[Settings] CLEARING THEME CACHE');
                      localStorage.removeItem('todo.term.theme');
                      alert('Theme cache cleared! Reloading...');
                      window.location.reload();
                    }}
                    style={{ fontSize: '13px', padding: '8px 12px', borderRadius: '6px' }}
                  >
                    Clear Cache
                  </button>
                  <button
                    className="badge success"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      console.log('[Settings] Setting to DEFAULT theme');
                      const defaultTheme = themePresets.default;
                      console.log('[Settings] Default theme:', defaultTheme);
                      setTheme(defaultTheme);
                      alert('Theme set to DEFAULT (dark blue). Check the colors!');
                    }}
                    style={{ fontSize: '13px', padding: '8px 12px', borderRadius: '6px' }}
                  >
                    Use Default
                  </button>
                </div>
              </div>
              <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
                Select a theme preset below. Each theme is ready to use.
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {Object.entries(themePresets).map(([name, preset]) => (
                  <div
                    key={name}
                    style={{
                      padding: '16px',
                      border: `2px solid ${selectedPreset === name ? 'var(--term-accent)' : 'var(--term-border)'}`,
                      borderRadius: '8px',
                      cursor: 'pointer',
                      transition: 'all 0.2s ease',
                      background: selectedPreset === name ? 'rgba(122, 162, 247, 0.05)' : 'var(--term-bg)'
                    }}
                    onClick={() => {
                      setSelectedPreset(name);
                      setLocalTheme(preset);
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                      <div style={{ fontSize: '13px', fontWeight: 600, textTransform: 'capitalize' }}>
                        {name}
                      </div>
                      {selectedPreset === name && (
                        <span className="badge success" style={{ padding: '4px 8px', fontSize: '11px', borderRadius: '6px' }}>Active</span>
                      )}
                    </div>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                      {Object.entries(preset)
                        .filter(([key]) => !key.startsWith('scrollbar'))
                        .map(([key, color]) => (
                          <div
                            key={key}
                            style={{
                              width: '28px',
                              height: '28px',
                              background: color,
                              borderRadius: '4px',
                              border: '1px solid var(--term-border)'
                            }}
                            title={`${key}: ${color}`}
                          />
                        ))}
                    </div>
                    <div style={{ display: 'flex', gap: '4px', marginTop: '6px', fontSize: '10px' }} className="text-dim">
                      <span>Scrollbar:</span>
                      <div style={{ display: 'flex', gap: '4px' }}>
                        <div
                          style={{
                            width: '20px',
                            height: '20px',
                            background: preset.scrollbarBg,
                            borderRadius: '3px',
                            border: '1px solid var(--term-border)'
                          }}
                          title={`Track: ${preset.scrollbarBg}`}
                        />
                        <div
                          style={{
                            width: '20px',
                            height: '20px',
                            background: preset.scrollbarThumb,
                            borderRadius: '3px',
                            border: '1px solid var(--term-border)'
                          }}
                          title={`Thumb: ${preset.scrollbarThumb}`}
                        />
                        <div
                          style={{
                            width: '20px',
                            height: '20px',
                            background: preset.scrollbarThumbHover,
                            borderRadius: '3px',
                            border: '1px solid var(--term-border)'
                          }}
                          title={`Hover: ${preset.scrollbarThumbHover}`}
                        />
                      </div>
                    </div>
                  </div>
                ))}

                {/* Custom Theme */}
                <div
                  style={{
                    padding: '16px',
                    border: `2px solid ${selectedPreset === 'custom' ? 'var(--term-accent)' : 'var(--term-border)'}`,
                    borderRadius: '8px',
                    transition: 'all 0.2s ease',
                    background: selectedPreset === 'custom' ? 'rgba(122, 162, 247, 0.05)' : 'var(--term-bg)'
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                    <div style={{ fontSize: '13px', fontWeight: 600 }}>
                      Custom
                    </div>
                    {selectedPreset === 'custom' && (
                      <span className="badge info" style={{ padding: '4px 8px', fontSize: '11px', borderRadius: '6px' }}>Editing</span>
                    )}
                  </div>
                  {selectedPreset === 'custom' && (
                    <div style={{ marginTop: '12px' }}>
                      {/* Base Colors */}
                      <div style={{ fontSize: '11px', fontWeight: 600, marginBottom: '8px', marginTop: '12px' }} className="text-dim">
                        Base Colors
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                        {Object.entries(localTheme)
                          .filter(([key]) => !key.startsWith('scrollbar'))
                          .map(([key, color]) => {
                            if (typeof color !== 'string') return null;
                            const label = key
                              .replace(/([A-Z])/g, ' $1')
                              .replace(/^./, (str) => str.toUpperCase())
                              .trim();

                            return (
                              <div key={key} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <label style={{ fontSize: '11px', width: '80px' }} className="text-dim">{label}</label>
                                <input
                                  type="color"
                                  value={color || '#000000'}
                                  onChange={(e)=>{
                                    const newTheme = {...localTheme, [key]: e.target.value};
                                    setLocalTheme(newTheme);
                                    setCustomTheme(newTheme);
                                  }}
                                  style={{ width: '40px', height: '28px', border: '1px solid var(--term-border)', borderRadius: '4px', cursor: 'pointer' }}
                                />
                                <input
                                  value={color || ''}
                                  onChange={(e)=>{
                                    const newTheme = {...localTheme, [key]: e.target.value};
                                    setLocalTheme(newTheme);
                                    setCustomTheme(newTheme);
                                  }}
                                  placeholder="#000000"
                                  style={{
                                    flex: 1,
                                    padding: '6px 10px',
                                    background: 'var(--term-bg)',
                                    border: '1px solid var(--term-border)',
                                    borderRadius: '4px',
                                    color: 'var(--term-fg)',
                                    fontSize: '12px',
                                    fontFamily: 'monospace'
                                  }}
                                />
                              </div>
                            );
                          })}
                      </div>

                      {/* Scrollbar Colors */}
                      <div style={{ fontSize: '11px', fontWeight: 600, marginBottom: '8px', marginTop: '16px' }} className="text-dim">
                        Scrollbar
                      </div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                        {Object.entries(localTheme)
                          .filter(([key]) => key.startsWith('scrollbar'))
                          .map(([key, color]) => {
                            if (typeof color !== 'string') return null;
                            const label = key
                              .replace(/scrollbar/i, '')
                              .replace(/([A-Z])/g, ' $1')
                              .replace(/^./, (str) => str.toUpperCase())
                              .trim();

                            return (
                              <div key={key} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <label style={{ fontSize: '11px', width: '80px' }} className="text-dim">{label}</label>
                                <input
                                  type="color"
                                  value={color || '#000000'}
                                  onChange={(e)=>{
                                    const newTheme = {...localTheme, [key]: e.target.value};
                                    setLocalTheme(newTheme);
                                    setCustomTheme(newTheme);
                                  }}
                                  style={{ width: '40px', height: '28px', border: '1px solid var(--term-border)', borderRadius: '4px', cursor: 'pointer' }}
                                />
                                <input
                                  value={color || ''}
                                  onChange={(e)=>{
                                    const newTheme = {...localTheme, [key]: e.target.value};
                                    setLocalTheme(newTheme);
                                    setCustomTheme(newTheme);
                                  }}
                                  placeholder="#000000"
                                  style={{
                                    flex: 1,
                                    padding: '6px 10px',
                                    background: 'var(--term-bg)',
                                    border: '1px solid var(--term-border)',
                                    borderRadius: '4px',
                                    color: 'var(--term-fg)',
                                    fontSize: '12px',
                                    fontFamily: 'monospace'
                                  }}
                                />
                              </div>
                            );
                          })}
                      </div>
                    </div>
                  )}
                  {selectedPreset !== 'custom' && (
                    <button
                      className="badge"
                      onClick={(e) => {
                        e.stopPropagation();
                        setSelectedPreset('custom');
                        setLocalTheme(customTheme);
                      }}
                      style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', marginTop: '8px' }}
                    >
                      Edit Custom Theme
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Import/Export + API Settings */}
          {activeTab === 'data' && (
            <div>
              <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Import/Export</h3>
              <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
                Import and export your todos in todo.txt format.
              </div>

              <div style={{ display: 'flex', gap: '12px' }}>
                <button
                  className="badge info"
                  onClick={async () => {
                    try {
                      const content = await Backend.ExportTodoTxt();
                      const blob = new Blob([content], { type: 'text/plain' });
                      const url = URL.createObjectURL(blob);
                      const a = document.createElement('a');
                      a.href = url;
                      a.download = `todos-${new Date().toISOString().slice(0, 10)}.txt`;
                      a.click();
                      URL.revokeObjectURL(url);
                    } catch (err) {
                      console.error('Export failed:', err);
                      alert('Export failed: ' + err);
                    }
                  }}
                  style={{ flex: 1, padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
                >
                  Export to todo.txt
                </button>

                <label
                  htmlFor="import-file"
                  className="badge success"
                  style={{ flex: 1, padding: '8px 12px', fontSize: '13px', borderRadius: '6px', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
                >
                  Import from todo.txt
                </label>
                <input
                  id="import-file"
                  type="file"
                  accept=".txt"
                  style={{ display: 'none' }}
                  onChange={async (e) => {
                    const file = e.target.files?.[0];
                    if (!file) return;
                    try {
                      const content = await file.text();
                      await Backend.ImportTodoTxt(content);
                      alert('Import successful!');
                      window.location.reload();
                    } catch (err) {
                      console.error('Import failed:', err);
                      alert('Import failed: ' + err);
                    }
                  }}
                />
              </div>

              {/* API Access */}
              <div style={{ marginTop: '28px', paddingTop: '20px', borderTop: '1px solid var(--term-border)' }}>
                <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>API Access</h3>
                <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
                  Enable a local HTTP API so external agents and scripts can push items to your inbox or query your data.
                </div>

                {apiConfig ? (
                  <div>
                    {/* Enable toggle */}
                    <div
                      style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px', cursor: 'pointer' }}
                      onClick={async () => {
                        const newEnabled = !apiConfig.enabled;
                        try {
                          await (Backend as any).SetAPIEnabled(newEnabled);
                          setApiConfig({ ...apiConfig, enabled: newEnabled });
                        } catch (err: any) {
                          alert(`Error: ${err.message || err}`);
                        }
                      }}
                    >
                      <div style={{
                        width: '18px',
                        height: '18px',
                        border: '1px solid var(--term-border)',
                        borderRadius: '4px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: apiConfig.enabled ? 'var(--term-accent)' : 'transparent',
                        transition: 'all 0.2s ease',
                        flexShrink: 0,
                      }}>
                        {apiConfig.enabled && <Check size={14} style={{ color: '#000' }} />}
                      </div>
                      <label style={{ cursor: 'pointer', fontSize: '13px' }}>
                        Enable local HTTP API
                      </label>
                      {apiConfig.enabled && (
                        <span style={{ fontSize: '11px', color: 'var(--term-accent)', marginLeft: '4px' }}>
                          — listening on port {apiConfig.port}
                        </span>
                      )}
                    </div>

                    {/* API Key */}
                    <div style={{ marginBottom: '8px' }}>
                      <label style={{ fontSize: '12px', display: 'block', marginBottom: '6px' }} className="text-dim">
                        API Key
                      </label>
                      <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                        <div style={{
                          flex: 1,
                          padding: '8px 12px',
                          background: 'var(--term-panel)',
                          border: '1px solid var(--term-border)',
                          borderRadius: '6px',
                          fontFamily: 'monospace',
                          fontSize: '11px',
                          color: 'var(--term-fg)',
                          overflowX: 'auto',
                          whiteSpace: 'nowrap',
                          userSelect: 'all',
                          opacity: 0.9,
                        }}>
                          {apiConfig.api_key}
                        </div>
                        <button
                          type="button"
                          className="badge"
                          onClick={() => {
                            navigator.clipboard.writeText(apiConfig.api_key);
                            setApiKeyCopied(true);
                            setTimeout(() => setApiKeyCopied(false), 2000);
                          }}
                          style={{ padding: '8px 12px', fontSize: '12px', borderRadius: '6px', whiteSpace: 'nowrap' }}
                        >
                          {apiKeyCopied ? 'Copied!' : 'Copy'}
                        </button>
                      </div>
                    </div>

                    <div style={{ fontSize: '11px', marginTop: '10px', lineHeight: '1.6' }} className="text-dim">
                      Send <code style={{ fontFamily: 'monospace', background: 'var(--term-panel)', padding: '1px 4px', borderRadius: '3px' }}>X-API-Key</code> header to authenticate.
                      {' '}Use <code style={{ fontFamily: 'monospace', background: 'var(--term-panel)', padding: '1px 4px', borderRadius: '3px' }}>X-Agent-Source</code> to identify your agent.
                      <br />
                      Endpoint: <code style={{ fontFamily: 'monospace', background: 'var(--term-panel)', padding: '1px 4px', borderRadius: '3px' }}>POST http://127.0.0.1:{apiConfig.port}/api/v1/inbox</code>
                    </div>
                  </div>
                ) : (
                  <div style={{ fontSize: '12px' }} className="text-dim">Loading API config…</div>
                )}
              </div>
            </div>
          )}
            </div>
          </CustomScrollbar>

          {/* Fixed Footer */}
          <div style={{
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px',
            padding: '16px 20px 20px 20px',
            borderTop: '1px solid var(--term-border)'
          }}>
            <button className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Cancel</button>
            <button className="badge success" onClick={handleSave} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Save Settings</button>
          </div>
        </div>
      </div>

      <ConfirmDialog
        open={showRestartPrompt}
        title="Restart Required"
        message="Database location has been updated. NANITE needs to restart to apply the changes."
        confirmText="Restart Now"
        cancelText="Later"
        onConfirm={async () => {
          setShowRestartPrompt(false);
          try {
            await Backend.Restart();
          } catch (err) {
            console.error('Restart failed:', err);
          }
        }}
        onCancel={() => {
          setShowRestartPrompt(false);
          onOpenChange(false);
        }}
      />

      <ConfirmDialog
        open={confirmDeleteVaultId !== null}
        title="Delete Vault"
        message={`Remove "${vaults.find(v => v.id === confirmDeleteVaultId)?.name ?? ''}" from the registry? Database files on disk will be preserved.`}
        confirmText="Delete"
        cancelText="Cancel"
        onConfirm={async () => {
          if (!confirmDeleteVaultId) return;
          try {
            await Backend.DeleteVault(confirmDeleteVaultId);
            setVaults(prev => prev.filter(v => v.id !== confirmDeleteVaultId));
          } catch (err: any) {
            alert(`Failed to delete vault: ${err.message || err}`);
          }
          setConfirmDeleteVaultId(null);
        }}
        onCancel={() => setConfirmDeleteVaultId(null)}
      />
    </>
  );
}
