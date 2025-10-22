import * as React from "react";
import { useTermTheme } from "@/theme/ThemeProvider";
import { themePresets } from "@/theme/theme";

const tailwindColors = {
  slate: ['#f8fafc', '#f1f5f9', '#e2e8f0', '#cbd5e1', '#94a3b8', '#64748b', '#475569', '#334155', '#1e293b', '#0f172a'],
  gray: ['#f9fafb', '#f3f4f6', '#e5e7eb', '#d1d5db', '#9ca3af', '#6b7280', '#4b5563', '#374151', '#1f2937', '#111827'],
  zinc: ['#fafafa', '#f4f4f5', '#e4e4e7', '#d4d4d8', '#a1a1aa', '#71717a', '#52525b', '#3f3f46', '#27272a', '#18181b'],
  red: ['#fef2f2', '#fee2e2', '#fecaca', '#fca5a5', '#f87171', '#ef4444', '#dc2626', '#b91c1c', '#991b1b', '#7f1d1d'],
  orange: ['#fff7ed', '#ffedd5', '#fed7aa', '#fdba74', '#fb923c', '#f97316', '#ea580c', '#c2410c', '#9a3412', '#7c2d12'],
  amber: ['#fffbeb', '#fef3c7', '#fde68a', '#fcd34d', '#fbbf24', '#f59e0b', '#d97706', '#b45309', '#92400e', '#78350f'],
  yellow: ['#fefce8', '#fef9c3', '#fef08a', '#fde047', '#facc15', '#eab308', '#ca8a04', '#a16207', '#854d0e', '#713f12'],
  lime: ['#f7fee7', '#ecfccb', '#d9f99d', '#bef264', '#a3e635', '#84cc16', '#65a30d', '#4d7c0f', '#3f6212', '#365314'],
  green: ['#f0fdf4', '#dcfce7', '#bbf7d0', '#86efac', '#4ade80', '#22c55e', '#16a34a', '#15803d', '#166534', '#14532d'],
  emerald: ['#ecfdf5', '#d1fae5', '#a7f3d0', '#6ee7b7', '#34d399', '#10b981', '#059669', '#047857', '#065f46', '#064e3b'],
  teal: ['#f0fdfa', '#ccfbf1', '#99f6e4', '#5eead4', '#2dd4bf', '#14b8a6', '#0d9488', '#0f766e', '#115e59', '#134e4a'],
  cyan: ['#ecfeff', '#cffafe', '#a5f3fc', '#67e8f9', '#22d3ee', '#06b6d4', '#0891b2', '#0e7490', '#155e75', '#164e63'],
  sky: ['#f0f9ff', '#e0f2fe', '#bae6fd', '#7dd3fc', '#38bdf8', '#0ea5e9', '#0284c7', '#0369a1', '#075985', '#0c4a6e'],
  blue: ['#eff6ff', '#dbeafe', '#bfdbfe', '#93c5fd', '#60a5fa', '#3b82f6', '#2563eb', '#1d4ed8', '#1e40af', '#1e3a8a'],
  indigo: ['#eef2ff', '#e0e7ff', '#c7d2fe', '#a5b4fc', '#818cf8', '#6366f1', '#4f46e5', '#4338ca', '#3730a3', '#312e81'],
  violet: ['#f5f3ff', '#ede9fe', '#ddd6fe', '#c4b5fd', '#a78bfa', '#8b5cf6', '#7c3aed', '#6d28d9', '#5b21b6', '#4c1d95'],
  purple: ['#faf5ff', '#f3e8ff', '#e9d5ff', '#d8b4fe', '#c084fc', '#a855f7', '#9333ea', '#7e22ce', '#6b21a8', '#581c87'],
  fuchsia: ['#fdf4ff', '#fae8ff', '#f5d0fe', '#f0abfc', '#e879f9', '#d946ef', '#c026d3', '#a21caf', '#86198f', '#701a75'],
  pink: ['#fdf2f8', '#fce7f3', '#fbcfe8', '#f9a8d4', '#f472b6', '#ec4899', '#db2777', '#be185d', '#9d174d', '#831843'],
  rose: ['#fff1f2', '#ffe4e6', '#fecdd3', '#fda4af', '#fb7185', '#f43f5e', '#e11d48', '#be123c', '#9f1239', '#881337'],
};

