import { Power, RotateCw } from "lucide-react";

type Props = {
  open: boolean;
  onQuit: () => void;
  onRestart: () => void;
  onCancel: () => void;
};

export default function PowerMenu({ open, onQuit, onRestart, onCancel }: Props) {
  if (!open) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.75)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 10000,
      }}
      onClick={onCancel}
    >
      <div
        style={{
          background: 'var(--term-bg)',
          border: '1px solid var(--term-border)',
          borderRadius: '8px',
          padding: '24px',
          maxWidth: '400px',
          width: '90%',
          boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h3
          style={{
            margin: '0 0 16px 0',
            fontSize: '16px',
            fontWeight: 600,
            color: 'var(--term-fg)',
          }}
        >
          Power Options
        </h3>
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '8px',
          }}
        >
          <button
            className="badge info"
            onClick={onRestart}
            style={{
              padding: '12px 16px',
              fontSize: '13px',
              borderRadius: '6px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              justifyContent: 'center',
            }}
          >
            <RotateCw size={14} />
            Restart NIL
          </button>
          <button
            className="badge warn"
            onClick={onQuit}
            style={{
              padding: '12px 16px',
              fontSize: '13px',
              borderRadius: '6px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              justifyContent: 'center',
            }}
          >
            <Power size={14} />
            Quit NIL
          </button>
          <button
            className="badge"
            onClick={onCancel}
            style={{
              padding: '12px 16px',
              fontSize: '13px',
              borderRadius: '6px',
              cursor: 'pointer',
              background: 'var(--term-bg)',
              border: '1px solid var(--term-border)',
            }}
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
