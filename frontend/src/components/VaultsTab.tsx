import * as React from "react";
import * as Backend from "../../wailsjs/go/main/App";
import { config } from "../../wailsjs/go/models";

type Props = {
  vaults: config.Vault[];
  setVaults: React.Dispatch<React.SetStateAction<config.Vault[]>>;
  activeVaultId: string;
  setActiveVaultId: (id: string) => void;
  setConfirmDeleteVaultId: (id: string) => void;
};

export default function VaultsTab({ vaults, setVaults, activeVaultId, setActiveVaultId, setConfirmDeleteVaultId }: Props) {
  const [editingVaultId, setEditingVaultId] = React.useState<string | null>(null);
  const [editingVaultName, setEditingVaultName] = React.useState('');
  const [newVaultName, setNewVaultName] = React.useState('');
  const [newVaultDir, setNewVaultDir] = React.useState('');
  const [showNewVaultForm, setShowNewVaultForm] = React.useState(false);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
        <h3 style={{ fontSize: '14px', fontWeight: 600 }}>Vaults</h3>
        <button
          className="badge success"
          onClick={() => setShowNewVaultForm(v => !v)}
          style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
        >
          {showNewVaultForm ? 'Cancel' : '+ New Vault'}
        </button>
      </div>
      <div style={{ fontSize: '11px', marginBottom: '16px' }} className="text-dim">
        Each vault is a separate SQLite database. Use ⌘⇧V to quickly switch between them.
      </div>

      {/* New vault form */}
      {showNewVaultForm && (
        <div style={{
          padding: '12px',
          background: 'var(--term-panel)',
          border: '1px solid var(--term-border)',
          borderRadius: '6px',
          marginBottom: '16px',
          display: 'flex',
          flexDirection: 'column',
          gap: '8px'
        }}>
          <input
            type="text"
            placeholder="Vault name (e.g. Work, RPG World, Research)"
            value={newVaultName}
            onChange={e => setNewVaultName(e.target.value)}
            style={{
              padding: '6px 10px',
              background: 'var(--term-bg)',
              border: '1px solid var(--term-border)',
              borderRadius: '4px',
              color: 'var(--term-fg)',
              fontSize: '13px'
            }}
          />
          <input
            type="text"
            placeholder="Directory path (e.g. /Users/you/Vaults/Work)"
            value={newVaultDir}
            onChange={e => setNewVaultDir(e.target.value)}
            style={{
              padding: '6px 10px',
              background: 'var(--term-bg)',
              border: '1px solid var(--term-border)',
              borderRadius: '4px',
              color: 'var(--term-fg)',
              fontSize: '13px',
              fontFamily: 'monospace'
            }}
          />
          <button
            className="badge success"
            disabled={!newVaultName.trim() || !newVaultDir.trim()}
            onClick={async () => {
              try {
                const created = await Backend.CreateVault(newVaultName.trim(), newVaultDir.trim());
                if (created) setVaults(prev => [...prev, created]);
                setNewVaultName('');
                setNewVaultDir('');
                setShowNewVaultForm(false);
              } catch (err: any) {
                alert(`Failed to create vault: ${err.message || err}`);
              }
            }}
            style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', alignSelf: 'flex-start' }}
          >
            Create Vault
          </button>
        </div>
      )}

      {/* Vault list */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {vaults.map(vault => {
          const isActive = vault.id === activeVaultId;
          const isEditing = editingVaultId === vault.id;
          return (
            <div
              key={vault.id}
              style={{
                padding: '10px 12px',
                background: isActive ? 'rgba(var(--term-accent-rgb, 122, 162, 247), 0.06)' : 'var(--term-panel)',
                border: `1px solid ${isActive ? 'var(--term-accent)' : 'var(--term-border)'}`,
                borderRadius: '6px',
                display: 'flex',
                alignItems: 'center',
                gap: '8px'
              }}
            >
              {/* Name (editable inline) */}
              <div style={{ flex: 1 }}>
                {isEditing ? (
                  <input
                    autoFocus
                    value={editingVaultName}
                    onChange={e => setEditingVaultName(e.target.value)}
                    onKeyDown={async e => {
                      if (e.key === 'Enter') {
                        try {
                          await Backend.RenameVault(vault.id, editingVaultName.trim());
                          setVaults(prev => prev.map(v => v.id === vault.id ? { ...v, name: editingVaultName.trim() } : v));
                        } catch (err: any) {
                          alert(`Failed to rename: ${err.message || err}`);
                        }
                        setEditingVaultId(null);
                      } else if (e.key === 'Escape') {
                        setEditingVaultId(null);
                      }
                    }}
                    onBlur={() => setEditingVaultId(null)}
                    style={{
                      padding: '2px 6px',
                      background: 'var(--term-bg)',
                      border: '1px solid var(--term-accent)',
                      borderRadius: '3px',
                      color: 'var(--term-fg)',
                      fontSize: '13px',
                      width: '100%'
                    }}
                  />
                ) : (
                  <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                      <span style={{ fontSize: '13px', fontWeight: isActive ? 600 : 400 }}>
                        {vault.name}
                      </span>
                      {isActive && (
                        <span className="badge success" style={{ fontSize: '10px', padding: '2px 5px', borderRadius: '3px' }}>active</span>
                      )}
                    </div>
                    <div style={{ fontSize: '11px', color: 'var(--term-dim)', fontFamily: 'monospace', marginTop: '2px' }}>
                      {vault.path}
                    </div>
                  </div>
                )}
              </div>

              {/* Actions */}
              <div style={{ display: 'flex', gap: '4px', flexShrink: 0 }}>
                {!isActive && (
                  <button
                    className="badge info"
                    onClick={async () => {
                      try {
                        await Backend.SwitchVault(vault.id);
                        setActiveVaultId(vault.id);
                      } catch (err: any) {
                        alert(`Failed to switch: ${err.message || err}`);
                      }
                    }}
                    style={{ padding: '4px 8px', fontSize: '11px', borderRadius: '4px' }}
                  >
                    Switch To
                  </button>
                )}
                <button
                  className="badge"
                  onClick={() => {
                    setEditingVaultId(vault.id);
                    setEditingVaultName(vault.name);
                  }}
                  style={{ padding: '4px 8px', fontSize: '11px', borderRadius: '4px' }}
                >
                  Rename
                </button>
                {vaults.length > 1 && !isActive && (
                  <button
                    className="badge warn"
                    onClick={() => setConfirmDeleteVaultId(vault.id)}
                    style={{ padding: '4px 8px', fontSize: '11px', borderRadius: '4px' }}
                  >
                    Delete
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>

      <div style={{ marginTop: '16px', fontSize: '11px', lineHeight: '1.6' }} className="text-dim">
        Deleting a vault removes it from the registry only — the database files on disk are preserved.
      </div>
    </div>
  );
}
