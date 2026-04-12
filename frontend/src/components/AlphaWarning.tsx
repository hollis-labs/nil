import * as React from "react";

type Props = {
  open: boolean;
  onComplete: () => void;
};

const BeeIcon = () => (
  <svg viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg">
    <path d="M201.905,145.195c0,13.862,5.214,26.498,13.785,36.064h80.619c8.571-9.566,13.789-22.202,13.789-36.064c0-18.74-9.538-35.252-24.017-44.965l8.256-16.862c2.814-0.448,5.376-2.201,6.721-4.948c2.186-4.471,0.336-9.874-4.138-12.061c-4.472-2.194-9.875-0.344-12.058,4.134c-1.584,3.231-1.048,6.932,1.052,9.573l-7.73,15.797c-6.77-3.048-14.276-4.758-22.181-4.758c-7.912,0-15.415,1.71-22.188,4.758l-7.726-15.797c2.099-2.642,2.635-6.342,1.055-9.573c-2.19-4.478-7.59-6.328-12.061-4.134c-4.475,2.187-6.329,7.59-4.138,12.061c1.346,2.747,3.907,4.5,6.721,4.948l8.256,16.862C211.436,109.943,201.905,126.455,201.905,145.195z"/>
    <path d="M182.52,210.456c20.608-10.302,14.805-25.188,5.354-35.63C163.402,147.774,78.392,98.828,33.97,91.105C21.282,88.89-6.387,90.243,1.34,129.749c2.663,13.61,21.47,73.838,62.686,94.457C105.24,244.803,164.972,219.23,182.52,210.456z"/>
    <path d="M203.475,211.941c-4.394,4.219-10.004,7.443-15.194,10.036c-10.761,5.382-48.862,22.932-87.386,22.932c-3.413,0-6.732-0.168-9.987-0.442c-21.442,22.756-28.973,53.55-17.44,73.328c12.026,20.611,47.33,26.505,72.991,10.302c32.63-20.604,52.804-80.721,56.241-97.884C203.678,225.313,205.567,215.249,203.475,211.941z"/>
    <path d="M478.03,91.105c-44.422,7.723-129.432,56.669-153.905,83.72c-9.447,10.442-15.253,25.328,5.351,35.63c17.549,8.774,77.284,34.347,118.499,13.75c41.219-20.619,60.022-80.847,62.685-94.457C518.387,90.243,490.719,88.89,478.03,91.105z"/>
    <path d="M411.101,244.908c-38.52,0-76.621-17.549-87.382-22.932c-5.186-2.593-10.804-5.816-15.194-10.036c-2.092,3.308-0.207,13.372,0.774,18.271c3.434,17.163,23.614,77.28,56.241,97.884c25.657,16.203,60.964,10.309,72.987-10.302c11.535-19.778,4.005-50.572-17.437-73.328C417.836,244.74,414.518,244.908,411.101,244.908z"/>
    <path d="M293.997,191.562h-75.994c-2.846,0-5.155,2.312-5.155,5.158v32.841c0,2.846,2.309,5.158,5.155,5.158h75.994c2.842,0,5.151-2.312,5.151-5.158v-32.841C299.148,193.874,296.839,191.562,293.997,191.562z"/>
    <path d="M217.351,246.954c-3.346,3.182-13.729,19-20.734,39.926h118.766c-7.005-20.927-17.388-36.744-20.734-39.926H217.351z"/>
    <path d="M190.506,313.925c-0.869,7.842-0.925,15.916,0.214,23.948c0.816,5.746,2.011,11.052,3.48,15.986h123.608c1.472-4.934,2.66-10.239,3.473-15.986c1.139-8.032,1.083-16.105,0.214-23.948H190.506z"/>
    <path d="M249.726,412.356l4.692,31.67c0.217,1.458,1.472,2.538,2.95,2.538c1.479,0,2.737-1.079,2.95-2.538l4.846-32.721c10.169-3.904,26.824-12.573,39.582-30.402h-97.481C221.325,400.533,240.1,409.076,249.726,412.356z"/>
  </svg>
);

