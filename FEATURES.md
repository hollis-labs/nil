# New Features - Section-Based Organization

## What's New

### Three-Section Display
Todos are now organized into three draggable sections:

- **Now** - High priority, immediate tasks
- **Soon** - Tasks coming up
- **Anytime** - Backlog items (default)

### Drag & Drop
- Drag any todo between sections
- Simply drag and drop to reorganize priorities
- Visual feedback during drag operation
- "Drop items here" placeholder for empty sections

### Improved Quick Add
- Priority dropdown now matches terminal theme
- Styled select with custom arrow
- No default due date (optional field)
- Dark theme option styling

## How It Works

### Creating Todos
1. Press `⌘N` or click "New"
2. Enter your task
3. Optionally set priority (A/B/C)
4. Optionally set due date
5. New todos default to "Anytime" section

### Organizing
- **Drag & Drop**: Click and drag any todo to move between sections
- **Priority Badges**: 
  - ★ for Priority A (high)
  - • for Priority B (medium)
- **Sections**:
  - **Now**: Current focus
  - **Soon**: Next up
  - **Anytime**: Backlog

### Database Schema
The `todos` table now includes:
- `section` column (TEXT, defaults to 'anytime')
- Values: 'now', 'soon', 'anytime'

## Running the Updated App

```bash
cd /Users/chrispian/Downloads/todo-app-starter/todo-app

# Development
wails dev

# Production
wails build
open build/bin/todo-app.app
```

## Technical Details

### Backend Changes
- Added `Section` field to `Todo` model
- Updated schema with `section` column
- Modified Create/Update operations to handle section
- Section defaults to 'anytime' if not specified

### Frontend Changes
- Section-based grouping in `TerminalList`
- Drag and drop event handlers
- Visual drop zones for each section
- Custom styled priority dropdown
- Completed items excluded from section display

### Styling
- Terminal theme maintained throughout
- Custom select dropdown with themed arrow
- Hover effects on draggable items
- Empty section placeholders

## Tips
- Drag todos to "Now" for immediate focus
- Use "Soon" for planning upcoming work
- Keep "Anytime" as your backlog
- Completed items are hidden from sections but counted in stats
