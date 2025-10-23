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
  scrollbarBg?: string;
  scrollbarThumb?: string;
  scrollbarThumbHover?: string;
  // Tag gradient colors (3 shades)
  tagColor1?: string;
  tagColor2?: string;
  tagColor3?: string;
  // Project gradient colors (3 shades)
  projectColor1?: string;
  projectColor2?: string;
  projectColor3?: string;
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
    scrollbarBg: '#0f131a',
    scrollbarThumb: '#1f2937',
    scrollbarThumbHover: '#2d3748',
    tagColor1: '#b4b8c0',
    tagColor2: '#999da5',
    tagColor3: '#7e8289',
    projectColor1: '#9dc1fc',
    projectColor2: '#7aa2f7',
    projectColor3: '#5883d8',
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
    scrollbarBg: '#f1f5f9',
    scrollbarThumb: '#cbd5e1',
    scrollbarThumbHover: '#94a3b8',
    tagColor1: '#94a3b8',
    tagColor2: '#64748b',
    tagColor3: '#475569',
    projectColor1: '#60a5fa',
    projectColor2: '#3b82f6',
    projectColor3: '#2563eb',
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
    scrollbarBg: '#241b2f',
    scrollbarThumb: '#495495',
    scrollbarThumbHover: '#ff7edb',
    tagColor1: '#c8b5e6',
    tagColor2: '#9d8ec9',
    tagColor3: '#7367ab',
    projectColor1: '#ffb3ec',
    projectColor2: '#ff7edb',
    projectColor3: '#e64cc9',
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
    scrollbarBg: '#0a0a0a',
    scrollbarThumb: '#1a1a1a',
    scrollbarThumbHover: '#00aa00',
    tagColor1: '#00dd00',
    tagColor2: '#00bb00',
    tagColor3: '#009900',
    projectColor1: '#00ffff',
    projectColor2: '#00dddd',
    projectColor3: '#00bbbb',
  },
};

export const defaultTheme: TermTheme = themePresets.default;

export function applyTheme(t: TermTheme) {
  const root = document.documentElement;
  
  root.style.setProperty('--term-bg', t.bg);
  root.style.setProperty('--term-panel', t.panel);
  root.style.setProperty('--term-fg', t.fg);
  root.style.setProperty('--term-dim', t.dim);
  root.style.setProperty('--term-accent', t.accent);
  root.style.setProperty('--term-success', t.success);
  root.style.setProperty('--term-warn', t.warn);
  root.style.setProperty('--term-info', t.info);
  root.style.setProperty('--term-border', t.border);
  
  const scrollbarBg = t.scrollbarBg || t.panel;
  const scrollbarThumb = t.scrollbarThumb || t.border;
  const scrollbarThumbHover = t.scrollbarThumbHover || t.dim;
  
  root.style.setProperty('--scrollbar-bg', scrollbarBg);
  root.style.setProperty('--scrollbar-thumb', scrollbarThumb);
  root.style.setProperty('--scrollbar-thumb-hover', scrollbarThumbHover);
  
  console.log('[Theme] Applied theme:', {
    name: t === themePresets.default ? 'default' : 
          t === themePresets.terminal ? 'terminal' :
          t === themePresets.light ? 'light' :
          t === themePresets.synthwave ? 'synthwave' : 'custom',
    bg: t.bg,
    panel: t.panel,
    accent: t.accent,
    scrollbarBg,
    scrollbarThumb,
    scrollbarThumbHover
  });
  
  // Verify CSS variables are set
  const computedBg = getComputedStyle(root).getPropertyValue('--scrollbar-bg');
  const computedThumb = getComputedStyle(root).getPropertyValue('--scrollbar-thumb');
  console.log('[Theme] Computed CSS variables:', {
    '--scrollbar-bg': computedBg,
    '--scrollbar-thumb': computedThumb
  });
}
