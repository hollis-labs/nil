import * as React from "react";

export type ScopeTab = {
  id: string;
  label: string;
  query: string;
  appMode?: 'todos' | 'notes'; // defaults to 'todos'
};

export type Settings = {
  showCompleted: boolean;
  tabs: ScopeTab[];
  dbPath?: string;
  defaultView?: 'scope' | 'date';
  defaultTags?: string[];
  defaultInputMode?: 'search' | 'add';
  closeBehavior?: 'never' | 'always' | 'ask';
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
    if (oldData && !localStorage.getItem('nil.settings')) {
      localStorage.setItem('nil.settings', oldData);
      localStorage.removeItem('todo.settings');
    }

    const saved = localStorage.getItem('nil.settings');
    let parsed: Settings = saved ? JSON.parse(saved) : defaultSettings;

    // Inject notes-mode All tab if missing (one-time migration)
    if (!localStorage.getItem('nil.settings.migrated.notesTab')) {
      const hasNoteTab = parsed.tabs.some(t => (t.appMode || 'todos') === 'notes');
      if (!hasNoteTab) {
        parsed = {
          ...parsed,
          tabs: [...parsed.tabs, { id: 'notes-all', label: 'All', query: '', appMode: 'notes' }],
        };
        localStorage.setItem('nil.settings', JSON.stringify(parsed));
      }
      localStorage.setItem('nil.settings.migrated.notesTab', '1');
    }

    return parsed;
  });

  const setSettings = React.useCallback((s: Settings) => {
    setSettingsState(s);
    localStorage.setItem('nil.settings', JSON.stringify(s));
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
