# Todo App - Quick Start Guide

## ✅ Your app is now working!

### Running the App

**Development Mode (recommended for testing):**
```bash
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
wails dev
```

**Production Build:**
```bash
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
wails build
open build/bin/todo-app.app
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
✅ Theme customization  
✅ Keyboard shortcuts (⌘N for new)  

## Database

Data is stored in: `./data/todo.db`

The SQLite database is created automatically on first run.

## Technology Stack

- **Frontend:** React + TypeScript + Vite
- **Backend:** Go 1.24
- **Database:** SQLite (modernc.org/sqlite with FTS5)
- **Framework:** Wails v2
- **UI:** Custom terminal-style theme with shadcn patterns

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
Delete `./data/todo.db` to reset the database.

## Next Steps

1. **Run the app:** `wails dev`
2. **Create a todo:** Press ⌘N or click "New"
3. **Search:** Use the search box with filters like `pri:A` or `+project`
4. **Customize:** Click "Theme" to change colors

Enjoy your todo app! 🎉
