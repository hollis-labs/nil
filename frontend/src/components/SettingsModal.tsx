import * as React from "react";
import ThemeSettingsModal from "./ThemeSettingsModal";

type Settings = {
  showCompleted: boolean;
  tabs: ScopeTab[];
};

type ScopeTab = {
  id: string;
  label: string;
  context?: string;
  project?: string;
};

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
};

const defaultSettings: Settings = {
  showCompleted: false,
  tabs: [
    { id: '1', label: 'All', context: '', project: '' },
    { id: '2', label: 'Work', context: 'work', project: '' },
    { id: '3', label: 'Home', context: 'home', project: '' },
  ]
};

export function useSettings() {
  const [settings, setSettingsState] = React.useState<Settings>(() => {
    const saved = localStorage.getItem('todo.settings');
    return saved ? JSON.parse(saved) : defaultSettings;
  });

  const setSettings = (s: Settings) => {
    setSettingsState(s);
    localStorage.setItem('todo.settings', JSON.stringify(s));
  };

  return { settings, setSettings };
}

export default function SettingsModal({ open, onOpenChange }: Props) {
  const { settings, setSettings } = useSettings();
  const [local, setLocal] = React.useState(settings);
  const [activeTab, setActiveTab] = React.useState<'general' | 'tabs' | 'theme'>('general');
  const [themeOpen, setThemeOpen] = React.useState(false);

  React.useEffect(() => {
    if (open) setLocal(settings);
  }, [open, settings]);

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
    onOpenChange(false);
  };

  const addTab = () => {
    if (local.tabs.length >= 6) return;
    const newTab: ScopeTab = {
      id: Date.now().toString(),
      label: 'New Tab',
      context: '',
      project: ''
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

  const TabButton = ({ name, label }: { name: 'general' | 'tabs' | 'theme', label: string }) => (
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
          style={{ width: '700px', maxWidth: '95vw', padding: '20px', maxHeight: '90vh', overflow: 'auto' }} 
          onClick={(e) => e.stopPropagation()}
        >
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
            <div style={{ fontWeight: 600, fontSize: '16px' }}>Settings</div>
            <button className="badge" onClick={() => onOpenChange(false)}>Close</button>
          </div>

          {/* Tab Navigation */}
          <div style={{ display: 'flex', gap: '8px', marginBottom: '20px', paddingBottom: '12px', borderBottom: '1px solid var(--term-border)' }}>
            <TabButton name="general" label="General" />
            <TabButton name="tabs" label="Scope Tabs" />
            <TabButton name="theme" label="Theme" />
          </div>

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
                Create custom scope filters. Filter by context (@work, @home) or project (+myproject). Leave both empty for "All" view.
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {local.tabs.map((tab, idx) => (
                  <div 
                    key={tab.id} 
                    style={{ 
                      padding: '12px', 
                      background: 'var(--term-bg)', 
                      border: '1px solid var(--term-border)', 
                      borderRadius: '6px' 
                    }}
                  >
                    <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
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
                      <div style={{ flex: 1 }}>
                        <label style={{ fontSize: '11px', display: 'block', marginBottom: '4px' }} className="text-dim">
                          Context Filter
                        </label>
                        <input
                          style={inputStyle}
                          value={tab.context || ''}
                          onChange={(e) => updateTab(tab.id, { context: e.target.value })}
                          placeholder="e.g., work, home"
                        />
                      </div>
                      <div style={{ flex: 1 }}>
                        <label style={{ fontSize: '11px', display: 'block', marginBottom: '4px' }} className="text-dim">
                          Project Filter
                        </label>
                        <input
                          style={inputStyle}
                          value={tab.project || ''}
                          onChange={(e) => updateTab(tab.id, { project: e.target.value })}
                          placeholder="e.g., myproject"
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
              <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Color Theme</h3>
              <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
                Customize the color scheme of your todo app.
              </div>
              <button 
                className="badge info"
                onClick={() => setThemeOpen(true)}
                style={{ padding: '8px 16px' }}
              >
                Open Theme Editor
              </button>
            </div>
          )}

          {/* Footer Actions */}
          <div style={{ 
            display: 'flex', 
            justifyContent: 'flex-end', 
            gap: '8px', 
            marginTop: '20px', 
            paddingTop: '16px', 
            borderTop: '1px solid var(--term-border)' 
          }}>
            <button className="badge" onClick={() => onOpenChange(false)}>Cancel</button>
            <button className="badge success" onClick={handleSave}>Save Settings</button>
          </div>
        </div>
      </div>

      <ThemeSettingsModal open={themeOpen} onOpenChange={setThemeOpen} />
    </>
  );
}
