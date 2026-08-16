import * as React from "react";
import { themePresets, TermTheme } from "@/theme/theme";

// Theme tab — HIDDEN until the Tailwind/shadcn migration completes (see
// CLAUDE.md). SettingsModal renders this behind `{false && <ThemeTab .../>}`
// so it never actually mounts; kept as its own component (rather than
// deleted) so the theme system stays intact in code per project convention.
type Props = {
  setTheme: (t: TermTheme) => void;
  localTheme: TermTheme;
  setLocalTheme: React.Dispatch<React.SetStateAction<TermTheme>>;
  customTheme: TermTheme;
  setCustomTheme: React.Dispatch<React.SetStateAction<TermTheme>>;
  selectedPreset: string;
  setSelectedPreset: React.Dispatch<React.SetStateAction<string>>;
};

export default function ThemeTab({
  setTheme,
  localTheme,
  setLocalTheme,
  customTheme,
  setCustomTheme,
  selectedPreset,
  setSelectedPreset,
}: Props) {
  return (
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
              const defaultTheme = themePresets['default']!;
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
  );
}
