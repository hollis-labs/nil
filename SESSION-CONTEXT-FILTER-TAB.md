# Session Context as Filter Tab Feature

## Overview
This feature allows users to use their active session context as a dynamic filter tab, replacing the standard view tabs when enabled.

## Implementation Details

### 1. Data Model (`sessionContext.ts`)
- Added `useAsFilterTab?: boolean` to `ActiveSession` type
- Field is persisted in localStorage with the session state

### 2. UI Component (`SessionContextModal.tsx`)
- Added toggle checkbox in the priority chips row
- Label: "Use as filter tab"
- State is saved when clicking "Apply" or "Save & Apply"
- Toggle state persists across modal opens

### 3. Main Application (`App.tsx`)

#### State Management
- Added `sessionAsFilter` state variable to track when session filter is active
- Integrated with existing `updateSessionFilterCount()` callback to update both count and filter state

#### Search Logic
- Modified `runSearch()` to check for active session with `useAsFilterTab: true`
- When session filter is active:
  - Regular tabs are ignored (`activeTab = null`)
  - Session contexts, projects, tags, and priority are merged with search filters
  - Session filters are added alongside main query and tab filters

#### Visual Feedback

**Session Context Button:**
- Background: `var(--term-accent)` when filter is active (otherwise `var(--term-panel)`)
- Border: `var(--term-accent)` when active (otherwise `var(--term-border)`)
- Icon color: `#000` when active (otherwise `var(--term-fg)`)
- Title: "Session Context (Active Filter)" when active

**Tab Area:**
- Shows "Session Filter" badge with Target icon when active
- Badge style: `warn` class with bold font
- Regular tabs are dimmed (opacity: 0.4)
- Regular tabs are disabled (cursor: not-allowed, non-clickable)
- Active tab highlighting is removed when session filter is active

## User Workflow

1. **Enable Session Filter:**
   - Open Session Context modal (Target icon)
   - Set up desired contexts, projects, tags, priority
   - Check "Use as filter tab" checkbox
   - Click "Apply" or "Save & Apply"

2. **Visual Changes:**
   - Session Context button becomes highlighted (accent color background)
   - "Session Filter" badge appears in tab area
   - Regular tabs become dimmed and non-interactive
   - Search results filtered by session context settings

3. **Disable Session Filter:**
   - Open Session Context modal
   - Uncheck "Use as filter tab"
   - Click "Apply" or "Save & Apply"
   - Regular tabs become active again

## Technical Notes

- Session filter merges with main search query (doesn't override it)
- When session filter is active, tab queries are ignored
- Filter state updates on every `onSessionChanged` callback
- State persists across app restarts (stored in localStorage)

## Testing Checklist

- [ ] Toggle checkbox appears in SessionContextModal
- [ ] Toggle state persists when modal is closed and reopened
- [ ] "Session Filter" badge appears when toggle is enabled
- [ ] Regular tabs become dimmed when session filter is active
- [ ] Regular tabs cannot be clicked when session filter is active
- [ ] Session Context button highlights when filter is active
- [ ] Search results respect session filter settings
- [ ] Session filter merges correctly with main search query
- [ ] Disabling toggle restores normal tab behavior
- [ ] State persists after app restart
