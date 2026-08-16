import * as React from "react";
import { useTermTheme } from "@/theme/ThemeProvider";
import { themePresets } from "@/theme/theme";
import CustomScrollbar from "@/components/CustomScrollbar";
import ConfirmDialog from "@/components/ConfirmDialog";
import * as Backend from "../../wailsjs/go/main/App";
import { config } from "../../wailsjs/go/models";
import { useSettings, Settings } from "./SettingsContext";
import GeneralTab from "./GeneralTab";
import TabsTab from "./TabsTab";
import VaultsTab from "./VaultsTab";
import ChatTab, { ChatConfigState } from "./ChatTab";
import DataTab, { ApiConfigState } from "./DataTab";
import ThemeTab from "./ThemeTab";

type SettingsTab = 'general' | 'tabs' | 'data' | 'vaults' | 'chat'; // 'theme' hidden until Tailwind migration

type Props = {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  initialTab?: SettingsTab;
};

export default function SettingsModal({ open, onOpenChange, initialTab }: Props) {
  const { settings, setSettings } = useSettings();
  const [local, setLocal] = React.useState<Settings>(settings);
  const [activeTab, setActiveTab] = React.useState<SettingsTab>('general');
  const { theme, setTheme } = useTermTheme();
  const [localTheme, setLocalTheme] = React.useState(() => ({ ...themePresets['default']!, ...theme }));
  const [customTheme, setCustomTheme] = React.useState(() => ({ ...themePresets['default']!, ...theme }));
  const [selectedPreset, setSelectedPreset] = React.useState<string>('custom');
  const [databasePath, setDatabasePath] = React.useState<string>('');
  const [changingDbPath, setChangingDbPath] = React.useState(false);
  const [showRestartPrompt, setShowRestartPrompt] = React.useState(false);
  const [apiConfig, setApiConfig] = React.useState<ApiConfigState | null>(null);
  const [apiKeyCopied, setApiKeyCopied] = React.useState(false);
  const [vaults, setVaults] = React.useState<config.Vault[]>([]);
  const [activeVaultId, setActiveVaultId] = React.useState<string>('');
  const [confirmDeleteVaultId, setConfirmDeleteVaultId] = React.useState<string | null>(null);
  const [chatConfig, setChatConfig] = React.useState<ChatConfigState | null>(null);
  const [chatApiKeyVisible, setChatApiKeyVisible] = React.useState(false);

  React.useEffect(() => {
    if (open) {
      setLocal(settings);
      setActiveTab(initialTab || 'general');
      // Ensure theme has all properties by merging with default
      const fullTheme = { ...themePresets['default']!, ...theme };
      console.log('[Settings] Full theme properties:', Object.keys(fullTheme));
      console.log('[Settings] Scrollbar properties:', {
        scrollbarBg: fullTheme.scrollbarBg,
        scrollbarThumb: fullTheme.scrollbarThumb,
        scrollbarThumbHover: fullTheme.scrollbarThumbHover
      });
      setLocalTheme(fullTheme);
      // Check if current theme matches a preset
      const matchingPreset = Object.entries(themePresets).find(([_, preset]) =>
        JSON.stringify(preset) === JSON.stringify(theme)
      );
      setSelectedPreset(matchingPreset ? matchingPreset[0] : 'custom');
      // If it's custom, save it so we don't lose it
      if (!matchingPreset) {
        setCustomTheme(fullTheme);
      }

      // Load database path
      (Backend as any).GetDatabasePath?.().then((path: string) => {
        setDatabasePath(path);
      }).catch((err: any) => {
        console.error('Failed to load database path:', err);
        setDatabasePath('./data'); // fallback
      });

      // Load API config
      (Backend as any).GetAPIConfig?.().then((cfg: { enabled: boolean; port: number; api_key: string }) => {
        setApiConfig(cfg);
      }).catch((err: any) => {
        console.error('Failed to load API config:', err);
      });

      // Load vaults
      Promise.all([
        Backend.GetVaults(),
        Backend.GetActiveVault(),
      ]).then(([allVaults, active]) => {
        setVaults(allVaults ?? []);
        setActiveVaultId(active?.id ?? '');
      }).catch((err: any) => {
        console.error('Failed to load vaults:', err);
      });

      // Load chat config
      Backend.GetChatConfig().then(cfg => {
        setChatConfig({ enabled: cfg.enabled, apiKey: cfg.apiKey, model: cfg.model, dryRun: cfg.dryRun, vaultCaps: cfg.vaultCaps ?? {} });
      }).catch((err: any) => {
        console.error('Failed to load chat config:', err);
      });
    }
  }, [open, settings, theme]);

  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onOpenChange(false);
      }
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, [open, onOpenChange]);

  if (!open) return null;

  const modalStyle = {
    position: 'fixed' as const,
    inset: 0,
    background: 'rgba(0,0,0,0.7)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 50
  };

  const handleSave = () => {
    setSettings(local);
    // Theme tab hidden until Tailwind migration; keep setTheme call guarded
    // if (activeTab === 'theme') { setTheme(localTheme as TermTheme); }
    if (chatConfig) {
      Backend.SetChatConfig(chatConfig as config.ChatConfig).catch((err: any) => {
        console.error('Failed to save chat config:', err);
      });
    }
    onOpenChange(false);
  };

  return (
    <>
      <div style={modalStyle} onClick={() => onOpenChange(false)}>
        <div
          className="terminal-card"
          style={{
            width: '700px',
            maxWidth: '95vw',
            height: '650px',
            display: 'flex',
            flexDirection: 'column'
          }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Fixed Header */}
          <div style={{ padding: '20px 20px 0 20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
              <div style={{ fontWeight: 600, fontSize: '16px' }}>Settings</div>
              <button className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Close</button>
            </div>

            {/* Tab Navigation */}
            <div style={{ display: 'flex', gap: '0px', marginBottom: '20px', paddingBottom: '12px', borderBottom: '1px solid var(--term-border)' }}>
              <button
                onClick={() => setActiveTab('general')}
                className={`badge ${activeTab === 'general' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '4px 0 0 4px',
                  border: 'none',
                  opacity: activeTab === 'general' ? 1 : 0.8
                }}
              >
                General
              </button>
              <button
                onClick={() => setActiveTab('tabs')}
                className={`badge ${activeTab === 'tabs' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'tabs' ? 1 : 0.8
                }}
              >
                Scope Tabs
              </button>
              <button
                onClick={() => setActiveTab('data')}
                className={`badge ${activeTab === 'data' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'data' ? 1 : 0.8
                }}
              >
                Data
              </button>
              <button
                onClick={() => setActiveTab('vaults')}
                className={`badge ${activeTab === 'vaults' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'vaults' ? 1 : 0.8
                }}
              >
                Vaults
              </button>
              <button
                onClick={() => setActiveTab('chat')}
                className={`badge ${activeTab === 'chat' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0 4px 4px 0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'chat' ? 1 : 0.8
                }}
              >
                Chat
              </button>
              {/* HIDDEN until Tailwind migration:
              <button
                onClick={() => setActiveTab('theme')}
                className={`badge ${activeTab === 'theme' ? 'success' : ''}`}
                style={{
                  padding: '4px 8px',
                  fontSize: '11px',
                  borderRadius: '0 4px 4px 0',
                  border: 'none',
                  borderLeft: '1px solid var(--term-border)',
                  opacity: activeTab === 'theme' ? 1 : 0.8
                }}
              >
                Theme
              </button>
              */}
            </div>
          </div>

          {/* Scrollable Body */}
          <CustomScrollbar style={{
            flex: 1,
            minHeight: 0
          }}>
            <div style={{ padding: '0 20px' }}>
              {/* General Settings */}
              {activeTab === 'general' && (
                <GeneralTab
                  local={local}
                  setLocal={setLocal}
                  databasePath={databasePath}
                  changingDbPath={changingDbPath}
                  setChangingDbPath={setChangingDbPath}
                  setDatabasePath={setDatabasePath}
                  setShowRestartPrompt={setShowRestartPrompt}
                />
              )}

              {/* Scope Tabs Settings */}
              {activeTab === 'tabs' && (
                <TabsTab local={local} setLocal={setLocal} />
              )}

              {/* Vaults Settings */}
              {activeTab === 'vaults' && (
                <VaultsTab
                  vaults={vaults}
                  setVaults={setVaults}
                  activeVaultId={activeVaultId}
                  setActiveVaultId={setActiveVaultId}
                  setConfirmDeleteVaultId={setConfirmDeleteVaultId}
                />
              )}

              {/* Chat Settings */}
              {activeTab === 'chat' && (
                <ChatTab
                  chatConfig={chatConfig}
                  setChatConfig={setChatConfig}
                  chatApiKeyVisible={chatApiKeyVisible}
                  setChatApiKeyVisible={setChatApiKeyVisible}
                  vaults={vaults}
                  activeVaultId={activeVaultId}
                />
              )}

              {/* Theme Settings — HIDDEN until Tailwind migration */}
              {false && (
                <ThemeTab
                  setTheme={setTheme}
                  localTheme={localTheme}
                  setLocalTheme={setLocalTheme}
                  customTheme={customTheme}
                  setCustomTheme={setCustomTheme}
                  selectedPreset={selectedPreset}
                  setSelectedPreset={setSelectedPreset}
                />
              )}

              {/* Import/Export + API Settings */}
              {activeTab === 'data' && (
                <DataTab
                  apiConfig={apiConfig}
                  setApiConfig={setApiConfig}
                  apiKeyCopied={apiKeyCopied}
                  setApiKeyCopied={setApiKeyCopied}
                />
              )}
            </div>
          </CustomScrollbar>

          {/* Fixed Footer */}
          <div style={{
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '8px',
            padding: '16px 20px 20px 20px',
            borderTop: '1px solid var(--term-border)'
          }}>
            <button className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Cancel</button>
            <button className="badge success" onClick={handleSave} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}>Save Settings</button>
          </div>
        </div>
      </div>

      <ConfirmDialog
        open={showRestartPrompt}
        title="Restart Required"
        message="Database location has been updated. NIL needs to restart to apply the changes."
        confirmText="Restart Now"
        cancelText="Later"
        onConfirm={async () => {
          setShowRestartPrompt(false);
          try {
            await Backend.Restart();
          } catch (err) {
            console.error('Restart failed:', err);
          }
        }}
        onCancel={() => {
          setShowRestartPrompt(false);
          onOpenChange(false);
        }}
      />

      <ConfirmDialog
        open={confirmDeleteVaultId !== null}
        title="Delete Vault"
        message={`Remove "${vaults.find(v => v.id === confirmDeleteVaultId)?.name ?? ''}" from the registry? Database files on disk will be preserved.`}
        confirmText="Delete"
        cancelText="Cancel"
        onConfirm={async () => {
          if (!confirmDeleteVaultId) return;
          try {
            await Backend.DeleteVault(confirmDeleteVaultId);
            setVaults(prev => prev.filter(v => v.id !== confirmDeleteVaultId));
          } catch (err: any) {
            alert(`Failed to delete vault: ${err.message || err}`);
          }
          setConfirmDeleteVaultId(null);
        }}
        onCancel={() => setConfirmDeleteVaultId(null)}
      />
    </>
  );
}
