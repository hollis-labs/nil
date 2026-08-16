import type { config } from "../../wailsjs/go/models";

// The draggable NIL branding block, the "Remove Tutorial" badge, and the
// vault indicator button. Extracted verbatim from App.tsx's Inner() JSX —
// pure presentation over a handful of leaf callbacks, no shared app state.
interface AppHeaderBarProps {
  hasDemoData: boolean;
  onRemoveDemoDataClick: () => void;
  activeVault: config.Vault | null;
  onVaultIndicatorClick: () => void;
}

export default function AppHeaderBar({
  hasDemoData,
  onRemoveDemoDataClick,
  activeVault,
  onVaultIndicatorClick,
}: AppHeaderBarProps) {
  return (
    <div style={{ position: 'relative' }}>
      <div
        style={{
          padding: '16px 20px',
          paddingTop: '26px',
          display: 'flex',
          alignItems: 'center',
          gap: '6px',
          cursor: 'grab',
          userSelect: 'none',
          // @ts-ignore
          '--wails-draggable': 'drag',
          WebkitAppRegion: 'drag'
        } as any}
      >
        <div className="planck-header-wrapper" style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          <span
            className="badge warn planck-lightning"
            style={{
              padding: '2px',
              fontSize: '10px',
              borderRadius: '3px',
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: '14px',
              height: '14px',
              lineHeight: '1'
            }}
          >⚡</span>
          <div className="planck-label" style={{
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
            fontSize: '14px',
            fontWeight: 800,
            letterSpacing: '0.12em',
            color: 'var(--term-info)',
          }}>
            NIL
          </div>
        </div>
        <div style={{
          fontSize: '10px',
          color: 'var(--term-dim)',
          fontFamily: 'serif',
          fontStyle: 'italic',
          opacity: 0.6,
          letterSpacing: '0.02em'
        }}>
          <span style={{ fontStyle: 'italic' }}>(h)</span> — the quantum of action
        </div>
      </div>

      {hasDemoData && (
        <button
          className="badge warn"
          onClick={(e) => {
            e.stopPropagation();
            onRemoveDemoDataClick();
          }}
          style={{
            position: 'absolute',
            right: '18px',
            top: '24px',
            fontSize: '10px',
            padding: '4px 8px',
            borderRadius: '4px',
            cursor: 'pointer',
            zIndex: 50,
            // @ts-ignore
            WebkitAppRegion: 'no-drag'
          } as any}
        >
          Remove Tutorial
        </button>
      )}

      {/* Vault indicator — right-aligned with the search bar buttons below */}
      {activeVault && (
        <button
          onClick={(e) => { e.stopPropagation(); onVaultIndicatorClick(); }}
          title={`Vault: ${activeVault.name} — Click or ⌘⇧V to switch`}
          style={{
            position: 'absolute',
            right: 10,
            top: 24,
            padding: '2px 8px',
            fontSize: '10px',
            fontFamily: 'monospace',
            background: 'var(--term-panel)',
            border: '1px solid var(--term-border)',
            borderRadius: '3px',
            color: 'var(--term-dim)',
            cursor: 'pointer',
            // @ts-ignore
            WebkitAppRegion: 'no-drag',
          } as any}
        >
          {activeVault.name}
        </button>
      )}
    </div>
  );
}
