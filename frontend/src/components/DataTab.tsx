import * as Backend from "../../wailsjs/go/main/App";
import { Check } from "lucide-react";

export type ApiConfigState = { enabled: boolean; port: number; api_key: string };

type Props = {
  apiConfig: ApiConfigState | null;
  setApiConfig: (c: ApiConfigState) => void;
  apiKeyCopied: boolean;
  setApiKeyCopied: (v: boolean) => void;
};

export default function DataTab({ apiConfig, setApiConfig, apiKeyCopied, setApiKeyCopied }: Props) {
  return (
    <div>
      <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Import/Export</h3>
      <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
        Import and export your todos in todo.txt format.
      </div>

      <div style={{ display: 'flex', gap: '12px' }}>
        <button
          className="badge info"
          onClick={async () => {
            try {
              const content = await Backend.ExportTodoTxt();
              const blob = new Blob([content], { type: 'text/plain' });
              const url = URL.createObjectURL(blob);
              const a = document.createElement('a');
              a.href = url;
              a.download = `todos-${new Date().toISOString().slice(0, 10)}.txt`;
              a.click();
              URL.revokeObjectURL(url);
            } catch (err) {
              console.error('Export failed:', err);
              alert('Export failed: ' + err);
            }
          }}
          style={{ flex: 1, padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
        >
          Export to todo.txt
        </button>

        <label
          htmlFor="import-file"
          className="badge success"
          style={{ flex: 1, padding: '8px 12px', fontSize: '13px', borderRadius: '6px', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
        >
          Import from todo.txt
        </label>
        <input
          id="import-file"
          type="file"
          accept=".txt"
          style={{ display: 'none' }}
          onChange={async (e) => {
            const file = e.target.files?.[0];
            if (!file) return;
            try {
              const content = await file.text();
              await Backend.ImportTodoTxt(content);
              alert('Import successful!');
              window.location.reload();
            } catch (err) {
              console.error('Import failed:', err);
              alert('Import failed: ' + err);
            }
          }}
        />
      </div>

      {/* API Access */}
      <div style={{ marginTop: '28px', paddingTop: '20px', borderTop: '1px solid var(--term-border)' }}>
        <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>API Access</h3>
        <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
          Enable a local HTTP API so external agents and scripts can push items to your inbox or query your data.
        </div>

        {apiConfig ? (
          <div>
            {/* Enable toggle */}
            <div
              style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px', cursor: 'pointer' }}
              onClick={async () => {
                const newEnabled = !apiConfig.enabled;
                try {
                  await (Backend as any).SetAPIEnabled(newEnabled);
                  setApiConfig({ ...apiConfig, enabled: newEnabled });
                } catch (err: any) {
                  alert(`Error: ${err.message || err}`);
                }
              }}
            >
              <div style={{
                width: '18px',
                height: '18px',
                border: '1px solid var(--term-border)',
                borderRadius: '4px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: apiConfig.enabled ? 'var(--term-accent)' : 'transparent',
                transition: 'all 0.2s ease',
                flexShrink: 0,
              }}>
                {apiConfig.enabled && <Check size={14} style={{ color: '#000' }} />}
              </div>
              <label style={{ cursor: 'pointer', fontSize: '13px' }}>
                Enable local HTTP API
              </label>
              {apiConfig.enabled && (
                <span style={{ fontSize: '11px', color: 'var(--term-accent)', marginLeft: '4px' }}>
                  — listening on port {apiConfig.port}
                </span>
              )}
            </div>

            {/* API Key */}
            <div style={{ marginBottom: '8px' }}>
              <label style={{ fontSize: '12px', display: 'block', marginBottom: '6px' }} className="text-dim">
                API Key
              </label>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                <div style={{
                  flex: 1,
                  padding: '8px 12px',
                  background: 'var(--term-panel)',
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  fontFamily: 'monospace',
                  fontSize: '11px',
                  color: 'var(--term-fg)',
                  overflowX: 'auto',
                  whiteSpace: 'nowrap',
                  userSelect: 'all',
                  opacity: 0.9,
                }}>
                  {apiConfig.api_key}
                </div>
                <button
                  type="button"
                  className="badge"
                  onClick={() => {
                    navigator.clipboard.writeText(apiConfig.api_key);
                    setApiKeyCopied(true);
                    setTimeout(() => setApiKeyCopied(false), 2000);
                  }}
                  style={{ padding: '8px 12px', fontSize: '12px', borderRadius: '6px', whiteSpace: 'nowrap' }}
                >
                  {apiKeyCopied ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>

            <div style={{ fontSize: '11px', marginTop: '10px', lineHeight: '1.6' }} className="text-dim">
              Send <code style={{ fontFamily: 'monospace', background: 'var(--term-panel)', padding: '1px 4px', borderRadius: '3px' }}>X-API-Key</code> header to authenticate.
              {' '}Use <code style={{ fontFamily: 'monospace', background: 'var(--term-panel)', padding: '1px 4px', borderRadius: '3px' }}>X-Agent-Source</code> to identify your agent.
              <br />
              Endpoint: <code style={{ fontFamily: 'monospace', background: 'var(--term-panel)', padding: '1px 4px', borderRadius: '3px' }}>POST http://127.0.0.1:{apiConfig.port}/api/v1/inbox</code>
            </div>
          </div>
        ) : (
          <div style={{ fontSize: '12px' }} className="text-dim">Loading API config…</div>
        )}
      </div>
    </div>
  );
}
