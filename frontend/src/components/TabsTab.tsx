import * as React from "react";
import { Settings, ScopeTab } from "./SettingsContext";
import { inputStyle } from "./settingsShared";

type Props = {
  local: Settings;
  setLocal: (s: Settings) => void;
};

export default function TabsTab({ local, setLocal }: Props) {
  const [draggedTabId, setDraggedTabId] = React.useState<string | null>(null);

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
    if (!removed) return;
    newTabs.splice(targetIndex, 0, removed);

    setLocal({ ...local, tabs: newTabs });
    setDraggedTabId(null);
  };

  const handleDragEnd = () => {
    setDraggedTabId(null);
  };

  return (
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
        {local.tabs.map((tab) => (
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
  );
}
