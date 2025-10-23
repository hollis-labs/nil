# Feature Update - October 22, 2025

## Summary
Major UI enhancements to the PLANCK todo app including collapsible sections, dual view modes, gradient colors, and customizable scrollbars.

## Features Implemented

### 1. Done Section with Show/Hide
- **New "Done" section** appears below Anytime section
- Contains all completed todos
- **Collapsed by default** to keep UI clean
- Click to expand/collapse with arrow indicator (▸/▾)
- Shows count of completed items in header

### 2. Collapsible Section Headers
- **All sections now collapsible**: Now, Soon, Anytime, Done
- Click section header to toggle visibility
- Visual indicators: ▸ (collapsed) / ▾ (expanded)
- Item count shown in header: "Now (5)"
- State persists during session
- Done section defaults to collapsed

### 3. Gradient Colors for Tags & Projects
**Tags:**
- 3-shade gray gradient: bright → dark
- Colors: `#b4b8c0` → `#999da5` → `#7e8289`
- First tag gets brightest, third+ get darkest
- Creates visual hierarchy without overwhelming design

**Projects:**
- 3-shade gradient based on theme accent color
- Programmatically calculates: bright (+15%) → normal → dark (-25%)
- Adapts to any theme (default, light, synthwave, terminal)
- Maintains theme consistency while adding distinction

### 4. Date-Based View Mode
**New View:** Grouped by Due Dates
- Shows todos organized chronologically by due date
- Formatted headers: "Mon, Oct 22, 2025"
- Only shows todos WITH due dates
- Falls back to "No todos with due dates" if none exist
- Same gradient colors for tags/projects
- Clean, calendar-like organization

### 5. View Mode Toggle
**Scope View vs Date View:**
- Toggle buttons in header: List icon (Scope) / Calendar icon (Date)
- **Scope View:** Now / Soon / Anytime / Done sections
- **Date View:** Chronological by due date
- Persists selection in localStorage (`planck.viewMode`)
- Icons from lucide-react for clarity

### 6. Default View Setting
**Settings → General:**
- Choose default view on app load
- Options: "Scope (Now/Soon/Anytime)" or "Date"
- Stored in settings, syncs across sessions
- Helps users who prefer date-driven workflows

### 7. Custom Scrollbar Theme Controls
**New Theme Variables:**
- `--scrollbar-bg`: Scrollbar track background
- `--scrollbar-thumb`: Scrollbar thumb color
- `--scrollbar-thumb-hover`: Hover state color

**Implementation:**
- Added to all 4 theme presets (default, light, synthwave, terminal)
- Automatically appears in Custom Theme editor (12 color pickers total)
- Applied to all scrollable areas via CSS variables
- Works in both WebKit (Chrome/Safari) and Firefox

## Technical Changes

### Modified Files

**Frontend Components:**
- `TerminalList.tsx`: 
  - Added viewMode prop
  - Implemented collapsible sections
  - Added date-based grouping
  - Gradient color functions
  - Done section with default collapsed state
  
- `App.tsx`:
  - Added view mode state with localStorage persistence
  - View toggle buttons with icons
  - Pass viewMode to TerminalList

- `SettingsModal.tsx`:
  - Added `defaultView` to Settings type
  - UI controls for default view preference
  - Automatically includes new scrollbar theme fields

**Theme System:**
- `theme.ts`:
  - Extended TermTheme with scrollbar properties
  - Updated all presets with scrollbar colors
  - Modified applyTheme() to set CSS variables

- `theme.css`:
  - Added CSS variables for scrollbars
  - Updated all scrollbar selectors to use theme variables
  - Consistent styling across app

### New Functionality

**Collapsible Sections:**
```typescript
const [collapsedSections, setCollapsedSections] = useState<Record<string, boolean>>({
  done: true  // Done collapsed by default
});
```

**View Mode:**
```typescript
type ViewMode = 'scope' | 'date';
const [viewMode, setViewMode] = useState<ViewMode>('scope');
```

**Gradient Colors:**
```typescript
function getTagColor(index: number): string {
  const colors = ['#b4b8c0', '#999da5', '#7e8289'];
  return colors[Math.min(index, 2)];
}

function getProjectColor(index: number, accentColor: string): string {
  const bright = adjustBrightness(accentColor, 15);
  const dark = adjustBrightness(accentColor, -25);
  return [bright, accentColor, dark][Math.min(index, 2)];
}
```

## User Experience Improvements

1. **Cleaner Interface**: Completed todos hidden by default but easily accessible
2. **Flexible Organization**: Switch between scope-based and date-based views
3. **Visual Hierarchy**: Gradient colors help distinguish multiple projects/tags
4. **Theme Customization**: Full control over scrollbar appearance
5. **Persistent Preferences**: View mode and collapse states remembered
6. **Progressive Disclosure**: Collapse sections to focus on what matters

## Testing Notes

- ✅ TypeScript compilation passes
- ✅ All new props properly typed
- ✅ localStorage integration tested
- ✅ Theme presets include scrollbar colors
- ✅ Gradient calculations handle edge cases
- ✅ Collapsible sections maintain state

## Future Enhancements

- Remember individual section collapse states across sessions
- Keyboard shortcuts for view switching (Cmd+1, Cmd+2)
- Animation transitions for section collapse/expand
- Custom sort options within each view mode
- Filter date view by date range
