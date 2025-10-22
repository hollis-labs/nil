import * as React from 'react';
import { TermTheme, defaultTheme, applyTheme } from './theme';

type Ctx = { theme: TermTheme; setTheme: (t: TermTheme) => void };
const ThemeCtx = React.createContext<Ctx>({ theme: defaultTheme, setTheme: ()=>{} });
export const useTermTheme = () => React.useContext(ThemeCtx);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = React.useState<TermTheme>(() => {
    const saved = localStorage.getItem('todo.term.theme');
    return saved ? JSON.parse(saved) : defaultTheme;
  });
  const setTheme = (t: TermTheme) => {
    setThemeState(t);
    localStorage.setItem('todo.term.theme', JSON.stringify(t));
    applyTheme(t);
  };
  React.useEffect(()=>{ applyTheme(theme); }, []);
  return <ThemeCtx.Provider value={{ theme, setTheme }}>{children}</ThemeCtx.Provider>;
}
