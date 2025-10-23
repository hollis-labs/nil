# Task 04: Quick Add Modal UI Overhaul

## Overview
Restructure the Quick Add Todo modal with a modern tabbed interface, improved input controls, and better visual design.

## What Needs to Be Done

### 1. Priority: Convert to Radio Chips
**File**: `frontend/src/components/EditTodoModal.tsx`

**Current**: Dropdown with A/B/C/None

**New Design**:
- Radio button group displayed as chips
- Options: "High" (A), "Med" (B), "Low" (C), "None"
- Single selection only
- Visual: chip style like badges, selected state highlighted
- Horizontal layout

**Implementation**:
```tsx
<div>
  <label>Priority</label>
  <div style={{ display: 'flex', gap: '8px' }}>
    {['None', 'Low', 'Med', 'High'].map(level => (
      <button
        key={level}
        type="button"
        className={`badge ${priority === level ? 'success' : ''}`}
        onClick={() => setPriority(level === 'None' ? '' : level)}
      >
        {level}
      </button>
    ))}
  </div>
</div>
```

### 2. Due Date: Convert to Chip with Picker
**File**: `frontend/src/components/EditTodoModal.tsx`

**Current**: Date input always visible, defaults to today

**New Design**:
- Chip button showing "Set Due Date" or selected date
- Click opens native date picker
- No default date set
- Clear button (X) when date is set

**Implementation**:
```tsx
<div>
  <label>Due Date</label>
  {!due ? (
    <button type="button" className="badge" onClick={() => dateInputRef.current?.showPicker()}>
      Set Due Date
    </button>
  ) : (
    <div style={{ display: 'flex', gap: '4px' }}>
      <span className="badge info">{new Date(due).toLocaleDateString()}</span>
      <button type="button" className="badge" onClick={() => setDue('')}>✕</button>
    </div>
  )}
  <input
    ref={dateInputRef}
    type="date"
    style={{ display: 'none' }}
    value={due}
    onChange={(e) => setDue(e.target.value)}
  />
</div>
```

### 3. Grid Layout for Context/Project/Tags
**File**: `frontend/src/components/EditTodoModal.tsx`

**Current**: Vertical stack

**New Design**:
- CSS Grid: 2 columns × 2 rows
- Items: Context, Project, Tags, [Empty for future]
- Equal width columns
- Gap between items

**Implementation**:
```css
display: grid;
grid-template-columns: 1fr 1fr;
gap: 12px;
```

### 4. Tab Layout: Meta & Description
**File**: `frontend/src/components/EditTodoModal.tsx`

**Structure**:
```
Header (Fixed)
├─ Title: "Quick Add"
├─ Template Picker (right of title)
└─ Tab Buttons: [Meta] [Description]

Scrollable Body
├─ Tab: Meta
│   ├─ Task Input
│   ├─ Priority (radio chips)
│   ├─ Due Date (chip)
│   ├─ Grid (2x2):
│   │   ├─ Contexts
│   │   ├─ Projects
│   │   ├─ Tags
│   │   └─ [Empty]
│   └─ Use Defaults checkbox (if applicable)
│
└─ Tab: Description
    └─ TipTap Editor (full height with margin)

Footer (Fixed)
├─ Clear, Cancel (left)
└─ Create/Save (right)
```

**Implementation**:
- State: `const [activeTab, setActiveTab] = useState<'meta' | 'description'>('meta')`
- Conditional render based on activeTab
- Description tab: editor takes full height minus margin

### 5. Description Editor Improvements (DONE!)
**File**: `frontend/src/components/EditTodoModal.tsx`

**Changes**:
- **Horizontal scrollbar**: Add `overflow-x: hidden` to editor container
- **Vertical scrollbar**: Use CustomScrollbar with muted theme colors
- **Bullet padding**: Reduce left padding in `.tiptap-editor ul` by 25%
- **Font size/family**: Match main todo list display
  - Font family: `ui-monospace, SFMono-Regular, Menlo...`
  - Font size: `13px` (match main list)

**Custom Scrollbar for Editor**:
```tsx
<CustomScrollbar style={{ height: '200px' }}>
  <EditorContent editor={editor} />
</CustomScrollbar>
```

### 6. Template Picker Integration
**File**: `frontend/src/components/EditTodoModal.tsx`

**Location**: Header, right side next to "Task" label

**Layout**:
```
┌─────────────────────────────────────┐
│ Task          [Template ▼] [💾]     │
└─────────────────────────────────────┘
```

**Components**:
- Template dropdown (TemplatePickerCombobox)
- Save button (💾 icon or "Save")
- Positioned with flexbox: space-between

## Technical Notes
- Maintain all existing functionality
- Ensure keyboard shortcuts still work
- Test in both create and edit modes
- Preserve form state when switching tabs
- Grid should be responsive (stack on small screens if needed)

## CSS/Style Updates
**File**: `frontend/src/components/EditTodoModal.tsx` (inline styles)

- Add editor container styles
- Add tab button styles (active/inactive states)
- Add grid layout
- Add chip button styles for priority/date

**File**: `frontend/src/style.css` (if needed)

- TipTap bullet list padding adjustment
- Editor scrollbar customization

## Testing Checklist
- [ ] Tabs switch correctly
- [ ] Form state preserved across tab switches
- [ ] Priority chips work (single select)
- [ ] Due date chip opens picker
- [ ] Due date can be cleared
- [ ] Grid layout displays correctly
- [ ] Description scrollbar is custom styled
- [ ] Description horizontal scroll disabled
- [ ] Description font matches main list
- [ ] Bullet points have reduced padding
- [ ] Template picker appears in header
- [ ] All existing features still work
