import type { CloseBehaviorSetting } from "@/lib/useDirtyClose";

type Props = {
  open: boolean;
  pendingBehavior: CloseBehaviorSetting;
  onSelectBehavior: (behavior: CloseBehaviorSetting) => void;
  onDiscard: () => void;
  onSaveAndClose: () => void;
  onStay: () => void;
};

// The "Unsaved changes" prompt shown by requestClose() (via useDirtyClose)
// when the closeBehavior setting is 'ask' and the modal has unsaved edits.
// Narrow, self-contained prop surface — all decisions are delegated back to
// the caller (EditItemModal), which owns the actual save/discard/settings
// logic.
export default function ClosePromptDialog({ open, pendingBehavior, onSelectBehavior, onDiscard, onSaveAndClose, onStay }: Props) {
  if (!open) return null;

  return (
    <div style={{
      position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.65)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10,
    }}>
      <div className="terminal-card" style={{ padding: '24px', maxWidth: '360px', width: '90%' }}>
        <div style={{ fontWeight: 600, fontSize: '14px', marginBottom: '8px' }}>Unsaved changes</div>
        <div style={{ fontSize: '12px', color: 'var(--term-dim)', marginBottom: '20px' }}>
          You have unsaved changes. What would you like to do?
        </div>
        <div style={{ display: 'flex', gap: '8px', marginBottom: '20px' }}>
          <button className="badge warn" onClick={onDiscard}>Discard</button>
          <button className="badge success" onClick={onSaveAndClose}>Save &amp; Close</button>
          <button className="badge" onClick={onStay}>Stay</button>
        </div>
        <div style={{ borderTop: '1px solid var(--term-border)', paddingTop: '16px' }}>
          <div style={{ fontSize: '11px', color: 'var(--term-dim)', marginBottom: '8px' }}>
            Remember this choice:
          </div>
          <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
            {(['ask', 'always', 'never'] as const).map(opt => (
              <button key={opt}
                className={`badge ${pendingBehavior === opt ? 'info' : ''}`}
                onClick={() => onSelectBehavior(opt)}
                style={{
                  fontSize: '11px', padding: '4px 8px', border: 'none',
                  opacity: pendingBehavior === opt ? 1 : 0.6,
                }}>
                {opt === 'ask' ? 'Ask each time' : opt === 'always' ? 'Always save' : 'Never save'}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
