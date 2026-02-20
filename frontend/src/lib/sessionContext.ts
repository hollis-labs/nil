export type SessionProfile = {
  id: string;
  name: string;
  contexts: string[];
  projects: string[];
  tags: string[];
  priority?: 'A' | 'B' | 'C';
  createdAt: string;
};

export type ActiveSession = {
  profileId?: string;
  contexts: string[];
  projects: string[];
  tags: string[];
  priority?: 'A' | 'B' | 'C';
  useAsFilterTab?: boolean;
};

const PROFILES_KEY = 'nanite.sessionProfiles';
const ACTIVE_KEY = 'nanite.activeSession';

export function getSessionProfiles(): SessionProfile[] {
  try {
    const stored = localStorage.getItem(PROFILES_KEY);
    if (!stored) return [];
    const parsed = JSON.parse(stored);
    return Array.isArray(parsed) ? parsed : [];
  } catch (err) {
    console.error('Failed to load session profiles:', err);
    return [];
  }
}

export function saveSessionProfile(profile: Omit<SessionProfile, 'id' | 'createdAt'>): SessionProfile {
  const profiles = getSessionProfiles();
  const newProfile: SessionProfile = {
    ...profile,
    id: Date.now().toString(),
    createdAt: new Date().toISOString(),
  };
  profiles.push(newProfile);
  localStorage.setItem(PROFILES_KEY, JSON.stringify(profiles));
  return newProfile;
}

export function updateSessionProfile(id: string, updates: Partial<SessionProfile>): void {
  const profiles = getSessionProfiles();
  const index = profiles.findIndex(p => p.id === id);
  if (index >= 0) {
    profiles[index] = { ...profiles[index], ...updates };
    localStorage.setItem(PROFILES_KEY, JSON.stringify(profiles));
  }
}

export function deleteSessionProfile(id: string): void {
  const profiles = getSessionProfiles();
  const filtered = profiles.filter(p => p.id !== id);
  localStorage.setItem(PROFILES_KEY, JSON.stringify(filtered));

  // Clear active session if it was using this profile
  const active = getActiveSession();
  if (active?.profileId === id) {
    clearActiveSession();
  }
}

export function getActiveSession(): ActiveSession | null {
  try {
    const stored = localStorage.getItem(ACTIVE_KEY);
    if (!stored) return null;
    return JSON.parse(stored);
  } catch (err) {
    console.error('Failed to load active session:', err);
    return null;
  }
}

export function setActiveSession(session: ActiveSession): void {
  localStorage.setItem(ACTIVE_KEY, JSON.stringify(session));
}

export function clearActiveSession(): void {
  localStorage.removeItem(ACTIVE_KEY);
}
