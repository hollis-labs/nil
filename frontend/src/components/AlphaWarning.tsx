import * as React from "react";

type Props = {
  open: boolean;
  onComplete: () => void;
};

export default function AlphaWarning({ open, onComplete }: Props) {
  const [rotation, setRotation] = React.useState(0);
  const [speed, setSpeed] = React.useState(2);

  React.useEffect(() => {
    if (!open) return;
    
    const interval = setInterval(() => {
      setRotation(r => (r + speed) % 360);
      
      // Randomly change speed every few seconds
      if (Math.random() < 0.05) {
        setSpeed(Math.random() * 4 + 1); // Speed between 1-5
      }
    }, 50);

    return () => clearInterval(interval);
  }, [open, speed]);

  if (!open) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.9)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 20000,
      }}
    >
      <div
        style={{
          background: 'var(--term-bg)',
          border: '2px solid var(--term-accent)',
          borderRadius: '16px',
          padding: '40px',
          maxWidth: '600px',
          boxShadow: '0 12px 48px rgba(0, 0, 0, 0.6)',
          textAlign: 'center',
        }}
      >
        {/* Animated Bee */}
        <div style={{ marginBottom: '24px', display: 'flex', justifyContent: 'center' }}>
          <div
            style={{
              fontSize: '80px',
              transform: `rotate(${rotation}deg)`,
              transition: 'transform 0.05s linear',
              display: 'inline-block',
            }}
          >
            🐝
          </div>
        </div>

        {/* Title */}
        <h1
          style={{
            margin: '0 0 24px 0',
            color: 'var(--term-accent)',
            fontSize: '32px',
            fontWeight: 800,
            letterSpacing: '0.1em',
            fontFamily: '"Courier New", Courier, monospace',
          }}
        >
          PLANCK
        </h1>

        {/* Warning Text */}
        <div
          style={{
            color: 'var(--term-fg)',
            lineHeight: '1.8',
            fontSize: '16px',
            marginBottom: '32px',
          }}
        >
          <p style={{ margin: '0 0 16px 0', fontWeight: 600 }}>
            Thank you for trying out PLANCK!
          </p>
          <p style={{ margin: '0 0 16px 0' }}>
            This software is in <strong style={{ color: 'var(--term-accent)' }}>early alpha</strong>.
          </p>
          <p style={{ margin: '0 0 16px 0' }}>
            ⚠️ <strong>Backup your data often.</strong> Not responsible for data loss.
          </p>
          <p style={{ margin: '0 0 16px 0' }}>
            Seriously, be careful.
          </p>
          <p style={{ margin: '0', fontSize: '14px', opacity: 0.8 }}>
            Please report bugs or feedback - your input helps make PLANCK better!
          </p>
        </div>

        {/* OK Button */}
        <button
          className="badge success"
          onClick={onComplete}
          style={{
            padding: '14px 48px',
            fontSize: '16px',
            fontWeight: 600,
            cursor: 'pointer',
          }}
        >
          OK
        </button>
      </div>
    </div>
  );
}
