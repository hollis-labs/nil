import * as React from "react";
import { useTermTheme } from "@/theme/ThemeProvider";
import { themePresets, TermTheme } from "@/theme/theme";
import CustomScrollbar from "@/components/CustomScrollbar";
import * as Backend from "../../wailsjs/go/main/App";

type Settings = {
  showCompleted: boolean;
  tabs: ScopeTab[];
  dbPath?: string;
  defaultView?: 'scope' | 'date';
};

type ScopeTab = {
  id: string;
  label: string;
  query: string;
};

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
};

const defaultSettings: Settings = {
  showCompleted: false,
  tabs: [
    { id: '1', label: 'All', query: '' },
    { id: '2', label: 'Work', query: '@work' },
    { id: '3', label: 'Home', query: '@home' },
  ],
  dbPath: './data',
  defaultView: 'scope'
};

const SettingsContext = React.createContext<{
  settings: Settings;
  setSettings: (s: Settings) => void;
} | null>(null);

export function SettingsProvider({ children }: { children: React.ReactNode }) {
  const [settings, setSettingsState] = React.useState<Settings>(() => {
    const saved = localStorage.getItem('todo.settings');
    return saved ? JSON.parse(saved) : defaultSettings;
  });

  const setSettings = React.useCallback((s: Settings) => {
    setSettingsState(s);
    localStorage.setItem('todo.settings', JSON.stringify(s));
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

export default function SettingsModal({ open, onOpenChange }: Props) {
  const { settings, setSettings } = useSettings();
  const [local, setLocal] = React.useState(settings);
  const [activeTab, setActiveTab] = React.useState<'general' | 'tabs' | 'theme' | 'data'>('general');
  const { theme, setTheme } = useTermTheme();
  const [localTheme, setLocalTheme] = React.useState(() => ({ ...themePresets.default, ...theme }));
  const [customTheme, setCustomTheme] = React.useState(() => ({ ...themePresets.default, ...theme }));
  const [selectedPreset, setSelectedPreset] = React.useState<string>('custom');
  const [draggedTabId, setDraggedTabId] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (open) {
      setLocal(settings);
      setActiveTab('general');
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
    if (activeTab === 'theme') {
      setTheme(localTheme as TermTheme);
    }
    onOpenChange(false);
  };

  const addTab = () => {
    if (local.tabs.length >= 6) return;
    const newTab: ScopeTab = {
      id: Date.now().toString(),
      label: 'New Tab',
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

  const TabButton = ({ name, label }: { name: 'general' | 'tabs' | 'theme' | 'data', label: string }) => (
    <button
      className={`badge ${activeTab === name ? 'success' : ''}`}
      onClick={() => setActiveTab(name)}
      style={{ padding: '6px 12px' }}
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
              <button className="badge" onClick={() => onOpenChange(false)}>Close</button>
            </div>

            {/* Tab Navigation */}
            <div style={{ display: 'flex', gap: '8px', marginBottom: '20px', paddingBottom: '12px', borderBottom: '1px solid var(--term-border)' }}>
              <TabButton name="general" label="General" />
              <TabButton name="tabs" label="Scope Tabs" />
              <TabButton name="theme" label="Theme" />
              <TabButton name="data" label="Import/Export" />
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
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 0' }}>
                <input
                  type="checkbox"
                  id="showCompleted"
                  checked={local.showCompleted}
                  onChange={(e) => setLocal({ ...local, showCompleted: e.target.checked })}
                  style={{ cursor: 'pointer' }}
                />
                <label htmlFor="showCompleted" style={{ cursor: 'pointer', fontSize: '13px' }}>
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
                <div style={{ display: 'flex', gap: '8px' }}>
                  <button
                    className={`badge ${local.defaultView === 'scope' ? 'success' : ''}`}
                    onClick={() => setLocal({ ...local, defaultView: 'scope' })}
                    style={{ padding: '6px 12px' }}
                  >
                    Scope (Now/Soon/Anytime)
                  </button>
                  <button
                    className={`badge ${local.defaultView === 'date' ? 'success' : ''}`}
                    onClick={() => setLocal({ ...local, defaultView: 'date' })}
                    style={{ padding: '6px 12px' }}
                  >
                    Date
                  </button>
                </div>
                <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
                  Choose which view to show when the app loads
                </div>
              </div>
              
              <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px', marginTop: '24px' }}>Database Location</h3>
              <div style={{ marginBottom: '8px' }}>
                <label style={{ fontSize: '12px', display: 'block', marginBottom: '6px' }} className="text-dim">
                  Current Database Path
                </label>
                <div style={{
                  ...inputStyle,
                  background: 'var(--term-panel)',
                  opacity: 0.7,
                  fontFamily: 'monospace',
                  fontSize: '12px'
                }}>
                  ./data/todo.db
                </div>
              </div>
              
              <details style={{ marginTop: '12px' }}>
                <summary style={{ 
                  fontSize: '12px', 
                  cursor: 'pointer', 
                  padding: '8px',
                  background: 'var(--term-panel)',
                  borderRadius: '4px',
                  border: '1px solid var(--term-border)',
                  marginBottom: '8px'
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
                  disabled={local.tabs.length >= 6}
                  style={{ opacity: local.tabs.length >= 6 ? 0.5 : 1 }}
                >
                  + Add Tab ({local.tabs.length}/6)
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
                      {local.tabs.length > 1 && (
                        <div style={{ display: 'flex', alignItems: 'flex-end' }}>
                          <button
                            className="badge warn"
                            onClick={() => deleteTab(tab.id)}
                            style={{ padding: '6px 10px' }}
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

          {/* Theme Settings */}
          {activeTab === 'theme' && (
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
                    style={{ fontSize: '11px', padding: '4px 8px' }}
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
                    style={{ fontSize: '11px', padding: '4px 8px' }}
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
                        <span className="badge success" style={{ fontSize: '10px' }}>Active</span>
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
                      <span className="badge info" style={{ fontSize: '10px' }}>Editing</span>
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
                      style={{ marginTop: '8px' }}
                    >
                      Edit Custom Theme
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Import/Export Settings */}
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
                  style={{ flex: 1, padding: '10px 16px', fontSize: '13px' }}
                >
                  Export to todo.txt
                </button>
                
                <label 
                  htmlFor="import-file"
                  className="badge success"
                  style={{ flex: 1, padding: '10px 16px', fontSize: '13px', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
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
            <button className="badge" onClick={() => onOpenChange(false)}>Cancel</button>
            <button className="badge success" onClick={handleSave}>Save Settings</button>
          </div>
        </div>
      </div>

    </>
  );
}
