import * as React from 'react';
import { TermTheme, defaultTheme, applyTheme } from './theme';

type Ctx = { theme: TermTheme; setTheme: (t: TermTheme) => void };
const ThemeCtx = React.createContext<Ctx>({ theme: defaultTheme, setTheme: ()=>{} });
export const useTermTheme = () => React.useContext(ThemeCtx);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = React.useState<TermTheme>(() => {
    const saved = localStorage.getItem('todo.term.theme');
    let initialTheme = defaultTheme;
    if (saved) {
      try {
        const parsed = JSON.parse(saved);
        // Only add missing scrollbar properties if they don't exist
        // This preserves user's custom theme while adding new features
        const merged = {
          ...parsed,
          scrollbarBg: parsed.scrollbarBg || defaultTheme.scrollbarBg,
          scrollbarThumb: parsed.scrollbarThumb || defaultTheme.scrollbarThumb,
          scrollbarThumbHover: parsed.scrollbarThumbHover || defaultTheme.scrollbarThumbHover,
        };
        console.log('[Theme] Loaded theme with properties:', Object.keys(merged));
        initialTheme = merged;
      } catch (e) {
        console.error('[Theme] Failed to parse saved theme, using default', e);
        initialTheme = defaultTheme;
      }
    } else {
      console.log('[Theme] No saved theme, using default');
    }
    // Apply theme immediately during initialization
    applyTheme(initialTheme);
    return initialTheme;
  });
  const setTheme = (t: TermTheme) => {
    console.log('[Theme] Setting theme with properties:', Object.keys(t));
    setThemeState(t);
    localStorage.setItem('todo.term.theme', JSON.stringify(t));
    applyTheme(t);
  };
  React.useEffect(()=>{ 
    console.log('[Theme] useEffect applying theme');
    applyTheme(theme); 
  }, [theme]);
  return <ThemeCtx.Provider value={{ theme, setTheme }}>{children}</ThemeCtx.Provider>;
}
