# NIL Quick Start Guide

## Running The App

### Running the App

**Development Mode (recommended for testing):**
```bash
cd /Users/chrispian/Projects-apps/nil
wails dev
```

**Production Build:**
```bash
cd /Users/chrispian/Projects-apps/nil
wails build
open build/bin/NIL.app
```

## About the "Private APIs" Warning

You'll see this warning when building:

```
WARNING: This darwin build contains the use of private APIs. 
This will not pass Apple's AppStore approval process.
```

**This is normal and expected!** 

- ✅ The app works perfectly for personal use
- ✅ You can distribute it to others outside the App Store
- ❌ It won't pass App Store review (requires code signing + entitlements)

If you need App Store distribution:
1. Enroll in Apple Developer Program ($99/year)
2. Configure code signing in Wails
3. Use official APIs only (requires Wails configuration)

For most developers, this warning can be safely ignored.

## Features Working

✅ Full-text search (FTS5)  
✅ Todo.txt format parsing  
✅ Projects, contexts, tags  
✅ Priority levels  
✅ Due dates  
✅ Rich text notes (TipTap editor)  
✅ Theme system  
✅ Keyboard shortcuts (⌘N for new)  

## Database

The default vault database is created automatically on first run.

Default locations:
- macOS: `~/Library/Application Support/Nil`
- Linux: `~/.local/share/nil`
- Windows: `%APPDATA%/Nil/data`

## Technology Stack

- **Frontend:** React + TypeScript + Vite
- **Backend:** Go 1.24
- **Database:** SQLite (modernc.org/sqlite with FTS5)
- **Framework:** Wails v2
- **UI:** Custom terminal-style theme with CSS variables

## Troubleshooting

### "Port already in use"
```bash
pkill -f "wails dev"
pkill -f "vite"
```

### "Cannot find module" errors
```bash
cd frontend
npm install
```

### Database errors
Move or delete the active vault database directory to reset local data.

## Next Steps

1. **Run the app:** `wails dev`
2. **Create an item:** Press `Cmd/Ctrl+N`
3. **Search:** Use the search box with filters like `pri:A` or `+project`
4. **Open settings:** Click the gear icon to review tabs and app behavior
