# Todo App - Complete Feature List

## ✨ Core Features

### Task Management
- ✅ **Create todos** with ⌘N or "New" button
- ✅ **Quick add** with todo.txt format parsing
- ✅ **Check/uncheck** to mark complete
- ✅ **Drag & drop** between sections (Now/Soon/Anytime)
- ✅ **Rich text notes** with TipTap editor
- ✅ **Search** with full-text search (FTS5)

### Organization
- ✅ **3 Sections**: Now, Soon, Anytime
- ✅ **Priority levels**: A (★), B (•), C
- ✅ **Contexts**: @work, @home, @errands
- ✅ **Projects**: +myproject, +todoapp
- ✅ **Tags**: #review, #urgent
- ✅ **Due dates**: Optional date tracking

### Scope Filtering
- ✅ **Custom tabs** (up to 6)
- ✅ **Context filters**: Show only @work, @home, etc.
- ✅ **Project filters**: Show only +myproject items
- ✅ **Combined filters**: Both context AND project
- ✅ **Quick switching**: Click tab to filter instantly

### Display Options
- ✅ **Show/hide completed**: Toggle in settings
- ✅ **Completed styling**: Grayed out with strikethrough
- ✅ **Empty states**: Helpful messages when no items
- ✅ **Progress stats**: % complete, done count, pending count

### Theme Customization
- ✅ **240+ Tailwind colors**: Full color palette
- ✅ **Custom hex input**: Any color like #ffd700
- ✅ **Native color picker**: OS color selector
- ✅ **9 customizable elements**: Background, panel, text, etc.
- ✅ **Live preview**: See changes before applying
- ✅ **Persistent storage**: Theme saved to localStorage

## 🎯 User Interface

### Main View
```
┌─────────────────────────────────────┐
│ Search...              ⌘N New  ⚙   │  ← Actions
├─────────────────────────────────────┤
│ [All] [Work] [Home] [Personal]     │  ← Scope tabs
├─────────────────────────────────────┤
│ Now                          ▼      │  ← Drag & drop
│ ├─ ☐ Review PR @work ★             │
│ └─ ☐ Deploy app +myproject         │
│                                     │
│ Soon                                │
│ └─ ☐ Plan sprint @work             │
│                                     │
│ Anytime                             │
│ ├─ ☐ Read docs #learning           │
│ └─ ☑ Setup project (done)          │  ← Completed
│                                     │
│ 33% complete · 1 done · 2 pending  │  ← Stats
└─────────────────────────────────────┘
```

### Settings Modal
```
┌──────────────────────────────────┐
│ Settings               [Close]   │
├──────────────────────────────────┤
│ [General] [Scope Tabs] [Theme]  │  ← Tabs
├──────────────────────────────────┤
│                                  │
│ General:                         │
│ ☑ Show completed items           │
│                                  │
│ Scope Tabs:                      │
│ [+ Add Tab (3/6)]                │
│ • All (no filters)               │
│ • Work (@work)                   │
│ • Home (@home)                   │
│                                  │
│ Theme:                           │
│ [Open Theme Editor]              │
│                                  │
├──────────────────────────────────┤
│              [Cancel] [Save]     │
└──────────────────────────────────┘
```

## ⌨️ Keyboard Shortcuts

- `⌘N` / `Ctrl+N` - Create new todo
- `ESC` - Close any modal
- `Enter` - Submit forms
- `Enter` in search - Run search

## 🎨 Terminal Theme

### Default Colors
- **Background**: `#0b0e14` (dark blue-black)
- **Panel**: `#0f131a` (darker blue)
- **Text**: `#e6edf3` (light gray)
- **Dim**: `#8b949e` (medium gray)
- **Accent**: `#7aa2f7` (blue)
- **Success**: `#22c55e` (green)
- **Warning**: `#eab308` (yellow/gold)
- **Info**: `#60a5fa` (sky blue)
- **Border**: `#1f2937` (dark gray)

### Visual Elements
- **Monospace font**: Terminal aesthetic
- **Rounded cards**: 12px border radius
- **Subtle shadows**: 0 10px 30px rgba(0,0,0,0.35)
- **Hover effects**: Interactive elements scale/highlight
- **Badges**: Small rounded indicators
- **Checkboxes**: Custom terminal-style

## 💾 Data Persistence

### localStorage
- `todo.settings` - General settings + scope tabs
- `todo.term.theme` - Color theme customization

### SQLite Database
- File: `./data/todo.db`
- Tables: todos, projects, contexts, tags
- FTS5: Full-text search index
- Foreign keys: Relational integrity

### Database Schema
```sql
todos (
  id, title, priority, completed, archived,
  created_at, updated_at, due_at, threshold_at,
  section, notes_md, source_line
)
```

## 🔧 Technical Stack

### Frontend
- **React** 18.2.0
- **TypeScript** 4.6.4
- **Vite** 3.0.7
- **TipTap** 3.7.2 (rich text editor)
- **React Table** 8.21.3

### Backend
- **Go** 1.24.0
- **Wails** v2.10.2
- **SQLite** (modernc.org/sqlite with FTS5)

### Build
- **Platform**: macOS (darwin/arm64)
- **Size**: ~11MB
- **Package**: Native .app bundle

## 📋 Todo.txt Format

Supports standard todo.txt syntax:
```
(A) Review pull request @work +myproject #review
x 2025-01-15 Complete setup @home +personal
Buy groceries @errands due:2025-01-20
```

### Parsing
- `(A)`, `(B)`, `(C)` - Priority
- `@context` - Context tags
- `+project` - Project tags
- `#tag` - General tags
- `due:YYYY-MM-DD` - Due date
- `x` - Completed marker

## 🚀 Usage Examples

### Quick Add
```
⌘N → "Review PR #42 @work +todoapp #review"
→ Creates todo with:
  - Title: "Review PR #42"
  - Context: @work
  - Project: +todoapp
  - Tag: #review
  - Section: Anytime (default)
```

### Scope Filtering
```
Settings → Scope Tabs:
  Tab: "Work Stuff"
  Context: work
  Project: (empty)

Click "Work Stuff" tab
→ Shows only items with @work
→ All sections filtered
→ Drag & drop still works
```

### Theme Customization
```
Settings → Theme → Open Theme Editor
→ Click "Accent" field
→ Choose #ffd700 (gold)
→ Apply Theme
→ Links and highlights now gold
```

## 🎯 Best Practices

1. **Use scope tabs** for work/life separation
2. **Drag to Now** for current focus
3. **Drag to Soon** for next up
4. **Leave in Anytime** for backlog
5. **Hide completed** for clean views (default)
6. **Show completed** to review accomplishments
7. **Use priority** (A/B/C) for importance
8. **Add notes** for detailed context
9. **Combine filters** for laser focus
10. **Customize theme** for personal aesthetic

Your todo app is production-ready! 🎉
