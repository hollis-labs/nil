import * as React from "react";
import {
  SessionProfile,
  getSessionProfiles,
  saveSessionProfile,
  updateSessionProfile,
  deleteSessionProfile,
  getActiveSession,
  setActiveSession,
  clearActiveSession,
} from "@/lib/sessionContext";
import ContextsAutocomplete from "./ContextsAutocomplete";
import ProjectsAutocomplete from "./ProjectsAutocomplete";
import TagsAutocomplete from "./TagsAutocomplete";
import CustomScrollbar from "./CustomScrollbar";
import { Trash2, Edit3, Plus, Copy, ToggleLeft, ToggleRight } from "lucide-react";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSessionChanged?: () => void;
};

export default function SessionContextModal({ open, onOpenChange, onSessionChanged }: Props) {
  const [profiles, setProfiles] = React.useState<SessionProfile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = React.useState<string | null>(null);
  const [profileName, setProfileName] = React.useState("");
  const [contexts, setContexts] = React.useState<string[]>([]);
  const [projects, setProjects] = React.useState<string[]>([]);
  const [tags, setTags] = React.useState<string[]>([]);
  const [priority, setPriority] = React.useState<'A' | 'B' | 'C' | ''>('');
  const [isNewProfile, setIsNewProfile] = React.useState(false);
  const [statusMessage, setStatusMessage] = React.useState<string>('');
  const [showStatus, setShowStatus] = React.useState(false);
  const [useAsFilterTab, setUseAsFilterTab] = React.useState(false);
  const [comboboxOpen, setComboboxOpen] = React.useState(false);
  const [comboboxSearch, setComboboxSearch] = React.useState('');

  React.useEffect(() => {
    if (open) {
      loadProfiles();
      loadActiveSession();
    }
  }, [open]);

  const loadProfiles = () => {
    setProfiles(getSessionProfiles());
  };

  const loadActiveSession = () => {
    const active = getActiveSession();
    if (active) {
      setContexts(active.contexts);
      setProjects(active.projects);
      setTags(active.tags);
      setPriority(active.priority || '');
      setSelectedProfileId(active.profileId || null);
      setUseAsFilterTab(active.useAsFilterTab || false);

      if (active.profileId) {
        const profile = getSessionProfiles().find(p => p.id === active.profileId);
        if (profile) {
          setProfileName(profile.name);
        }
      }
    }
  };

  const handleSelectProfile = (profileId: string) => {
    const profile = profiles.find(p => p.id === profileId);
    if (profile) {
      setSelectedProfileId(profileId);
      setProfileName(profile.name);
      setContexts(profile.contexts);
      setProjects(profile.projects);
      setTags(profile.tags);
      setPriority(profile.priority || '');
      setIsNewProfile(false);
    }
  };

  const handleNewProfile = () => {
    setSelectedProfileId(null);
    setProfileName("");
    setContexts([]);
    setProjects([]);
    setTags([]);
    setPriority('');
    setIsNewProfile(true);
  };

  const showStatusMessage = (message: string) => {
    setStatusMessage(message);
    setShowStatus(true);
    setTimeout(() => {
      setShowStatus(false);
    }, 2000);
  };

  const handleApply = () => {
    setActiveSession({
      profileId: selectedProfileId || undefined,
      contexts,
      projects,
      tags,
      priority: priority || undefined,
      useAsFilterTab,
    });
    onSessionChanged?.();
    showStatusMessage('Settings applied');
    setTimeout(() => {
      onOpenChange(false);
    }, 500);
  };

  const handleClear = () => {
    clearActiveSession();
    setContexts([]);
    setProjects([]);
    setTags([]);
    setPriority('');
    setSelectedProfileId(null);
    setProfileName("");
    setIsNewProfile(false);
    onSessionChanged?.();
  };

  const handleSaveProfile = () => {
    if (!profileName.trim()) return;

    if (selectedProfileId && !isNewProfile) {
      updateSessionProfile(selectedProfileId, {
        name: profileName.trim(),
        contexts,
        projects,
        tags,
        priority: priority || undefined,
      });
    } else {
      const newProfile = saveSessionProfile({
        name: profileName.trim(),
        contexts,
        projects,
        tags,
        priority: priority || undefined,
      });
      setSelectedProfileId(newProfile.id);
      setIsNewProfile(false);
    }

    loadProfiles();
    showStatusMessage('Settings saved');
  };

  const handleSaveAndApply = () => {
    if (!profileName.trim()) {
      return;
    }

    let profileId = selectedProfileId;

    if (selectedProfileId && !isNewProfile) {
      updateSessionProfile(selectedProfileId, {
        name: profileName.trim(),
        contexts,
        projects,
        tags,
        priority: priority || undefined,
      });
    } else {
      const newProfile = saveSessionProfile({
        name: profileName.trim(),
        contexts,
        projects,
        tags,
        priority: priority || undefined,
      });
      profileId = newProfile.id;
      setSelectedProfileId(newProfile.id);
      setIsNewProfile(false);
    }

    loadProfiles();
    setActiveSession({
      profileId: profileId || undefined,
      contexts,
      projects,
      tags,
      priority: priority || undefined,
      useAsFilterTab,
    });
    
    showStatusMessage('Settings saved & applied');
    
    setTimeout(() => {
      onSessionChanged?.();
      onOpenChange(false);
    }, 2000);
  };

  const handleDeleteProfile = (profileId: string) => {
    deleteSessionProfile(profileId);
    if (selectedProfileId === profileId) {
      handleNewProfile();
    }
    loadProfiles();
  };

  const handleClone = () => {
    setProfileName(profileName ? `${profileName} (copy)` : '');
    setSelectedProfileId(null);
    setIsNewProfile(true);
    setTimeout(() => {
      const input = document.querySelector('input[placeholder="Profile name"]') as HTMLInputElement;
      if (input) {
        input.focus();
        input.select();
      }
    }, 0);
  };

  if (!open) return null;

  const modalStyle: React.CSSProperties = {
    position: 'fixed',
    inset: 0,
    background: 'rgba(0,0,0,0.7)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 50,
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '8px 12px',
    background: 'var(--term-bg)',
    border: '1px solid var(--term-border)',
    borderRadius: '6px',
    color: 'var(--term-fg)',
    fontSize: '13px',
  };

  return (
    <>
      <style>{`
        @keyframes fadeIn {
          from {
            opacity: 0;
            transform: translateX(-50%) translateY(-5px);
          }
          to {
            opacity: 1;
            transform: translateX(-50%) translateY(0);
          }
        }
      `}</style>
    <div style={modalStyle}>
      <div className="terminal-card" style={{
        width: '720px',
        maxWidth: '95vw',
        height: '650px',
        maxHeight: '90vh',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden'
      }}>
        {/* Header */}
        <div style={{ padding: '20px 20px 0 20px' }}>
          <div style={{ marginBottom: '16px' }}>
            <div style={{ display: 'flex', alignItems: 'end', justifyContent: 'flex-end', marginBottom: '1px' }}>
              <button
                className="badge info"
                onClick={() => onOpenChange(false)}
                style={{
                  padding: '8px 12px',
                  fontSize: '13px',
                  borderRadius: '6px'
                }}
              >
                Close
              </button>
            </div>
            <label style={{ fontSize: '12px', marginLeft: '3px', textTransform: 'uppercase', letterSpacing: '0.5px' }} className="text-dim">Profile:</label>
            <input
              type="text"
              style={{
                width: '100%',
                padding: '4px 4px',
                background: 'transparent',
                border: 'none',
                borderBottom: '1px solid var(--term-border)',
                borderRadius: '0',
                color: 'var(--term-fg)',
                fontSize: '18px',
                fontWeight: 500,
                outline: 'none'
              }}
              placeholder="Profile name"
              value={profileName}
              onChange={(e) => setProfileName(e.target.value)}
            />
          </div>
        </div>

        {/* Scrollable Body */}
        <CustomScrollbar style={{ flex: 1, minHeight: 0 }}>
          <div style={{ padding: '0 20px 40px 20px' }}>
            {/* Priority chips row */}
            <div style={{
              display: 'flex',
              gap: '0px',
              marginBottom: '12px',
              flexWrap: 'wrap'
            }}>
              <button
                type="button"
                className={`badge ${priority === 'C' ? 'success' : ''}`}
                onClick={() => setPriority(priority === 'C' ? '' : 'C')}
                style={{ 
                  flex: '0 0 auto',
                  border: 'none',
                  borderRadius: '4px 0 0 4px',
                  fontSize: '11px',
                  padding: '4px 8px',
                  opacity: priority === 'C' ? 1 : 0.8
                }}
              >
                LOW
              </button>
              <button
                type="button"
                className={`badge ${priority === 'B' ? 'info' : ''}`}
                onClick={() => setPriority(priority === 'B' ? '' : 'B')}
                style={{ 
                  flex: '0 0 auto',
                  border: 'none',
                  borderRadius: '0',
                  borderLeft: '1px solid var(--term-border)',
                  fontSize: '11px',
                  padding: '4px 8px',
                  opacity: priority === 'B' ? 1 : 0.8
                }}
              >
                MED
              </button>
              <button
                type="button"
                className={`badge ${priority === 'A' ? 'warn' : ''}`}
                onClick={() => setPriority(priority === 'A' ? '' : 'A')}
                style={{ 
                  flex: '0 0 auto',
                  border: 'none',
                  borderRadius: '0',
                  borderLeft: '1px solid var(--term-border)',
                  fontSize: '11px',
                  padding: '4px 8px',
                  opacity: priority === 'A' ? 1 : 0.8
                }}
              >
                HIGH
              </button>
              <button
                type="button"
                className={`badge ${priority === '' ? 'success' : ''}`}
                onClick={() => setPriority('')}
                style={{ 
                  flex: '0 0 auto',
                  border: 'none',
                  borderRadius: '0 4px 4px 0',
                  borderLeft: '1px solid var(--term-border)',
                  fontSize: '11px',
                  padding: '4px 8px'
                }}
              >
                NA
              </button>
              <div style={{ 
                marginLeft: 'auto', 
                display: 'flex', 
                alignItems: 'center', 
                gap: '8px',
                paddingLeft: '16px',
                borderLeft: '1px solid var(--term-border)',
                cursor: 'pointer'
              }}
              onClick={() => setUseAsFilterTab(!useAsFilterTab)}
              >
                {useAsFilterTab ? (
                  <ToggleRight size={18} style={{ color: 'var(--term-accent)' }} />
                ) : (
                  <ToggleLeft size={18} style={{ color: 'var(--term-dim)', opacity: 0.6 }} />
                )}
                <label style={{ fontSize: '11px', cursor: 'pointer', whiteSpace: 'nowrap' }} className="text-dim">
                  Use as filter tab
                </label>
              </div>
            </div>

            {/* Profile Selection */}
            <div style={{ display: 'flex', gap: '8px', marginBottom: '12px' }}>
              <div style={{ flex: 1, position: 'relative' }}>
                <input
                  type="text"
                  value={comboboxSearch}
                  onChange={(e) => setComboboxSearch(e.target.value)}
                  onFocus={(e) => {
                    setComboboxOpen(true);
                    e.currentTarget.style.borderColor = 'var(--term-accent)';
                  }}
                  onBlur={(e) => {
                    setTimeout(() => setComboboxOpen(false), 200);
                    e.currentTarget.style.borderColor = 'var(--term-border)';
                  }}
                  placeholder={selectedProfileId ? profiles.find(p => p.id === selectedProfileId)?.name || 'New Profile...' : 'New Profile...'}
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    background: 'var(--term-bg)',
                    border: '1px solid var(--term-border)',
                    borderRadius: '6px',
                    color: 'var(--term-fg)',
                    fontSize: '13px',
                    height: '40px',
                    outline: 'none'
                  }}
                />
                {comboboxOpen && (
                  <div style={{
                    position: 'absolute',
                    top: '100%',
                    left: 0,
                    right: 0,
                    marginTop: '4px',
                    background: 'var(--term-panel)',
                    border: '1px solid var(--term-border)',
                    borderRadius: '6px',
                    maxHeight: '200px',
                    overflowY: 'auto',
                    zIndex: 1000,
                    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.3)'
                  }}>
                    <div
                      onClick={() => {
                        handleNewProfile();
                        setComboboxSearch('');
                        setComboboxOpen(false);
                      }}
                      style={{
                        padding: '8px 12px',
                        cursor: 'pointer',
                        fontSize: '13px',
                        opacity: 0.8,
                        borderBottom: '1px solid var(--term-border)'
                      }}
                      onMouseEnter={(e) => e.currentTarget.style.background = 'var(--term-bg)'}
                      onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                    >
                      New Profile...
                    </div>
                    {profiles
                      .filter(p => p.name.toLowerCase().includes(comboboxSearch.toLowerCase()))
                      .slice(0, 5)
                      .sort((a, b) => a.name.localeCompare(b.name))
                      .map(profile => (
                        <div
                          key={profile.id}
                          onClick={() => {
                            handleSelectProfile(profile.id);
                            setComboboxSearch('');
                            setComboboxOpen(false);
                          }}
                          style={{
                            padding: '8px 12px',
                            cursor: 'pointer',
                            fontSize: '13px',
                            background: selectedProfileId === profile.id ? 'var(--term-accent-dim)' : 'transparent'
                          }}
                          onMouseEnter={(e) => e.currentTarget.style.background = 'var(--term-bg)'}
                          onMouseLeave={(e) => e.currentTarget.style.background = selectedProfileId === profile.id ? 'var(--term-accent-dim)' : 'transparent'}
                        >
                          {profile.name}
                        </div>
                      ))}
                  </div>
                )}
              </div>
              <button
                type="button"
                className="badge success"
                onClick={handleApply}
                style={{
                  padding: '8px 12px',
                  fontSize: '13px',
                  whiteSpace: 'nowrap',
                  borderRadius: '6px',
                  height: '40px'
                }}
              >
                Apply
              </button>
            </div>

            {/* 2x2 Grid */}
            <div style={{
              display: 'grid',
              gridTemplateColumns: '1fr 1fr',
              gap: '12px',
              marginBottom: '12px'
            }}>
              <div>
                <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">Projects</label>
                <ProjectsAutocomplete values={projects} onValuesChange={setProjects} placeholder="Add project..." />
              </div>

              <div>
                <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">Contexts</label>
                <ContextsAutocomplete values={contexts} onValuesChange={setContexts} placeholder="Add context..." />
              </div>

              <div>
                <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">Tags</label>
                <TagsAutocomplete values={tags} onValuesChange={setTags} placeholder="Add tag..." />
              </div>

              <div>
                <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">Summary</label>
                <div style={{
                  padding: '8px 12px',
                  background: 'var(--term-bg)',
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  minHeight: '44px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-around',
                  gap: '12px',
                  flexWrap: 'wrap'
                }}>
                  <div style={{ fontSize: '11px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <span style={{ fontWeight: 600, color: 'var(--term-fg)', opacity: 0.8 }}>PRO:</span>
                    <span style={{ color: 'var(--term-info)', minWidth: '10px', textAlign: 'center' }}>{projects.length}</span>
                  </div>
                  <div style={{ fontSize: '11px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <span style={{ fontWeight: 600, color: 'var(--term-fg)', opacity: 0.8 }}>CON:</span>
                    <span style={{ color: 'var(--term-info)', minWidth: '10px', textAlign: 'center' }}>{contexts.length}</span>
                  </div>
                  <div style={{ fontSize: '11px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <span style={{ fontWeight: 600, color: 'var(--term-fg)', opacity: 0.8 }}>TAG:</span>
                    <span style={{ color: 'var(--term-info)', minWidth: '10px', textAlign: 'center' }}>{tags.length}</span>
                  </div>
                  <div style={{ fontSize: '11px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                    <span style={{ fontWeight: 600, color: 'var(--term-fg)', opacity: 0.8 }}>PRI:</span>
                    <span style={{ color: 'var(--term-info)', minWidth: '32px', display: 'inline-block' }}>
                      {priority === 'A' ? 'High' : priority === 'B' ? 'Med' : priority === 'C' ? 'Low' : 'None'}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Help Section */}
            <div style={{ marginTop: '20px' }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', marginLeft: '4px', display: 'block' }} className="text-dim">How Session Context Works</label>
              <div style={{
                padding: '12px 16px',
                background: 'var(--term-bg)',
                border: '1px solid var(--term-border)',
                borderRadius: '6px',
                fontSize: '12px',
                lineHeight: '1.6',
                color: 'var(--term-dim)'
              }}>
                <p style={{ margin: '0 0 8px 0' }}>
                  <strong style={{ color: 'var(--term-fg)' }}>Session Context</strong> lets you set default filters for new todos.
                </p>
                <p style={{ margin: '0 0 8px 0' }}>
                  <strong style={{ color: 'var(--term-fg)' }}>Profiles:</strong> Save frequently-used combinations as profiles. Select from the dropdown to load, edit, or create new ones.
                </p>
                <p style={{ margin: '0' }}>
                  <strong style={{ color: 'var(--term-fg)' }}>Use as filter tab:</strong> When enabled, this profile becomes an active filter that overrides your tab filters. All searches will be restricted to items matching this context until you turn it off or clear the session.
                </p>
              </div>
            </div>

          </div>
        </CustomScrollbar>

        {/* Footer */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '16px 20px 20px 20px',
          borderTop: '1px solid var(--term-border)',
          position: 'relative'
        }}>
          {showStatus && (
            <div style={{
              position: 'absolute',
              top: '-30px',
              left: '50%',
              transform: 'translateX(-50%)',
              padding: '6px 12px',
              background: 'var(--term-success)',
              color: 'var(--term-bg)',
              borderRadius: '4px',
              fontSize: '12px',
              fontWeight: 500,
              whiteSpace: 'nowrap',
              boxShadow: '0 2px 8px rgba(0,0,0,0.2)',
              animation: 'fadeIn 0.2s ease-in-out'
            }}>
              {statusMessage}
            </div>
          )}
          <div style={{ display: 'flex', gap: '8px' }}>
            {selectedProfileId && (
              <button
                type="button"
                className="badge warn"
                onClick={() => handleDeleteProfile(selectedProfileId)}
                style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
              >
                Delete
              </button>
            )}
            {selectedProfileId && (
              <button
                type="button"
                className="badge info"
                onClick={handleClone}
                style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', display: 'flex', alignItems: 'center', gap: '4px' }}
              >
                <Copy size={14} />
                Clone
              </button>
            )}
            <button type="button" className="badge" onClick={handleClear} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', background: 'var(--term-bg)', border: '1px solid var(--term-border)' }}>
              Clear
            </button>
            <button type="button" className="badge" onClick={() => onOpenChange(false)} style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px', background: 'var(--term-bg)', border: '1px solid var(--term-border)' }}>Cancel</button>
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              type="button"
              className="badge info"
              onClick={handleSaveProfile}
              disabled={!profileName.trim()}
              style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
            >
              Save
            </button>
            <button
              type="button"
              className="badge success"
              onClick={handleSaveAndApply}
              disabled={!profileName.trim()}
              style={{ padding: '8px 12px', fontSize: '13px', borderRadius: '6px' }}
            >
              Save & Apply
            </button>
          </div>
        </div>
      </div>
    </div>
    </>
  );
}
