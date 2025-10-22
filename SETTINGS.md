# Settings Modal - Complete Configuration System

## Overview

The new Settings modal provides comprehensive customization with three tabs:
- **General** - Display options
- **Scope Tabs** - Custom filter views
- **Theme** - Color customization

## Features

### ⚙️ Settings Button
- Replaced "Theme" button with gear icon (⚙)
- Cleaner, more recognizable settings interface
- Opens tabbed modal for all configuration

### 📋 General Tab

**Show Completed Items**
- Checkbox to toggle completed items visibility
- When OFF (default): Completed items hidden from sections
- When ON: Completed items shown in sections (grayed out)
- Affects all views immediately

### 🏷️ Scope Tabs

**What Are Scope Tabs?**
- Custom filter views that appear below the search bar
- Filter todos by context (@work, @home) or project (+myproject)
- Click a tab to instantly filter your view
- Up to 6 custom tabs

**Creating Tabs:**
1. Click "Add Tab" button (shows count: 3/6)
2. Set tab label (e.g., "Work", "Home", "Personal")
3. Add context filter (e.g., "work" for @work items)
4. Add project filter (e.g., "myproject" for +myproject items)
5. Leave both filters empty for "All" view

**Tab Configuration:**
- **Tab Label**: Display name (e.g., "Work", "Home")
- **Context Filter**: Matches @context (without the @)
- **Project Filter**: Matches +project (without the +)
- **Delete**: Red ✕ button (requires at least 1 tab)

**Example Tabs:**
```
Label: All
Context: (empty)
Project: (empty)
→ Shows all todos

Label: Work
Context: work
Project: (empty)
→ Shows only @work todos

Label: Home Projects
Context: home
Project: (empty)
→ Shows only @home todos

Label: MyApp Dev
Context: (empty)
Project: myapp
→ Shows only +myapp todos

Label: Work Urgent
Context: work
Project: urgent
→ Shows todos with BOTH @work AND +urgent
```

### 🎨 Theme Tab

**Quick Access to Theme Editor**
- "Open Theme Editor" button
- Same powerful color picker as before
- 240+ Tailwind colors + custom hex input
- Opens in nested modal

## How Scope Filtering Works

### Active Tab Filter
When you click a scope tab:
1. Filter applied to all sections (Now/Soon/Anytime)
2. Only matching todos appear
3. Search still works within filtered view
4. Drag & drop works within filtered view

### Combined Filters
- Context AND Project filters combine (both must match)
- Empty filter = no restriction on that dimension
- All sections respect the active tab filter

### Example Flow
```
1. You have 50 todos total
2. Click "Work" tab (context: work)
3. Now see only 15 @work todos
4. Drag one from "Anytime" to "Now"
5. Search for "review" → only @work todos matching "review"
```

## Settings Persistence

All settings saved to `localStorage`:
- `todo.settings` - General settings + scope tabs
- `todo.term.theme` - Color theme
- Persists across app restarts

## UI Layout

```
┌─────────────────────────────────────────┐
│  Search...                  ⌘N  ⚙      │
├─────────────────────────────────────────┤
│  [All] [Work] [Home]  <-- Scope Tabs    │
├─────────────────────────────────────────┤
│                                         │
│  Now Section                            │
│  ├─ Todo 1                              │
│  └─ Todo 2                              │
│                                         │
│  Soon Section                           │
│  └─ Todo 3                              │
│                                         │
│  Anytime Section                        │
│  ├─ Todo 4                              │
│  └─ Todo 5                              │
│                                         │
└─────────────────────────────────────────┘
```

## Settings Modal Structure

```
┌───────────────────────────────────────┐
│ Settings                      [Close] │
├───────────────────────────────────────┤
│ [General] [Scope Tabs] [Theme]        │
├───────────────────────────────────────┤
│                                       │
│  (Content for active tab)             │
│                                       │
│                                       │
├───────────────────────────────────────┤
│                    [Cancel] [Save]    │
└───────────────────────────────────────┘
```

## Keyboard Shortcuts

- `⌘N` - New todo (unchanged)
- Click outside modal - Close settings

## Best Practices

### Organizing with Scope Tabs
1. **All Tab**: Always keep one "All" tab with no filters
2. **Context-Based**: Create tabs for @work, @home, @errands
3. **Project-Based**: Create tabs for active projects
4. **Combined**: Mix context + project for focused views

### Example Setup
```
Tab 1: All (no filters)
Tab 2: Work (@work)
Tab 3: Home (@home)
Tab 4: Current Project (+myapp, @work)
Tab 5: Personal (@personal)
Tab 6: Errands (@errands)
```

## Technical Details

**Settings Structure:**
```typescript
{
  showCompleted: boolean;
  tabs: Array<{
    id: string;
    label: string;
    context?: string;
    project?: string;
  }>;
}
```

**Default Settings:**
- Show Completed: false
- Tabs: All, Work (@work), Home (@home)

**Filtering Logic:**
1. Apply active tab context filter
2. Apply active tab project filter
3. Apply show completed setting
4. Apply search query (if any)
5. Group into sections (Now/Soon/Anytime)

The settings system provides powerful organization without complexity!
