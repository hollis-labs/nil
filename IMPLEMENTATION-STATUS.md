# PLANCK Todo App - Implementation Status

## Completed Features ✅

### Session 1: Core Fixes & Tab-to-Query Conversion
- [x] Fixed FTS5 search syntax error
- [x] Added wildcard prefix matching for search
- [x] Converted ScopeTab from context/project to unified query field
- [x] Updated SettingsModal UI to use single query input
- [x] Merged tab queries with main search query in App.tsx
- [x] Removed debug console.log statements
- [x] Fixed modal structure (header, scrollable body, footer)

### Session 2: Settings UI & Cloud Sync Foundation
- [x] Made settings modal fixed height (650px)
- [x] Fixed header and footer in settings modal
- [x] Made import/export buttons side-by-side (50/50)
- [x] Enabled SQLite WAL mode for better cloud sync
- [x] Added Store.Close() method
- [x] Made database path read-only in UI
- [x] Added cloud sync setup instructions (collapsible)
- [x] Created CLOUD-SYNC-GUIDE.md documentation
- [x] Created DATABASE-MIGRATION-PLAN.md for future feature

## In Progress 🚧

### Tab Drag-and-Drop Reordering
- [ ] Not yet started (was next on the list)

## Planned Features 📋

### MVP Phase
- [ ] Drag-and-drop tab reordering in Settings
- [ ] Better error handling/messaging
- [ ] Loading states for async operations
- [ ] Keyboard shortcuts documentation
- [ ] Initial user onboarding

### Post-MVP (Nice to Have)
- [ ] Database migration UI (see DATABASE-MIGRATION-PLAN.md)
- [ ] Automatic backup before major operations
- [ ] Cloud sync status indicator
- [ ] Conflict resolution UI
- [ ] Custom keyboard shortcuts
- [ ] Dark/light mode auto-switch based on system
- [ ] Todo templates
- [ ] Recurring todos
- [ ] Calendar view
- [ ] Statistics/analytics dashboard

## Known Issues 🐛

None currently!

## Documentation 📚

Created:
- [x] CLOUD-SYNC-GUIDE.md - Complete guide for syncing via Dropbox/iCloud/Google Drive
- [x] DATABASE-MIGRATION-PLAN.md - Comprehensive plan for future UI-based database relocation

To Create:
- [ ] USER-GUIDE.md - Complete user manual
- [ ] KEYBOARD-SHORTCUTS.md - All available shortcuts
- [ ] CONTRIBUTING.md - For future contributors
- [ ] CHANGELOG.md - Version history

## Technical Debt 💳

1. TypeScript module resolution warnings (non-blocking, build succeeds)
2. Hardcoded database path (will be resolved when config system is implemented)
3. Alert() and confirm() dialogs (should use custom modal components)
4. No automated tests yet (should add Jest/Testing Library)

## Next Session Goals 🎯

1. Implement drag-and-drop tab reordering
2. Replace browser alert/confirm with custom modals
3. Add loading states to async operations
4. Start on user documentation

## Architecture Notes 🏗️

### Current Stack
- **Frontend**: React + TypeScript + Vite
- **Backend**: Go + Wails v2
- **Database**: SQLite with FTS5 + WAL mode
- **Styling**: Custom CSS with CSS variables for theming

### Database Schema
- todos table (main data)
- projects, contexts, tags tables (normalized)
- todos_fts (FTS5 virtual table for full-text search)

### Key Files
- `frontend/src/pages/App.tsx` - Main app component
- `frontend/src/components/SettingsModal.tsx` - Settings UI
- `frontend/src/lib/query.ts` - Query parser
- `app.go` - Backend API methods
- `store/store.go` - Database operations

## How to Move Database Manually (Until UI Feature is Ready)

### iCloud Setup
```bash
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
mv data ~/Library/Mobile\ Documents/com~apple~CloudDocs/planck-todo-data
ln -s ~/Library/Mobile\ Documents/com~apple~CloudDocs/planck-todo-data data
```

### Dropbox Setup
```bash
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
mv data ~/Dropbox/planck-todo-data
ln -s ~/Dropbox/planck-todo-data data
```

See CLOUD-SYNC-GUIDE.md for full details.

## Build Commands

```bash
# Frontend only
cd frontend
npm run build

# Full app (dev mode)
wails dev

# Full app (production)
wails build
```

## Questions or Issues?

Refer to:
- CLOUD-SYNC-GUIDE.md - For cloud sync questions
- DATABASE-MIGRATION-PLAN.md - For future migration feature details
- Create GitHub issue - For bugs or feature requests
