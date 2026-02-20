import * as React from "react";
import * as Backend from "../../wailsjs/go/main/App";

type Props = {
  open: boolean;
  onComplete: () => void;
};

export default function WelcomeDialog({ open, onComplete }: Props) {
  const [selectedOption, setSelectedOption] = React.useState<'default' | 'custom' | 'existing'>('default');
  const [customPath, setCustomPath] = React.useState('');
  const [defaultPath, setDefaultPath] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState('');

  React.useEffect(() => {
    if (open) {
      (Backend as any).GetDefaultDatabasePath?.().then((path: string) => {
        setDefaultPath(path);
      }).catch((err: any) => {
        console.error('Failed to get default path:', err);
      });
    }
  }, [open]);

  const handleContinue = async () => {
    setLoading(true);
    setError('');

    try {
      let pathToUse = '';
      
      if (selectedOption === 'default') {
        pathToUse = defaultPath;
      } else if (selectedOption === 'custom' || selectedOption === 'existing') {
        pathToUse = customPath.trim();
      }

      if (!pathToUse) {
        setError('Please provide a valid path');
        setLoading(false);
        return;
      }

      await (Backend as any).SetDatabasePath?.(pathToUse);
      onComplete();
    } catch (err: any) {
      setError(err.message || 'Failed to set database location');
      setLoading(false);
    }
  };

  if (!open) return null;

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      background: 'rgba(0, 0, 0, 0.85)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 9999
    }}>
      <div style={{
        width: '600px',
        maxWidth: '90vw',
        background: 'var(--term-bg)',
        border: '1px solid var(--term-border)',
        borderRadius: '12px',
        padding: '32px',
        boxShadow: '0 8px 32px rgba(0, 0, 0, 0.5)'
      }}>
        {/* Header */}
        <div style={{ marginBottom: '24px', textAlign: 'center' }}>
          <div style={{
            fontFamily: '"Courier New", Courier, monospace',
            fontSize: '24px',
            fontWeight: 800,
            letterSpacing: '0.15em',
            color: 'var(--term-accent)',
            marginBottom: '8px'
          }}>
            WELCOME TO NANITE
          </div>
          <div style={{
            fontSize: '14px',
            color: 'var(--term-dim)',
            fontStyle: 'italic'
          }}>
            Where should we store your todos?
          </div>
        </div>

        {/* Options */}
        <div style={{ marginBottom: '24px' }}>
          {/* Default Location */}
          <label style={{
            display: 'block',
            padding: '16px',
            marginBottom: '12px',
            background: selectedOption === 'default' ? 'var(--term-panel)' : 'transparent',
            border: `1px solid ${selectedOption === 'default' ? 'var(--term-accent)' : 'var(--term-border)'}`,
            borderRadius: '8px',
            cursor: 'pointer',
            transition: 'all 0.2s ease'
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <input
                type="radio"
                name="db-location"
                value="default"
                checked={selectedOption === 'default'}
                onChange={() => setSelectedOption('default')}
                style={{ cursor: 'pointer' }}
              />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: '14px', fontWeight: 600, marginBottom: '4px', color: 'var(--term-fg)' }}>
                  Default Location (Recommended)
                </div>
                <div style={{ fontSize: '11px', color: 'var(--term-dim)', fontFamily: 'monospace', marginTop: '4px' }}>
                  {defaultPath || 'Loading...'}
                </div>
              </div>
            </div>
          </label>

          {/* Custom Location */}
          <label style={{
            display: 'block',
            padding: '16px',
            marginBottom: '12px',
            background: selectedOption === 'custom' ? 'var(--term-panel)' : 'transparent',
            border: `1px solid ${selectedOption === 'custom' ? 'var(--term-accent)' : 'var(--term-border)'}`,
            borderRadius: '8px',
            cursor: 'pointer',
            transition: 'all 0.2s ease'
          }}>
            <div style={{ display: 'flex', alignItems: 'start', gap: '12px' }}>
              <input
                type="radio"
                name="db-location"
                value="custom"
                checked={selectedOption === 'custom'}
                onChange={() => setSelectedOption('custom')}
                style={{ cursor: 'pointer', marginTop: '4px' }}
              />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: '14px', fontWeight: 600, marginBottom: '8px', color: 'var(--term-fg)' }}>
                  Custom Location (e.g., iCloud)
                </div>
                {selectedOption === 'custom' && (
                  <>
                    <input
                      type="text"
                      placeholder="~/Library/Mobile Documents/com~apple~CloudDocs/Nanite"
                      value={customPath}
                      onChange={(e) => setCustomPath(e.target.value)}
                      style={{
                        width: '100%',
                        padding: '8px 12px',
                        background: 'var(--term-bg)',
                        border: '1px solid var(--term-border)',
                        borderRadius: '6px',
                        color: 'var(--term-fg)',
                        fontSize: '12px',
                        fontFamily: 'monospace'
                      }}
                      onClick={(e) => e.stopPropagation()}
                    />
                    <div style={{ fontSize: '10px', color: 'var(--term-dim)', marginTop: '6px' }}>
                      Paste the full directory path where you want to store the database
                    </div>
                  </>
                )}
              </div>
            </div>
          </label>

          {/* Existing Database */}
          <label style={{
            display: 'block',
            padding: '16px',
            background: selectedOption === 'existing' ? 'var(--term-panel)' : 'transparent',
            border: `1px solid ${selectedOption === 'existing' ? 'var(--term-accent)' : 'var(--term-border)'}`,
            borderRadius: '8px',
            cursor: 'pointer',
            transition: 'all 0.2s ease'
          }}>
            <div style={{ display: 'flex', alignItems: 'start', gap: '12px' }}>
              <input
                type="radio"
                name="db-location"
                value="existing"
                checked={selectedOption === 'existing'}
                onChange={() => setSelectedOption('existing')}
                style={{ cursor: 'pointer', marginTop: '4px' }}
              />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: '14px', fontWeight: 600, marginBottom: '8px', color: 'var(--term-fg)' }}>
                  Use Existing Database
                </div>
                {selectedOption === 'existing' && (
                  <>
                    <input
                      type="text"
                      placeholder="Path to directory containing todo.db"
                      value={customPath}
                      onChange={(e) => setCustomPath(e.target.value)}
                      style={{
                        width: '100%',
                        padding: '8px 12px',
                        background: 'var(--term-bg)',
                        border: '1px solid var(--term-border)',
                        borderRadius: '6px',
                        color: 'var(--term-fg)',
                        fontSize: '12px',
                        fontFamily: 'monospace'
                      }}
                      onClick={(e) => e.stopPropagation()}
                    />
                    <div style={{ fontSize: '10px', color: 'var(--term-dim)', marginTop: '6px' }}>
                      Point to the directory that contains your existing todo.db file
                    </div>
                  </>
                )}
              </div>
            </div>
          </label>
        </div>

        {/* Error Message */}
        {error && (
          <div style={{
            padding: '12px',
            marginBottom: '16px',
            background: 'rgba(239, 68, 68, 0.1)',
            border: '1px solid rgba(239, 68, 68, 0.3)',
            borderRadius: '6px',
            color: '#ef4444',
            fontSize: '12px'
          }}>
            {error}
          </div>
        )}

        {/* Continue Button */}
        <button
          onClick={handleContinue}
          disabled={loading || (selectedOption !== 'default' && !customPath.trim())}
          style={{
            width: '100%',
            padding: '12px 24px',
            background: 'var(--term-accent)',
            color: '#000',
            border: 'none',
            borderRadius: '6px',
            fontSize: '14px',
            fontWeight: 600,
            cursor: loading ? 'wait' : 'pointer',
            opacity: loading || (selectedOption !== 'default' && !customPath.trim()) ? 0.5 : 1,
            transition: 'opacity 0.2s ease'
          }}
        >
          {loading ? 'Setting up...' : 'Continue'}
        </button>
      </div>
    </div>
  );
}
