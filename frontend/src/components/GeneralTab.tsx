import * as Backend from "../../wailsjs/go/main/App";
import { Check } from "lucide-react";
import { Settings } from "./SettingsContext";
import { inputStyle } from "./settingsShared";

type Props = {
  local: Settings;
  setLocal: (s: Settings) => void;
  databasePath: string;
  changingDbPath: boolean;
  setChangingDbPath: (v: boolean) => void;
  setDatabasePath: (v: string) => void;
  setShowRestartPrompt: (v: boolean) => void;
};

export default function GeneralTab({
  local,
  setLocal,
  databasePath,
  changingDbPath,
  setChangingDbPath,
  setDatabasePath,
  setShowRestartPrompt,
}: Props) {
  return (
    <div>
      <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Display Options</h3>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 0', cursor: 'pointer' }}
        onClick={() => setLocal({ ...local, showCompleted: !local.showCompleted })}
      >
        <div style={{
          width: '18px',
          height: '18px',
          border: '1px solid var(--term-border)',
          borderRadius: '4px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: local.showCompleted ? 'var(--term-accent)' : 'transparent',
          transition: 'all 0.2s ease'
        }}>
          {local.showCompleted && <Check size={14} style={{ color: '#000' }} />}
        </div>
        <label style={{ cursor: 'pointer', fontSize: '13px' }}>
          Show completed items by default
        </label>
      </div>
      <div style={{ marginTop: '12px', fontSize: '11px' }} className="text-dim">
        When enabled, completed todos will appear in the section lists (grayed out).
      </div>

      <div style={{ marginTop: '20px' }}>
        <label style={{ fontSize: '13px', display: 'block', marginBottom: '8px' }}>
          Default View
        </label>
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, defaultView: 'scope' })}>
            <div style={{
              width: '16px',
              height: '16px',
              borderRadius: '50%',
              border: '1px solid var(--term-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: local.defaultView === 'scope' ? 'var(--term-accent)' : 'transparent'
            }}>
              {local.defaultView === 'scope' && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
            </div>
            <label style={{ cursor: 'pointer', fontSize: '12px' }}>Scope</label>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, defaultView: 'date' })}>
            <div style={{
              width: '16px',
              height: '16px',
              borderRadius: '50%',
              border: '1px solid var(--term-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: local.defaultView === 'date' ? 'var(--term-accent)' : 'transparent'
            }}>
              {local.defaultView === 'date' && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
            </div>
            <label style={{ cursor: 'pointer', fontSize: '12px' }}>Date</label>
          </div>
        </div>
        <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
          Scope organizes by Now/Soon/Anytime priority. Date groups by due date. Choose which view to show on app load.
        </div>
      </div>

      <div style={{ marginTop: '20px' }}>
        <label style={{ fontSize: '13px', display: 'block', marginBottom: '8px' }}>
          Default Input Mode
        </label>
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, defaultInputMode: 'search' })}>
            <div style={{
              width: '16px',
              height: '16px',
              borderRadius: '50%',
              border: '1px solid var(--term-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: local.defaultInputMode === 'search' ? 'var(--term-accent)' : 'transparent'
            }}>
              {local.defaultInputMode === 'search' && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
            </div>
            <label style={{ cursor: 'pointer', fontSize: '12px' }}>Search</label>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, defaultInputMode: 'add' })}>
            <div style={{
              width: '16px',
              height: '16px',
              borderRadius: '50%',
              border: '1px solid var(--term-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: local.defaultInputMode === 'add' ? 'var(--term-accent)' : 'transparent'
            }}>
              {local.defaultInputMode === 'add' && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
            </div>
            <label style={{ cursor: 'pointer', fontSize: '12px' }}>Add</label>
          </div>
        </div>
        <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
          Search mode filters todos as you type. Add mode creates a new todo when you press Enter. Toggle with Tab key anytime.
        </div>
      </div>

      <div style={{ marginTop: '20px' }}>
        <label style={{ fontSize: '13px', display: 'block', marginBottom: '8px' }}>
          Close &amp; Save Behavior
        </label>
        <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
          {(['ask', 'always', 'never'] as const).map(opt => (
            <div key={opt} style={{ display: 'flex', alignItems: 'center', gap: '6px', cursor: 'pointer' }} onClick={() => setLocal({ ...local, closeBehavior: opt })}>
              <div style={{
                width: '16px',
                height: '16px',
                borderRadius: '50%',
                border: '1px solid var(--term-border)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: (local.closeBehavior ?? 'ask') === opt ? 'var(--term-accent)' : 'transparent'
              }}>
                {(local.closeBehavior ?? 'ask') === opt && <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#000' }} />}
              </div>
              <label style={{ cursor: 'pointer', fontSize: '12px' }}>
                {opt === 'ask' ? 'Ask each time' : opt === 'always' ? 'Always save' : 'Never save'}
              </label>
            </div>
          ))}
        </div>
        <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
          When closing a todo or note with unsaved changes: save automatically, discard, or be prompted.
        </div>
      </div>

      <div style={{ marginTop: '20px' }}>
        <label style={{ fontSize: '13px', display: 'block', marginBottom: '8px' }}>
          Default Tags
        </label>
        <input
          type="text"
          style={inputStyle}
          value={(local.defaultTags || []).join(', ')}
          onChange={(e) => {
            const tags = e.target.value.split(',').map(t => t.trim().replace(/^#/, '')).filter(t => t);
            setLocal({ ...local, defaultTags: tags });
          }}
          placeholder="e.g., inbox, review (comma-separated)"
        />
        <div style={{ marginTop: '8px', fontSize: '11px' }} className="text-dim">
          These tags will be automatically added to new todos when no tags or projects are specified
        </div>
      </div>

      <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px', marginTop: '24px' }}>Database Location</h3>
      <div style={{ marginBottom: '8px' }}>
        <label style={{ fontSize: '12px', display: 'block', marginBottom: '6px' }} className="text-dim">
          Current Database Path
        </label>
        {!changingDbPath ? (
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            <div style={{
              flex: 1,
              padding: '8px 12px',
              background: 'var(--term-panel)',
              border: '1px solid var(--term-border)',
              borderRadius: '6px',
              fontFamily: 'monospace',
              fontSize: '12px',
              color: 'var(--term-fg)',
              opacity: 0.8
            }}>
              {databasePath || 'Loading...'}
            </div>
            <button
              type="button"
              className="badge info"
              onClick={() => setChangingDbPath(true)}
              style={{ padding: '8px 12px', fontSize: '12px', borderRadius: '6px', whiteSpace: 'nowrap', height: '44px' }}
            >
              Change Location
            </button>
          </div>
        ) : (
          <div>
            <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
              <input
                type="text"
                style={{
                  flex: 1,
                  padding: '8px 12px',
                  background: 'var(--term-bg)',
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  color: 'var(--term-fg)',
                  fontSize: '12px',
                  fontFamily: 'monospace'
                }}
                placeholder="Paste new database directory path..."
                defaultValue={databasePath}
                id="new-db-path-input"
              />
            </div>
            <div style={{ display: 'flex', gap: '8px' }}>
              <button
                type="button"
                className="badge success"
                onClick={async () => {
                  const input = document.getElementById('new-db-path-input') as HTMLInputElement;
                  const newPath = input?.value?.trim();
                  if (newPath) {
                    try {
                      await (Backend as any).SetDatabasePath(newPath);
                      setDatabasePath(newPath);
                      setChangingDbPath(false);
                      setShowRestartPrompt(true);
                    } catch (err: any) {
                      alert(`Error: ${err.message || err}`);
                    }
                  }
                }}
                style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
              >
                Save
              </button>
              <button
                type="button"
                className="badge"
                onClick={() => setChangingDbPath(false)}
                style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
              >
                Cancel
              </button>
            </div>
            <div style={{ fontSize: '11px', marginTop: '8px', color: 'var(--term-dim)' }}>
              Paste the full path to the directory where you want to store the database.
              <br />
              For iCloud: <code style={{ fontSize: '10px' }}>~/Library/Mobile Documents/com~apple~CloudDocs/Planck</code>
            </div>
          </div>
        )}
      </div>

      <details style={{ marginTop: '12px', marginBottom: '24px' }}>
        <summary style={{
          fontSize: '12px',
          cursor: 'pointer',
          padding: '8px',
          background: 'var(--term-panel)',
          borderRadius: '4px',
          border: '1px solid var(--term-border)',
          marginBottom: '16px'
        }}>
          💡 Cloud Sync Setup (Dropbox, iCloud, Google Drive)
        </summary>
        <div style={{
          fontSize: '11px',
          padding: '12px',
          background: 'var(--term-panel)',
          borderRadius: '4px',
          border: '1px solid var(--term-border)',
          lineHeight: '1.6'
        }} className="text-dim">
          <p style={{ marginBottom: '8px' }}>To sync your todos across multiple computers:</p>
          <ol style={{ marginLeft: '16px', marginBottom: '8px' }}>
            <li>Quit the app completely</li>
            <li>Move the <code>data</code> folder to your cloud sync location</li>
            <li>Create a symbolic link pointing to the new location</li>
            <li>Restart the app</li>
          </ol>
          <p style={{ marginTop: '8px', padding: '6px', background: 'var(--term-bg)', borderRadius: '3px', fontFamily: 'monospace', fontSize: '10px' }}>
            Example: ln -s ~/Dropbox/planck-data ./data
          </p>
          <p style={{ marginTop: '8px', color: 'var(--term-warn)' }}>
            ⚠️ Never open the app on multiple computers simultaneously before sync completes!
          </p>
        </div>
      </details>
    </div>
  );
}
