# PLANCK Alpha Release - Ready for Testing!

## ✅ Completed Features

### Core App
- ✅ Todo.txt format support
- ✅ Search & filtering (keywords, +project, @context, #tag, pri:A, due dates)
- ✅ Negative filters (exclude with -)
- ✅ Three scope sections (Now/Soon/Anytime)
- ✅ Session Context for focused work
- ✅ Custom tabs for saved views
- ✅ Scope vs Date view modes
- ✅ Priority system (A/B/C)
- ✅ Due dates and threshold dates
- ✅ Markdown notes on any todo
- ✅ Right-click context menu
- ✅ Keyboard shortcuts (⌘N for quick add, Esc to clear)

### UI/UX
- ✅ Chromeless frameless window
- ✅ Transparent background with floating panel effect
- ✅ Drop shadow for depth
- ✅ Draggable logo area
- ✅ 20+ beautiful themes
- ✅ Dark mode optimized
- ✅ Smooth animations

### Onboarding
- ✅ **Alpha warning modal** with animated bee
- ✅ First-run database location setup
- ✅ Tutorial system with 19 rich examples
- ✅ Organized into #now, #soon, #anytime
- ✅ All tutorials have markdown notes
- ✅ "Remove Tutorial" button when ready

### Polish
- ✅ App named "PLANCK" in menu bar
- ✅ Help modal with App & Todo.txt guides
- ✅ Quit button with confirmation
- ✅ Settings panel
- ✅ iCloud sync instructions

## 🚫 Known Limitations (Hidden/Disabled)

- ❌ Import/Export (hidden - not yet working)

## 📍 File Locations

**Built App:**
```
/Users/chrispian/Downloads/todo-app-starter/todo-app/build/bin/PLANCK.app
```

**Config:**
```
~/.config/planck/config.json
```

**Default Database:**
```
~/Library/Application Support/planck/todo.db
```

## 🧪 Testing Checklist

### First Launch Test
- [ ] Delete `~/.config/planck/config.json`
- [ ] Delete localStorage: Open Dev Tools → Application → Local Storage → Clear
- [ ] Launch PLANCK.app
- [ ] **Alpha warning** should appear with spinning bee
- [ ] Click "OK"
- [ ] **Welcome dialog** should appear
- [ ] Choose "Default Location" or test custom path
- [ ] **Demo data prompt** should appear
- [ ] Click "Yes, Add Tutorial"
- [ ] Should see 19 tutorial todos organized into sections

### Core Features Test
- [ ] Press ⌘N to add a todo
- [ ] Search for `#now` - should filter
- [ ] Try `+tutorial @getting-started` search
- [ ] Try negative filter: `+tutorial -#now`
- [ ] Click any todo to view markdown notes
- [ ] Right-click a todo → Move to Soon
- [ ] Click 🎯 icon → Set Session Context
- [ ] Toggle between Scope and Date views
- [ ] Open Settings → Try different themes
- [ ] Click ? icon → View Help
- [ ] Click "Remove Tutorial" button

### UI Test
- [ ] Drag window by PLANCK logo area
- [ ] Resize window (min 800x750)
- [ ] Check drop shadow visible
- [ ] Hover over buttons (should highlight)
- [ ] Click power button → Confirm quit works

## 📤 Sharing with Friends

### Quick Distribution
```bash
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
./build-for-friends.sh
```

Creates: `PLANCK-macOS.zip`

### Installation Instructions for Friends
1. Download PLANCK-macOS.zip
2. Unzip
3. Drag PLANCK.app to Applications folder
4. **First launch:** Right-click → Open (bypass Gatekeeper)
5. System Settings → Privacy & Security → "Open Anyway"
6. Future launches: Normal double-click

## ⚠️ Alpha Warning to Share

> PLANCK is in early alpha. Backup your data often. The database is stored as a SQLite file - you can copy it manually for backups. iCloud sync is supported but use at your own risk. Not responsible for data loss.

## 🐛 Known Issues to Watch For

- First-time Gatekeeper warnings (expected, unsigned app)
- Wails runtime messages in console (normal)
- Window might not remember position between launches
- No auto-updates yet

## 💡 Feedback to Request

- Is the onboarding clear?
- Are the tutorials helpful?
- Is todo.txt syntax intuitive?
- Does search work as expected?
- Are the themes appealing?
- Is the UI responsive?
- Any crashes or bugs?
- Feature requests?

## 🎯 Ready for Alpha!

The app is **polished and ready** for early alpha testing!

**To launch:**
```bash
open /Users/chrispian/Downloads/todo-app-starter/todo-app/build/bin/PLANCK.app
```

Or drag to /Applications for permanent install.
