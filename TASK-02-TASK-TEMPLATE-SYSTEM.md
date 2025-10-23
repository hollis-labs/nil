# Task 02: Task Template System

## Overview
Create a template system that allows users to save and reuse common task configurations (tags, projects, contexts, priority). Templates can be applied in Quick Add modal and Session Context modal.

## What Needs to Be Done

### 1. Define Template Type & Storage
**File**: `frontend/src/lib/templates.ts`

**Template Type**:
```tsx
type TaskTemplate = {
  id: string;
  name: string;
  contexts: string[];
  projects: string[];
  tags: string[];
  priority?: 'A' | 'B' | 'C';
  createdAt: string;
}
```

**Storage Functions**:
- `getTemplates(): TaskTemplate[]` - Load from localStorage
- `saveTemplate(template: Omit<TaskTemplate, 'id' | 'createdAt'>): void`
- `updateTemplate(id: string, updates: Partial<TaskTemplate>): void`
- `deleteTemplate(id: string): void`

**Storage Key**: `'planck.taskTemplates'`

### 2. Create TemplatePickerCombobox Component
**File**: `frontend/src/components/TemplatePickerCombobox.tsx`

**Features**:
- Combobox with search/filter
- List of saved templates
- Select template → fires onSelect callback
- Shows template details on hover (tags, projects, contexts, priority)
- Empty state when no templates

**Props**:
```tsx
type Props = {
  onSelect: (template: TaskTemplate) => void;
  currentValues?: {
    contexts: string[];
    projects: string[];
    tags: string[];
    priority?: string;
  };
  onSaveAsTemplate: () => void; // Opens save dialog
}
```

### 3. Create Template Save Dialog
**File**: `frontend/src/components/TemplateSaveDialog.tsx`

**Features**:
- Modal dialog
- Input for template name
- Preview of what will be saved (tags, projects, contexts, priority)
- Save button
- Cancel button

### 4. Integrate into EditTodoModal
**File**: `frontend/src/components/EditTodoModal.tsx`

**Location**: Beside "Task" label at top

**Features**:
- Template picker dropdown
- Save button beside picker
- When template selected: merge with current values
- Save button opens dialog to save current values as new template

### 5. Create Template Management UI (Optional)
**File**: Could be tab in SettingsModal

**Features**:
- List all templates
- Edit template (name, values)
- Delete template
- Duplicate template

## Technical Notes
- Templates are stored in localStorage
- IDs generated with `Date.now().toString()` or uuid
- Merge strategy: append arrays, override priority
- Consider template categories in future (work, personal, etc.)

## Testing
- Create template from Quick Add modal
- Apply template in Quick Add modal
- Edit existing template
- Delete template
- Test merge behavior (template + manual input)