export default function AlphaWarning({ open, onComplete }: Props) {
  const [wiggleSpeed, setWiggleSpeed] = React.useState(0.4);

  React.useEffect(() => {
    if (!open) return;
    
    // Randomly change wiggle speed
    const interval = setInterval(() => {
      setWiggleSpeed(Math.random() * 0.6 + 0.3); // Speed between 0.3s - 0.9s
    }, 2000);

    return () => clearInterval(interval);
  }, [open]);

  if (!open) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.85)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 20000,
        animation: 'fadeIn 0.2s ease-out',
      }}
    >
      <div
        style={{
          background: 'var(--term-panel)',
          border: '1px solid var(--term-border)',
          borderRadius: '12px',
          padding: '2rem',
          maxWidth: '600px',
          boxShadow: '0 8px 32px rgba(0, 0, 0, 0.5)',
          display: 'flex',
          gap: '1.5rem',
          alignItems: 'flex-start',
          animation: 'slideUp 0.3s ease-out',
        }}
      >
        {/* Animated Bee Icon */}
        <div
          style={{
            width: '60px',
            height: '60px',
            minWidth: '60px',
            borderRadius: '50%',
            background: 'var(--term-accent)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '10px',
            boxShadow: '0 0 0 3px var(--term-panel), 0 4px 12px rgba(0, 0, 0, 0.4)',
            animation: `brandingWiggle ${wiggleSpeed}s ease-in-out infinite`,
          }}
        >
          <BeeIcon />
        </div>

        {/* Content */}
        <div style={{ flex: 1 }}>
          {/* Title matching NIL branding */}
          <div style={{ display: 'flex', alignItems: 'baseline', gap: '6px', marginBottom: '0.75rem' }}>
            <h3 style={{
              color: 'var(--term-accent)',
              fontSize: '1.2rem',
              margin: 0,
              letterSpacing: '0.12em',
              fontWeight: 800,
              fontFamily: '"Courier New", Courier, monospace',
            }}>
              NIL
            </h3>
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

          {/* Warning Text */}
          <div style={{
            color: 'var(--term-dim)',
            fontSize: '0.9rem',
            lineHeight: '1.7',
            marginBottom: '1.5rem',
          }}>
            <p style={{ margin: '0 0 0.75rem 0' }}>
              Thank you for trying out NIL!
            </p>
            <p style={{ margin: '0 0 0.75rem 0' }}>
              This software is in <strong style={{ color: 'var(--term-accent)', fontWeight: 600 }}>early alpha</strong>. 
              <strong style={{ fontWeight: 600 }}> Backup your data often.</strong> Not responsible for data loss. Seriously, be careful.
            </p>
            <p style={{ margin: '0', fontSize: '0.85rem', opacity: 0.8 }}>
              Please report any bugs or let me know what you think!
            </p>
          </div>

          {/* OK Button */}
          <button
            onClick={onComplete}
            style={{
              background: 'oklch(82.8% 0.189 84.429)',
              color: '#000',
              border: 'none',
              borderRadius: '6px',
              padding: '8px 12px',
              fontSize: '13px',
              fontWeight: 600,
              cursor: 'pointer',
              transition: 'all 0.2s ease',
              boxShadow: '0 2px 8px rgba(0, 0, 0, 0.2)',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.transform = 'translateY(-1px)';
              e.currentTarget.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.3)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.transform = 'translateY(0)';
              e.currentTarget.style.boxShadow = '0 2px 8px rgba(0, 0, 0, 0.2)';
            }}
          >
            OK
          </button>
        </div>
      </div>

      <style>{`
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        
        @keyframes slideUp {
          from {
            transform: translateY(20px);
            opacity: 0;
          }
          to {
            transform: translateY(0);
            opacity: 1;
          }
        }
        
        @keyframes brandingWiggle {
          0%, 100% {
            transform: rotate(0deg) translateX(0);
          }
          10%, 30%, 50%, 70%, 90% {
            transform: rotate(-1.5deg) translateX(-1px);
          }
          20%, 40%, 60%, 80% {
            transform: rotate(1.5deg) translateX(1px);
          }
        }
      `}</style>
    </div>
  );
}
