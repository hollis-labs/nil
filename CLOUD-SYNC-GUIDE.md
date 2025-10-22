# Cloud Sync Guide for PLANCK Todo App

## Overview

PLANCK now uses SQLite WAL (Write-Ahead Logging) mode, which is optimized for cloud sync scenarios. This allows you to store your todo database in Dropbox, iCloud Drive, or Google Drive and access it from multiple computers.

## ⚠️ Important Rules

1. **Never run the app on multiple computers simultaneously**
2. **Always wait for cloud sync to complete before switching computers**
3. **Always quit the app cleanly (don't force quit)**

## Quick Setup

### Option 1: Dropbox

```bash
# 1. Quit the app completely

# 2. Move database to Dropbox
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
mv data ~/Dropbox/planck-todo-data

# 3. Create symbolic link
ln -s ~/Dropbox/planck-todo-data data

# 4. Verify the link
ls -la data/

# 5. Restart the app
```

### Option 2: iCloud Drive

```bash
# 1. Quit the app completely

# 2. Move database to iCloud
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
mv data ~/Library/Mobile\ Documents/com~apple~CloudDocs/planck-todo-data

# 3. Create symbolic link
ln -s ~/Library/Mobile\ Documents/com~apple~CloudDocs/planck-todo-data data

# 4. Verify the link
ls -la data/

# 5. Restart the app
```

### Option 3: Google Drive

```bash
# 1. Quit the app completely

# 2. Move database to Google Drive
cd /Users/chrispian/Downloads/todo-app-starter/todo-app
mv data ~/Google\ Drive/My\ Drive/planck-todo-data

# 3. Create symbolic link
ln -s ~/Google\ Drive/My\ Drive/planck-todo-data data

# 4. Verify the link
ls -la data/

# 5. Restart the app
```

## How It Works

### WAL Mode Benefits

- **Three files**: `todo.db`, `todo.db-wal`, `todo.db-shm`
- **Better concurrency**: Readers don't block writers
- **Safer for cloud sync**: Checkpoint operations are atomic
- **Faster writes**: Changes written to WAL file first

### Cloud Sync Behavior

1. You make changes on Computer A
2. App writes to `todo.db-wal` file
3. On checkpoint (periodically), WAL merges into `todo.db`
4. Cloud service syncs all three files
5. You open app on Computer B - sees fully synced state

## Troubleshooting

### "Database is locked" error

**Cause**: App is still running on another computer or cloud sync is incomplete

**Solution**:
1. Quit app on all computers
2. Wait 1-2 minutes for cloud sync to complete
3. Check cloud service web interface to verify sync
4. Restart app on one computer

### Sync conflict files appear

**Cause**: App was opened on two computers before sync completed

**Solution**:
1. Quit app on all computers
2. Identify which conflict file is newer (check timestamps)
3. Keep the newer file, rename it to `todo.db`
4. Delete old `todo.db` and all `-wal`/`-shm` files
5. Restart app

### Data loss concerns

**Always keep backups!** Use the Export feature regularly:

Settings → Import/Export → Export to todo.txt

## Technical Details

### Database Files

- `todo.db`: Main database file
- `todo.db-wal`: Write-Ahead Log (recent changes)
- `todo.db-shm`: Shared memory file (index)

### WAL Checkpointing

WAL automatically checkpoints (merges WAL → main DB) when:
- WAL file reaches ~1000 pages (~4MB)
- App closes cleanly
- After certain operations

### Cloud Service Compatibility

| Service | Compatibility | Notes |
|---------|---------------|-------|
| Dropbox | ✅ Excellent | Best for cross-platform |
| iCloud Drive | ✅ Good | Best for Mac-only |
| Google Drive | ✅ Good | Stream may delay sync |
| OneDrive | ⚠️ Caution | May have lock issues |

## Best Practices

1. **Always export before major changes**: Use todo.txt export as backup
2. **Wait for sync indicator**: Check your cloud service's sync status before switching computers
3. **Close cleanly**: Always use app menu to quit, not force quit
4. **One computer at a time**: Develop a habit - close on one, open on another
5. **Regular backups**: Export weekly to a separate location

## Future Improvements

For MVP+1, we're considering:
- Automatic export on close
- Visual sync status indicator
- Built-in conflict resolution
- Cloud service integration
- Automatic database relocation tool

## Questions?

See the in-app Settings → General → Database Location for more information.
