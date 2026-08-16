import { config } from "../../wailsjs/go/models";
import { Check } from "lucide-react";
import { inputStyle } from "./settingsShared";

export type ChatConfigState = {
  enabled: boolean;
  apiKey: string;
  model: string;
  dryRun: boolean;
  vaultCaps: Record<string, { read: boolean; write: boolean; delete: boolean; directCreate: boolean }>;
};

type Props = {
  chatConfig: ChatConfigState | null;
  setChatConfig: (c: ChatConfigState) => void;
  chatApiKeyVisible: boolean;
  setChatApiKeyVisible: (v: boolean | ((prev: boolean) => boolean)) => void;
  vaults: config.Vault[];
  activeVaultId: string;
};

export default function ChatTab({ chatConfig, setChatConfig, chatApiKeyVisible, setChatApiKeyVisible, vaults, activeVaultId }: Props) {
  return (
    <div>
      <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Chat Configuration</h3>
      <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
        Configure the Vault Chat addon (⌘⇧C). Requires a Claude API key.
      </div>
      {!chatConfig ? (
        <div style={{ fontSize: '12px' }} className="text-dim">Loading…</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

          {/* Enable Chat */}
          <div>
            <div
              style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}
              onClick={() => setChatConfig({ ...chatConfig, enabled: !chatConfig.enabled })}
            >
              <div style={{
                width: '18px', height: '18px',
                border: '1px solid var(--term-border)', borderRadius: '4px',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                background: chatConfig.enabled ? 'var(--term-accent)' : 'transparent',
                transition: 'all 0.2s ease', flexShrink: 0
              }}>
                {chatConfig.enabled && <Check size={14} style={{ color: '#000' }} />}
              </div>
              <label style={{ cursor: 'pointer', fontSize: '13px' }}>Enable Vault Chat</label>
            </div>
            <div style={{ marginTop: '6px', fontSize: '11px' }} className="text-dim">
              Enables the chat panel for natural language search and guided item creation.
            </div>
          </div>

          {/* API Key */}
          <div>
            <label style={{ fontSize: '13px', display: 'block', marginBottom: '6px' }}>Claude API Key</label>
            <div style={{ display: 'flex', gap: '8px' }}>
              <input
                type={chatApiKeyVisible ? 'text' : 'password'}
                value={chatConfig.apiKey ?? ''}
                onChange={e => setChatConfig({ ...chatConfig, apiKey: e.target.value })}
                placeholder="sk-ant-..."
                style={{ ...inputStyle, flex: 1, fontFamily: 'monospace' }}
                autoComplete="off"
              />
              <button
                type="button"
                className="badge"
                onClick={() => setChatApiKeyVisible(v => !v)}
                style={{ padding: '6px 10px', fontSize: '12px', borderRadius: '6px', whiteSpace: 'nowrap' }}
              >
                {chatApiKeyVisible ? 'Hide' : 'Show'}
              </button>
            </div>
            <div style={{ marginTop: '6px', fontSize: '11px' }} className="text-dim">
              Get your API key from console.anthropic.com. Stored locally in app config.
            </div>
          </div>

          {/* Model */}
          <div>
            <label style={{ fontSize: '13px', display: 'block', marginBottom: '6px' }}>Model</label>
            <select
              value={chatConfig.model || 'claude-haiku-4-5-20251001'}
              onChange={e => setChatConfig({ ...chatConfig, model: e.target.value })}
              style={{ ...inputStyle, cursor: 'pointer' }}
            >
              <option value="claude-haiku-4-5-20251001">Claude Haiku (fast, efficient)</option>
              <option value="claude-sonnet-4-6">Claude Sonnet (balanced)</option>
            </select>
            <div style={{ marginTop: '6px', fontSize: '11px' }} className="text-dim">
              Haiku is recommended for quick vault queries. Sonnet for complex reasoning tasks.
            </div>
          </div>

          {/* Dry Run */}
          <div>
            <div
              style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}
              onClick={() => setChatConfig({ ...chatConfig, dryRun: !chatConfig.dryRun })}
            >
              <div style={{
                width: '18px', height: '18px',
                border: '1px solid var(--term-border)', borderRadius: '4px',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                background: chatConfig.dryRun ? 'var(--term-accent)' : 'transparent',
                transition: 'all 0.2s ease', flexShrink: 0
              }}>
                {chatConfig.dryRun && <Check size={14} style={{ color: '#000' }} />}
              </div>
              <label style={{ cursor: 'pointer', fontSize: '13px' }}>Dry-run mode</label>
            </div>
            <div style={{ marginTop: '6px', fontSize: '11px' }} className="text-dim">
              When enabled, all proposed actions are shown for review but never executed against your vault.
            </div>
          </div>

          {/* Vault Capabilities */}
          {vaults.length > 0 && (
            <div>
              <label style={{ fontSize: '13px', display: 'block', marginBottom: '6px' }}>Vault Permissions</label>
              <div style={{ fontSize: '11px', marginBottom: '10px' }} className="text-dim">
                Control what the chat agent is allowed to do in each vault.
              </div>
              <div style={{ border: '1px solid var(--term-border)', borderRadius: '6px', overflow: 'hidden' }}>
                {/* Header row */}
                <div style={{
                  display: 'grid', gridTemplateColumns: '1fr 60px 60px 60px 90px',
                  padding: '6px 12px',
                  background: 'var(--term-panel)',
                  borderBottom: '1px solid var(--term-border)',
                  fontSize: '11px', fontWeight: 600
                }} className="text-dim">
                  <span>Vault</span>
                  <span style={{ textAlign: 'center' }}>Read</span>
                  <span style={{ textAlign: 'center' }}>Write</span>
                  <span style={{ textAlign: 'center' }}>Delete</span>
                  <span style={{ textAlign: 'center' }}>Direct Create</span>
                </div>
                {/* Vault rows */}
                {vaults.map((vault, idx) => {
                  const caps = chatConfig.vaultCaps?.[vault.id] ?? { read: false, write: false, delete: false, directCreate: false };
                  const updateCap = (field: 'read' | 'write' | 'delete' | 'directCreate', value: boolean) => {
                    const newCaps = { ...(chatConfig.vaultCaps ?? {}), [vault.id]: { ...caps, [field]: value } };
                    setChatConfig({ ...chatConfig, vaultCaps: newCaps });
                  };
                  const isActive = vault.id === activeVaultId;
                  return (
                    <div
                      key={vault.id}
                      style={{
                        display: 'grid', gridTemplateColumns: '1fr 60px 60px 60px 90px',
                        padding: '8px 12px',
                        background: isActive ? 'rgba(var(--term-accent-rgb, 122, 162, 247), 0.04)' : 'var(--term-bg)',
                        borderBottom: idx < vaults.length - 1 ? '1px solid var(--term-border)' : 'none',
                        alignItems: 'center'
                      }}
                    >
                      <span style={{ fontSize: '12px', fontWeight: isActive ? 600 : 400 }}>
                        {vault.name}
                        {isActive && <span className="badge success" style={{ fontSize: '9px', padding: '1px 4px', borderRadius: '3px', marginLeft: '6px' }}>active</span>}
                      </span>
                      {(['read', 'write', 'delete', 'directCreate'] as const).map(field => (
                        <div key={field} style={{ display: 'flex', justifyContent: 'center' }}>
                          <button
                            type="button"
                            onClick={() => updateCap(field, !caps[field])}
                            style={{
                              width: '18px', height: '18px',
                              border: '1px solid var(--term-border)', borderRadius: '4px',
                              display: 'flex', alignItems: 'center', justifyContent: 'center',
                              background: caps[field] ? 'var(--term-accent)' : 'transparent',
                              cursor: 'pointer', transition: 'all 0.2s ease', padding: 0
                            }}
                          >
                            {caps[field] && <Check size={12} style={{ color: '#000' }} />}
                          </button>
                        </div>
                      ))}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

        </div>
      )}
    </div>
  );
}
