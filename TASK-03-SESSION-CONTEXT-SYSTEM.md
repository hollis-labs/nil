# Task 03: Session Context System

## Overview
Create a session context system that allows users to set default values (contexts, projects, tags, priority) that apply to all new todos until cleared. This takes priority over search filters.

## What Needs to Be Done

### 1. Define Session Context Type & Storage
**File**: `frontend/src/lib/sessionContext.ts`

**Types**:
```tsx
type SessionProfile = {
  id: string;
  name: string;
  contexts: string[];
  projects: string[];
  tags: string[];
  priority?: 'A' | 'B' | 'C';
  createdAt: string;
}

type ActiveSession = {
  profileId?: string; // If loaded from profile
  contexts: string[];
  projects: string[];
  tags: string[];
  priority?: 'A' | 'B' | 'C';
}
```

**Functions**:
- `getSessionProfiles(): SessionProfile[]`
- `saveSessionProfile(profile: Omit<SessionProfile, 'id' | 'createdAt'>): void`
- `updateSessionProfile(id: string, updates: Partial<SessionProfile>): void`
- `deleteSessionProfile(id: string): void`
- `getActiveSession(): ActiveSession | null`
- `setActiveSession(session: ActiveSession): void`
- `clearActiveSession(): void`

**Storage Keys**:
- `'planck.sessionProfiles'`
- `'planck.activeSession'`

### 2. Create SessionContextModal Component
**File**: `frontend/src/components/SessionContextModal.tsx`

**Features**:
- Header: "Session Context" with close button
- Profile selector combobox (search + select)
- "+ New Profile" button (clears form, creates new on save)
- Form fields using autocomplete components:
  - Contexts (multi-select autocomplete)
  - Projects (multi-select autocomplete)
  - Tags (multi-select autocomplete)
  - Priority (radio chips: High/Med/Low/None)
- Profile name input (when creating/editing profile)
- Action buttons:
  - "Apply" - Sets active session, closes modal
  - "Clear" - Clears active session
  - "Save Profile" - Saves as named profile (if new) or updates (if editing)
  - "Delete Profile" - Deletes selected profile (if not new)
- Table view of saved profiles with edit/delete icons

**Behavior**:
- Opens with current active session pre-filled
- If active filters exist on open, suggest converting to session
- Selecting profile loads its values
- Apply button sets as active session
- Profile changes only saved when "Save Profile" clicked

### 3. Update App.tsx Navigation
**File**: `frontend/src/pages/App.tsx`

**Changes**:
- Move settings icon (SlidersVertical) from header to footer (right-aligned)
- Change to gear icon in footer
- In header where settings icon was: add new icon for Session Context
- Suggested icon: `User` or `UserCog` or `Target` from lucide-react
- Wire up to open SessionContextModal

### 4. Integrate with EditTodoModal
**File**: `frontend/src/components/EditTodoModal.tsx`

**Priority Order** (when creating new todo):
1. Active session context (if set)
2. Search filters (current behavior)
3. Default tags (from settings)

**Implementation**:
- Read active session on mount
- Merge session values with existing default logic
- Session values should append to (not replace) manual user input

### 5. Visual Indicator
**Location**: Header near Session Context button

**Features**:
- Show small badge/indicator when session context is active
- Badge shows # of active filters (e.g., "3")
- Clicking badge opens session modal
- Badge color matches theme accent

## Technical Notes
- Session persists across app restarts (localStorage)
- Session is separate from search filters
- Session can be loaded from profile or set manually
- Profile changes don't affect active session until "Apply" clicked
- Auto-convert search filters to session (optional prompt)

## Testing
- Create new session profile
- Load profile into session
- Verify new todos get session defaults
- Edit profile, verify session updates only on "Apply"
- Clear session, verify defaults revert
- Test priority: session > search > defaults
- Test with search filters active
- Test persistence across refresh
