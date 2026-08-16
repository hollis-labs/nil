import { HelpCircle, Power, Settings } from "lucide-react";
import { CopyrightFooter } from "@/components/CopyrightFooter";

// The persistent footer row (copyright + help/settings/power buttons).
// Extracted verbatim from App.tsx's Inner() JSX — pure presentation over
// three leaf callbacks, no shared app state.
interface AppFooterBarProps {
  onHelpClick: () => void;
  onSettingsClick: () => void;
  onPowerClick: () => void;
}

export default function AppFooterBar({ onHelpClick, onSettingsClick, onPowerClick }: AppFooterBarProps) {
  return (
    <div style={{
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      paddingTop: '8px',
    }}>
      <CopyrightFooter version="1.0.0" buildDate={new Date().toISOString().slice(0, 10)} />
      <div style={{ display: 'flex', gap: '6px' }}>
        {(['help', 'settings', 'power'] as const).map(btn => (
          <button
            key={btn}
            style={{
              padding: '6px',
              background: 'var(--term-panel)',
              border: '1px solid var(--term-border)',
              borderRadius: '6px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'border-color 0.15s ease',
              height: '28px',
              width: '28px',
            }}
            title={btn === 'help' ? 'Help & Guide' : btn === 'settings' ? 'Settings (⌘,)' : 'Quit NIL'}
            onMouseEnter={(e) => { e.currentTarget.style.borderColor = 'var(--term-accent)'; }}
            onMouseLeave={(e) => { e.currentTarget.style.borderColor = 'var(--term-border)'; }}
            onMouseDown={btn === 'power' ? (e) => { e.stopPropagation(); e.preventDefault(); } : undefined}
            onClick={
              btn === 'help' ? () => onHelpClick() :
              btn === 'settings' ? () => onSettingsClick() :
              (e) => { e.stopPropagation(); e.preventDefault(); onPowerClick(); }
            }
          >
            {btn === 'help' && <HelpCircle size={14} color="var(--term-fg)" />}
            {btn === 'settings' && <Settings size={14} color="var(--term-fg)" />}
            {btn === 'power' && <Power size={14} color="var(--term-fg)" />}
          </button>
        ))}
      </div>
    </div>
  );
}
