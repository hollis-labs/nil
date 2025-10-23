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
import { Trash2, Edit3, Plus } from "lucide-react";

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

  const handleApply = () => {
    setActiveSession({
      profileId: selectedProfileId || undefined,
      contexts,
      projects,
      tags,
      priority: priority || undefined,
    });
    onSessionChanged?.();
    onOpenChange(false);
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
  };

  const handleDeleteProfile = (profileId: string) => {
    deleteSessionProfile(profileId);
    if (selectedProfileId === profileId) {
      handleNewProfile();
    }
    loadProfiles();
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
    <div style={modalStyle}>
      <div className="terminal-card" style={{
        width: '720px',
        maxWidth: '95vw',
        height: 'auto',
        maxHeight: '90vh',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden'
      }}>
        {/* Header */}
        <div style={{ padding: '20px 20px 0 20px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
            <div style={{ fontWeight: 600 }}>Session Context</div>
            <button className="badge" onClick={() => onOpenChange(false)}>Close</button>
          </div>
        </div>

        {/* Scrollable Body */}
        <CustomScrollbar style={{ flex: 1, minHeight: '400px', maxHeight: '500px' }}>
          <div style={{ paddingLeft: '20px', marginRight: '30px', paddingBottom: '40px' }}>
            {/* Profile Selection */}
            <div style={{ marginBottom: '16px' }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">
                Select Profile
              </label>
              <div style={{ display: 'flex', gap: '8px' }}>
                <select
                  style={{
                    ...inputStyle,
                    flex: 1,
                    appearance: 'none',
                    backgroundImage: `url("data:image/svg+xml;charset=UTF-8,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3e%3cpolyline points='6 9 12 15 18 9'%3e%3c/polyline%3e%3c/svg%3e")`,
                    backgroundRepeat: 'no-repeat',
                    backgroundPosition: 'right 8px center',
                    backgroundSize: '16px',
                    paddingRight: '32px'
                  }}
                  value={selectedProfileId || ''}
                  onChange={(e) => {
                    if (e.target.value) {
                      handleSelectProfile(e.target.value);
                    } else {
                      handleNewProfile();
                    }
                  }}
                >
                  <option value="" style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>
                    New Profile...
                  </option>
                  {profiles.map(profile => (
                    <option key={profile.id} value={profile.id} style={{ background: 'var(--term-panel)', color: 'var(--term-fg)' }}>
                      {profile.name}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  className="badge success"
                  onClick={handleApply}
                  style={{
                    padding: '6px 12px',
                    fontSize: '11px',
                    borderRadius: '3px',
                    whiteSpace: 'nowrap'
                  }}
                >
                  Apply
                </button>
              </div>
            </div>

            {/* Profile Name (always shown) */}
            <div style={{ marginBottom: '16px' }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">
                Profile Name
              </label>
              <input
                type="text"
                style={inputStyle}
                placeholder="e.g., Work Context"
                value={profileName}
                onChange={(e) => setProfileName(e.target.value)}
              />
            </div>

            {/* Priority */}
            <div style={{ marginBottom: '16px' }}>
              <label style={{ fontSize: '12px', marginBottom: '8px', display: 'block' }} className="text-dim">
                Priority
              </label>
              <div style={{ display: 'flex', gap: '8px' }}>
                <button
                  type="button"
                  className={`badge ${priority === 'A' ? 'warn' : ''}`}
                  onClick={() => setPriority(priority === 'A' ? '' : 'A')}
                  style={{ flex: 1 }}
                >
                  High
                </button>
                <button
                  type="button"
                  className={`badge ${priority === 'B' ? 'info' : ''}`}
                  onClick={() => setPriority(priority === 'B' ? '' : 'B')}
                  style={{ flex: 1 }}
                >
                  Medium
                </button>
                <button
                  type="button"
                  className={`badge ${priority === 'C' ? 'success' : ''}`}
                  onClick={() => setPriority(priority === 'C' ? '' : 'C')}
                  style={{ flex: 1 }}
                >
                  Low
                </button>
                <button
                  type="button"
                  className={`badge ${priority === '' ? 'success' : ''}`}
                  onClick={() => setPriority('')}
                  style={{ flex: 1 }}
                >
                  None
                </button>
              </div>
            </div>

            {/* Contexts */}
            <div style={{ marginBottom: '16px' }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">
                Contexts
              </label>
              <ContextsAutocomplete values={contexts} onValuesChange={setContexts} placeholder="Add context..." />
            </div>

            {/* Projects */}
            <div style={{ marginBottom: '16px' }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">
                Projects
              </label>
              <ProjectsAutocomplete values={projects} onValuesChange={setProjects} placeholder="Add project..." />
            </div>

            {/* Tags */}
            <div style={{ marginBottom: '16px' }}>
              <label style={{ fontSize: '12px', marginBottom: '4px', display: 'block' }} className="text-dim">
                Tags
              </label>
              <TagsAutocomplete values={tags} onValuesChange={setTags} placeholder="Add tag..." />
            </div>

            {/* Saved Profiles Table */}
            {profiles.length > 0 && (
              <div style={{ marginTop: '24px' }}>
                <label style={{ fontSize: '12px', marginBottom: '8px', display: 'block', fontWeight: 500 }}>
                  Saved Profiles
                </label>
                <div style={{
                  border: '1px solid var(--term-border)',
                  borderRadius: '6px',
                  overflow: 'hidden'
                }}>
                  {profiles.map(profile => (
                    <div
                      key={profile.id}
                      style={{
                        padding: '12px',
                        borderBottom: '1px solid var(--term-border)',
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        background: selectedProfileId === profile.id ? 'rgba(122, 162, 247, 0.1)' : 'transparent',
                      }}
                    >
                      <div style={{ flex: 1 }}>
                        <div style={{ fontWeight: 500, marginBottom: '4px', fontSize: '13px' }}>{profile.name}</div>
                        <div style={{ fontSize: '11px', color: 'var(--term-dim)' }}>
                          {profile.projects.length > 0 && <span>+{profile.projects.join(', +')} </span>}
                          {profile.contexts.length > 0 && <span>@{profile.contexts.join(', @')} </span>}
                          {profile.tags.length > 0 && <span>#{profile.tags.join(', #')} </span>}
                          {profile.priority && <span>pri:{profile.priority}</span>}
                        </div>
                      </div>
                      <div style={{ display: 'flex', gap: '4px' }}>
                        <button
                          type="button"
                          onClick={() => handleSelectProfile(profile.id)}
                          style={{
                            background: 'transparent',
                            border: 'none',
                            cursor: 'pointer',
                            padding: '4px',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: 'var(--term-fg)',
                            opacity: 0.6,
                            transition: 'opacity 0.15s ease'
                          }}
                          onMouseEnter={(e) => { e.currentTarget.style.opacity = '1'; }}
                          onMouseLeave={(e) => { e.currentTarget.style.opacity = '0.6'; }}
                          title="Edit"
                        >
                          <Edit3 size={14} />
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDeleteProfile(profile.id)}
                          style={{
                            background: 'transparent',
                            border: 'none',
                            cursor: 'pointer',
                            padding: '4px',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: 'var(--term-fg)',
                            opacity: 0.6,
                            transition: 'opacity 0.15s ease'
                          }}
                          onMouseEnter={(e) => { e.currentTarget.style.opacity = '1'; }}
                          onMouseLeave={(e) => { e.currentTarget.style.opacity = '0.6'; }}
                          title="Delete"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </CustomScrollbar>

        {/* Footer */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '16px 20px 20px 20px',
          borderTop: '1px solid var(--term-border)',
          flexWrap: 'wrap',
          gap: '8px'
        }}>
          <div style={{ display: 'flex', gap: '8px' }}>
            {selectedProfileId && (
              <button
                type="button"
                className="badge warn"
                onClick={() => handleDeleteProfile(selectedProfileId)}
              >
                Delete
              </button>
            )}
            <button type="button" className="badge" onClick={handleClear}>
              Clear
            </button>
          </div>
          <button
            type="button"
            className="badge info"
            onClick={handleSaveProfile}
            disabled={!profileName.trim()}
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
