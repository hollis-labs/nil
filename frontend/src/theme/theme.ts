export type TermTheme = {
  bg: string;
  panel: string;
  fg: string;
  dim: string;
  accent: string;
  success: string;
  warn: string;
  info: string;
  border: string;
};

export const themePresets: Record<string, TermTheme> = {
  default: {
    bg: '#0b0e14',
    panel: '#0f131a',
    fg: '#e6edf3',
    dim: '#8b949e',
    accent: '#7aa2f7',
    success: '#22c55e',
    warn: '#eab308',
    info: '#60a5fa',
    border: '#1f2937',
  },
  light: {
    bg: '#f8fafc',
    panel: '#ffffff',
    fg: '#1e293b',
    dim: '#64748b',
    accent: '#3b82f6',
    success: '#22c55e',
    warn: '#f59e0b',
    info: '#0ea5e9',
    border: '#e2e8f0',
  },
  synthwave: {
    bg: '#2b213a',
    panel: '#241b2f',
    fg: '#f8f8f2',
    dim: '#9d8ec9',
    accent: '#ff7edb',
    success: '#72f1b8',
    warn: '#fede5d',
    info: '#36f9f6',
    border: '#495495',
  },
  terminal: {
    bg: '#000000',
    panel: '#0a0a0a',
    fg: '#00ff00',
    dim: '#00aa00',
    accent: '#00ffff',
    success: '#00ff00',
    warn: '#ffff00',
    info: '#00aaff',
    border: '#333333',
  },
};

export const defaultTheme: TermTheme = themePresets.default;

export function applyTheme(t: TermTheme) {
  document.documentElement.style.setProperty('--term-bg', t.bg);
  document.documentElement.style.setProperty('--term-panel', t.panel);
  document.documentElement.style.setProperty('--term-fg', t.fg);
  document.documentElement.style.setProperty('--term-dim', t.dim);
  document.documentElement.style.setProperty('--term-accent', t.accent);
  document.documentElement.style.setProperty('--term-success', t.success);
  document.documentElement.style.setProperty('--term-warn', t.warn);
  document.documentElement.style.setProperty('--term-info', t.info);
  document.documentElement.style.setProperty('--term-border', t.border);
}
