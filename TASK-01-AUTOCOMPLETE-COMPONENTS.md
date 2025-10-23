# Task 01: Reusable Autocomplete Input Components

## Overview
Create reusable autocomplete input components for Tags, Projects, and Contexts that can be used throughout the app. These will replace the current TagsInput component and provide a consistent UX.

## What Needs to Be Done

### 1. Create Base AutocompleteInput Component
**File**: `frontend/src/components/AutocompleteInput.tsx`

**Features**:
- Accepts a list of suggestions
- Filters suggestions as user types (no prefix required - @, +, # handled by parent)
- Shows dropdown with filtered suggestions
- Tab/Enter to select current suggestion
- Supports adding multiple values (chip display)
- Configurable prefix for display (#, @, +)
- Click outside to close dropdown
- Arrow keys to navigate suggestions

**Props**:
```tsx
type Props = {
  values: string[];
  onValuesChange: (values: string[]) => void;
  suggestions: string[];
  placeholder: string;
  prefix?: string; // For display only
  label?: string;
}
```

### 2. Create Specific Components
**Files**:
- `frontend/src/components/TagsAutocomplete.tsx`
- `frontend/src/components/ProjectsAutocomplete.tsx`
- `frontend/src/components/ContextsAutocomplete.tsx`

Each wraps `AutocompleteInput` with appropriate prefix and fetches its own suggestions from `Backend.GetFilters()`.

### 3. Replace Existing Usage
**Files to Update**:
- `frontend/src/components/EditTodoModal.tsx` - Replace TagsInput with new components
- Future: Session Context Modal
- Future: Task Template Modal

## Technical Notes
- Use existing `TodoTitleInput.tsx` as reference for autocomplete behavior
- Maintain focus management for keyboard navigation
- Ensure dropdown positioning works in modals
- Style to match theme system

## Testing
- Test in EditTodoModal
- Test with keyboard navigation (Tab, Enter, Arrow keys)
- Test filtering with partial matches
- Test adding multiple items
- Test removing items (backspace or click X)