export default function ThemeSettingsModal({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const { theme, setTheme } = useTermTheme();
  const [local, setLocal] = React.useState(theme);
  const [activeField, setActiveField] = React.useState<string | null>(null);
  const [selectedPreset, setSelectedPreset] = React.useState<string>('custom');

  React.useEffect(()=>{ if(open) setLocal(theme); }, [open, theme]);

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
    fontSize: '13px',
    fontFamily: 'monospace'
  };

  function ColorRow({ label, keyName }: { label: string; keyName: keyof typeof theme }) {
    const isActive = activeField === keyName;
    return (
      <div style={{ marginBottom: '12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
          <label style={{ fontSize: '12px', width: '80px' }} className="text-dim">{label}</label>
          <input 
            type="color" 
            value={local[keyName] as string} 
            onChange={(e)=>setLocal({...local, [keyName]: e.target.value})}
            style={{ width: '40px', height: '28px', border: '1px solid var(--term-border)', borderRadius: '4px', cursor: 'pointer' }}
          />
          <input 
            style={inputStyle}
            value={local[keyName] as string} 
            onChange={(e)=>setLocal({...local, [keyName]: e.target.value})}
            onFocus={() => setActiveField(keyName)}
            onBlur={() => setActiveField(null)}
            placeholder="#000000"
          />
        </div>
        {isActive && (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(10, 1fr)', gap: '4px', marginTop: '6px', marginLeft: '88px' }}>
            {Object.values(tailwindColors).flat().map((color, i) => (
              <button
                key={i}
                onClick={() => setLocal({...local, [keyName]: color})}
                style={{ 
                  width: '24px', 
                  height: '24px', 
                  background: color, 
                  border: '1px solid var(--term-border)',
                  borderRadius: '3px',
                  cursor: 'pointer',
                  transition: 'transform 0.1s'
                }}
                onMouseEnter={(e) => e.currentTarget.style.transform = 'scale(1.2)'}
                onMouseLeave={(e) => e.currentTarget.style.transform = 'scale(1)'}
                title={color}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  const handlePresetChange = (presetName: string) => {
    setSelectedPreset(presetName);
    if (presetName !== 'custom') {
      setLocal(themePresets[presetName]);
    }
  };

  return (
    <div style={modalStyle} onClick={() => onOpenChange(false)}>
      <div className="terminal-card" style={{ width: '700px', maxWidth: '95vw', padding: '20px', maxHeight: '90vh', overflow: 'auto' }} onClick={(e) => e.stopPropagation()}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
          <div style={{ fontWeight: 600, fontSize: '16px' }}>Theme Settings</div>
          <button className="badge" onClick={()=>onOpenChange(false)}>Close</button>
        </div>

        {/* Theme Presets */}
        <div style={{ marginBottom: '20px' }}>
          <label style={{ fontSize: '12px', display: 'block', marginBottom: '8px' }} className="text-dim">
            Theme Presets
          </label>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            {Object.keys(themePresets).map((presetName) => (
              <button
                key={presetName}
                className={`badge ${selectedPreset === presetName ? 'success' : ''}`}
                onClick={() => handlePresetChange(presetName)}
                style={{ 
                  padding: '8px 14px', 
                  textTransform: 'capitalize',
                  cursor: 'pointer'
                }}
              >
                {presetName}
              </button>
            ))}
            <button
              className={`badge ${selectedPreset === 'custom' ? 'info' : ''}`}
              onClick={() => setSelectedPreset('custom')}
              style={{ padding: '8px 14px', cursor: 'pointer' }}
            >
              Custom
            </button>
          </div>
        </div>
        
        <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
          {selectedPreset === 'custom' 
            ? 'Click on any color field to see Tailwind color palette, or enter custom hex values like #ffd700'
            : `Using ${selectedPreset} theme preset. Switch to Custom to fine-tune colors.`}
        </div>

        {selectedPreset === 'custom' && (
          <div>
            <ColorRow label="Background" keyName="bg" />
            <ColorRow label="Panel" keyName="panel" />
            <ColorRow label="Text" keyName="fg" />
            <ColorRow label="Dim Text" keyName="dim" />
            <ColorRow label="Accent" keyName="accent" />
            <ColorRow label="Success" keyName="success" />
            <ColorRow label="Warning" keyName="warn" />
            <ColorRow label="Info" keyName="info" />
            <ColorRow label="Border" keyName="border" />
          </div>
        )}

        {selectedPreset !== 'custom' && (
          <div style={{ 
            padding: '40px 20px', 
            textAlign: 'center',
            border: '1px solid var(--term-border)',
            borderRadius: '8px',
            background: 'var(--term-bg)'
          }}>
            <div style={{ fontSize: '13px', marginBottom: '8px' }}>
              Preview of <span style={{ fontWeight: 600, textTransform: 'capitalize' }}>{selectedPreset}</span> theme
            </div>
            <div style={{ fontSize: '11px' }} className="text-dim">
              Click "Apply Theme" to use this preset, or switch to "Custom" to modify colors
            </div>
          </div>
        )}
        
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: '8px', marginTop: '20px', paddingTop: '16px', borderTop: '1px solid var(--term-border)' }}>
          <button className="badge" onClick={()=>setLocal(theme)}>Reset to Current</button>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button className="badge" onClick={()=>onOpenChange(false)}>Cancel</button>
            <button className="badge success" onClick={()=>{ setTheme(local as any); onOpenChange(false); }}>Apply Theme</button>
          </div>
        </div>
      </div>
    </div>
  );
}
