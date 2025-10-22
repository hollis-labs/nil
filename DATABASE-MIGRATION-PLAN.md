# Database Migration Feature - Implementation Plan

## Overview

This document outlines the complete implementation plan for allowing users to relocate their PLANCK todo database through the UI, with automatic migration and app restart.

## User Experience Flow

### Happy Path
1. User opens Settings → General → Database Location
2. Clicks "Select New Location" button
3. Native directory picker opens
4. User selects destination folder (e.g., `~/Dropbox/planck-data`)
5. App shows confirmation dialog:
   ```
   Move database to new location?
   
   Current: /Users/username/planck/data
   New: /Users/username/Dropbox/planck-data
   
   The app will:
   • Close the database safely
   • Copy all data to the new location
   • Update configuration
   • Restart automatically
   
   [Cancel] [Move & Restart]
   ```
6. User clicks "Move & Restart"
7. App performs migration (with progress indicator)
8. App restarts automatically
9. Success notification shows new location

### Error Paths
- **Insufficient permissions**: Show error, suggest different location
- **Not enough disk space**: Calculate size, show error with space needed
- **Migration fails mid-copy**: Rollback, keep using original location
- **New location already has data**: Offer to merge or cancel

## Technical Architecture

### 1. Configuration System

**File Location**: `~/.planck/config.json` (user's home directory)

**Structure**:
```json
{
  "version": "1.0",
  "database": {
    "path": "/Users/username/Dropbox/planck-data",
    "last_migrated": "2025-10-22T14:30:00Z",
    "migration_history": [
      {
        "from": "./data",
        "to": "/Users/username/Dropbox/planck-data",
        "timestamp": "2025-10-22T14:30:00Z",
        "method": "copy"
      }
    ]
  },
  "app": {
    "theme": "default",
    "last_backup": "2025-10-22T10:00:00Z"
  }
}
```

**Config Package** (`config/config.go`):
```go
package config

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Config struct {
    Version  string           `json:"version"`
    Database DatabaseConfig   `json:"database"`
    App      AppConfig        `json:"app"`
}

type DatabaseConfig struct {
    Path             string             `json:"path"`
    LastMigrated     string             `json:"last_migrated,omitempty"`
    MigrationHistory []MigrationRecord  `json:"migration_history,omitempty"`
}

type MigrationRecord struct {
    From      string `json:"from"`
    To        string `json:"to"`
    Timestamp string `json:"timestamp"`
    Method    string `json:"method"`
}

type AppConfig struct {
    Theme      string `json:"theme"`
    LastBackup string `json:"last_backup,omitempty"`
}

func Load() (*Config, error) {
    configDir := filepath.Join(os.Getenv("HOME"), ".planck")
    configPath := filepath.Join(configDir, "config.json")
    
    // Create default config if doesn't exist
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return createDefault(configDir, configPath)
    }
    
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }
    
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    
    return &cfg, nil
}

func (c *Config) Save() error {
    configDir := filepath.Join(os.Getenv("HOME"), ".planck")
    configPath := filepath.Join(configDir, "config.json")
    
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(configPath, data, 0644)
}

func createDefault(configDir, configPath string) (*Config, error) {
    if err := os.MkdirAll(configDir, 0755); err != nil {
        return nil, err
    }
    
    cfg := &Config{
        Version: "1.0",
        Database: DatabaseConfig{
            Path: "./data",
        },
        App: AppConfig{
            Theme: "default",
        },
    }
    
    if err := cfg.Save(); err != nil {
        return nil, err
    }
    
    return cfg, nil
}
```

### 2. Migration Service

**Migration Package** (`migration/migrate.go`):
```go
package migration

import (
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
    
    "todo-app/config"
    "todo-app/store"
)

type MigrationService struct {
    store  *store.Store
    config *config.Config
}

type MigrationProgress struct {
    Stage       string  // "validating", "copying", "verifying", "updating", "complete"
    Percent     int     // 0-100
    Message     string
    Error       error
}

type ProgressCallback func(MigrationProgress)

func NewMigrationService(st *store.Store, cfg *config.Config) *MigrationService {
    return &MigrationService{
        store:  st,
        config: cfg,
    }
}

func (m *MigrationService) Migrate(ctx context.Context, newPath string, callback ProgressCallback) error {
    // Stage 1: Validation
    callback(MigrationProgress{Stage: "validating", Percent: 10, Message: "Validating new location..."})
    if err := m.validateNewLocation(newPath); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // Stage 2: Pre-migration backup
    callback(MigrationProgress{Stage: "backup", Percent: 20, Message: "Creating backup..."})
    backupPath, err := m.createBackup(ctx)
    if err != nil {
        return fmt.Errorf("backup failed: %w", err)
    }
    defer os.RemoveAll(backupPath) // Clean up backup on success
    
    // Stage 3: Close database
    callback(MigrationProgress{Stage: "closing", Percent: 30, Message: "Closing database..."})
    if err := m.store.Close(); err != nil {
        return fmt.Errorf("failed to close database: %w", err)
    }
    
    // Stage 4: Copy files
    callback(MigrationProgress{Stage: "copying", Percent: 40, Message: "Copying database files..."})
    oldPath := m.config.Database.Path
    if err := m.copyDatabaseFiles(oldPath, newPath, callback); err != nil {
        // Rollback: restore from backup
        m.rollback(backupPath, oldPath)
        return fmt.Errorf("copy failed: %w", err)
    }
    
    // Stage 5: Verify integrity
    callback(MigrationProgress{Stage: "verifying", Percent: 70, Message: "Verifying database integrity..."})
    if err := m.verifyDatabase(newPath); err != nil {
        m.rollback(backupPath, oldPath)
        return fmt.Errorf("verification failed: %w", err)
    }
    
    // Stage 6: Update config
    callback(MigrationProgress{Stage: "updating", Percent: 85, Message: "Updating configuration..."})
    if err := m.updateConfig(oldPath, newPath); err != nil {
        m.rollback(backupPath, oldPath)
        return fmt.Errorf("config update failed: %w", err)
    }
    
    // Stage 7: Clean up old location (optional - ask user)
    callback(MigrationProgress{Stage: "cleanup", Percent: 95, Message: "Migration complete..."})
    
    callback(MigrationProgress{Stage: "complete", Percent: 100, Message: "Database migrated successfully!"})
    return nil
}

func (m *MigrationService) validateNewLocation(newPath string) error {
    // Check if directory exists or can be created
    if err := os.MkdirAll(newPath, 0755); err != nil {
        return fmt.Errorf("cannot create directory: %w", err)
    }
    
    // Check write permissions
    testFile := filepath.Join(newPath, ".write-test")
    if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
        return fmt.Errorf("no write permission: %w", err)
    }
    os.Remove(testFile)
    
    // Check available space
    oldPath := m.config.Database.Path
    requiredSpace, err := calculateDiskUsage(oldPath)
    if err != nil {
        return fmt.Errorf("cannot calculate required space: %w", err)
    }
    
    availableSpace, err := getAvailableSpace(newPath)
    if err != nil {
        return fmt.Errorf("cannot check available space: %w", err)
    }
    
    if availableSpace < requiredSpace*2 { // 2x for safety margin
        return fmt.Errorf("insufficient disk space: need %d MB, have %d MB", 
            requiredSpace/1024/1024, availableSpace/1024/1024)
    }
    
    // Check if destination already has database files
    dbFile := filepath.Join(newPath, "todo.db")
    if _, err := os.Stat(dbFile); err == nil {
        return fmt.Errorf("database already exists at destination")
    }
    
    return nil
}

func (m *MigrationService) createBackup(ctx context.Context) (string, error) {
    timestamp := time.Now().Format("20060102-150405")
    backupDir := filepath.Join(os.TempDir(), fmt.Sprintf("planck-backup-%s", timestamp))
    
    oldPath := m.config.Database.Path
    if err := copyDir(oldPath, backupDir); err != nil {
        return "", err
    }
    
    return backupDir, nil
}

func (m *MigrationService) copyDatabaseFiles(oldPath, newPath string, callback ProgressCallback) error {
    files := []string{"todo.db", "todo.db-wal", "todo.db-shm"}
    
    for i, file := range files {
        src := filepath.Join(oldPath, file)
        dst := filepath.Join(newPath, file)
        
        // Skip if file doesn't exist (WAL/SHM may not exist)
        if _, err := os.Stat(src); os.IsNotExist(err) {
            continue
        }
        
        if err := copyFile(src, dst); err != nil {
            return fmt.Errorf("failed to copy %s: %w", file, err)
        }
        
        percent := 40 + (i+1)*10 // Progress from 40% to 70%
        callback(MigrationProgress{
            Stage:   "copying",
            Percent: percent,
            Message: fmt.Sprintf("Copied %s", file),
        })
    }
    
    return nil
}

func (m *MigrationService) verifyDatabase(path string) error {
    // Open database and run integrity check
    dbPath := filepath.Join(path, "todo.db?_fk=1")
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return fmt.Errorf("cannot open database: %w", err)
    }
    defer db.Close()
    
    // Run integrity check
    var result string
    err = db.QueryRow("PRAGMA integrity_check").Scan(&result)
    if err != nil {
        return fmt.Errorf("integrity check failed: %w", err)
    }
    
    if result != "ok" {
        return fmt.Errorf("database integrity check failed: %s", result)
    }
    
    // Quick count to ensure data is there
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&count)
    if err != nil {
        return fmt.Errorf("cannot verify data: %w", err)
    }
    
    return nil
}

func (m *MigrationService) updateConfig(oldPath, newPath string) error {
    m.config.Database.Path = newPath
    m.config.Database.LastMigrated = time.Now().Format(time.RFC3339)
    
    record := config.MigrationRecord{
        From:      oldPath,
        To:        newPath,
        Timestamp: time.Now().Format(time.RFC3339),
        Method:    "copy",
    }
    
    m.config.Database.MigrationHistory = append(m.config.Database.MigrationHistory, record)
    
    return m.config.Save()
}

func (m *MigrationService) rollback(backupPath, originalPath string) error {
    // Remove partially copied files
    os.RemoveAll(originalPath)
    
    // Restore from backup
    return copyDir(backupPath, originalPath)
}

// Helper functions

func copyFile(src, dst string) error {
    sourceFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer sourceFile.Close()
    
    destFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destFile.Close()
    
    if _, err := io.Copy(destFile, sourceFile); err != nil {
        return err
    }
    
    return destFile.Sync()
}

func copyDir(src, dst string) error {
    return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        relPath, err := filepath.Rel(src, path)
        if err != nil {
            return err
        }
        
        dstPath := filepath.Join(dst, relPath)
        
        if info.IsDir() {
            return os.MkdirAll(dstPath, info.Mode())
        }
        
        return copyFile(path, dstPath)
    })
}

func calculateDiskUsage(path string) (int64, error) {
    var size int64
    err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() {
            size += info.Size()
        }
        return nil
    })
    return size, err
}

func getAvailableSpace(path string) (int64, error) {
    // Platform-specific implementation needed
    // For now, return a large number
    return 1024 * 1024 * 1024 * 10, nil // 10GB placeholder
}
```

### 3. Backend API Methods

**Add to `app.go`**:
```go
import (
    "todo-app/migration"
    "github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetCurrentDatabasePath returns the current database location
func (a *App) GetCurrentDatabasePath() (string, error) {
    cfg, err := config.Load()
    if err != nil {
        return "", err
    }
    return cfg.Database.Path, nil
}

// SelectDatabaseDirectory opens a native directory picker
func (a *App) SelectDatabaseDirectory() (string, error) {
    path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
        Title: "Select New Database Location",
    })
    
    if err != nil {
        return "", err
    }
    
    return path, nil
}

// MigrateDatabase performs the database migration
func (a *App) MigrateDatabase(newPath string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    
    migrator := migration.NewMigrationService(a.Store, cfg)
    
    // Progress callback - emit events to frontend
    progressCallback := func(progress migration.MigrationProgress) {
        runtime.EventsEmit(a.ctx, "migration:progress", progress)
    }
    
    if err := migrator.Migrate(a.ctx, newPath, progressCallback); err != nil {
        return err
    }
    
    // Emit completion event
    runtime.EventsEmit(a.ctx, "migration:complete", newPath)
    
    // Request app restart
    go func() {
        time.Sleep(2 * time.Second)
        runtime.Quit(a.ctx)
    }()
    
    return nil
}

// ValidateDatabaseLocation checks if a path is suitable
func (a *App) ValidateDatabaseLocation(path string) (bool, string, error) {
    cfg, err := config.Load()
    if err != nil {
        return false, "", err
    }
    
    migrator := migration.NewMigrationService(a.Store, cfg)
    
    if err := migrator.validateNewLocation(path); err != nil {
        return false, err.Error(), nil
    }
    
    return true, "Location is valid", nil
}
```

### 4. Frontend UI Implementation

**Update `SettingsModal.tsx`**:
```typescript
import { EventsOn } from '../../wailsjs/runtime/runtime';

// Inside component
const [migrating, setMigrating] = React.useState(false);
const [migrationProgress, setMigrationProgress] = React.useState({
  stage: '',
  percent: 0,
  message: ''
});

React.useEffect(() => {
  // Listen for migration progress events
  const unsubscribe = EventsOn('migration:progress', (progress: any) => {
    setMigrationProgress(progress);
  });
  
  const unsubscribeComplete = EventsOn('migration:complete', (newPath: string) => {
    setMigrating(false);
    alert(`Database migrated successfully to ${newPath}\nThe app will restart now.`);
  });
  
  return () => {
    unsubscribe();
    unsubscribeComplete();
  };
}, []);

const handleSelectNewLocation = async () => {
  try {
    const newPath = await Backend.SelectDatabaseDirectory();
    if (!newPath) return; // User cancelled
    
    // Validate location
    const [valid, message] = await Backend.ValidateDatabaseLocation(newPath);
    
    if (!valid) {
      alert(`Invalid location: ${message}`);
      return;
    }
    
    // Show confirmation
    const currentPath = await Backend.GetCurrentDatabasePath();
    const confirmed = confirm(
      `Move database to new location?\n\n` +
      `Current: ${currentPath}\n` +
      `New: ${newPath}\n\n` +
      `The app will:\n` +
      `• Close the database safely\n` +
      `• Copy all data to the new location\n` +
      `• Update configuration\n` +
      `• Restart automatically\n\n` +
      `Continue?`
    );
    
    if (!confirmed) return;
    
    // Start migration
    setMigrating(true);
    await Backend.MigrateDatabase(newPath);
    
  } catch (err) {
    console.error('Migration failed:', err);
    alert(`Migration failed: ${err}`);
    setMigrating(false);
  }
};

// In the UI
<h3>Database Location</h3>
<div>
  <label>Current Path</label>
  <div>{currentPath || './data/todo.db'}</div>
</div>

<button 
  onClick={handleSelectNewLocation}
  disabled={migrating}
  className="badge info"
>
  {migrating ? 'Migrating...' : 'Select New Location'}
</button>

{migrating && (
  <div>
    <div>Stage: {migrationProgress.stage}</div>
    <div>Progress: {migrationProgress.percent}%</div>
    <div>{migrationProgress.message}</div>
    <progress value={migrationProgress.percent} max={100} />
  </div>
)}
```

### 5. App Startup Changes

**Update `main.go`**:
```go
func main() {
    // Load config first
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    app := NewApp(cfg)
    
    err = wails.Run(&options.App{
        // ... existing options ...
        OnStartup: app.startup,
        OnShutdown: app.shutdown,
    })
}
```

**Update `app.go`**:
```go
type App struct {
    ctx    context.Context
    Store  *store.Store
    Config *config.Config
}

func NewApp(cfg *config.Config) *App {
    return &App{Config: cfg}
}

func (a *App) startup(ctx context.Context) {
    a.ctx = ctx
    
    // Use config path instead of hardcoded
    dbPath := a.Config.Database.Path
    
    s, err := store.Open(ctx, dbPath)
    if err != nil {
        panic(err)
    }
    a.Store = s
}

func (a *App) shutdown(ctx context.Context) {
    if a.Store != nil {
        a.Store.Close()
    }
}
```

## Testing Plan

### Unit Tests
1. Config loading/saving
2. Migration validation logic
3. File copying operations
4. Rollback mechanisms

### Integration Tests
1. Full migration flow (happy path)
2. Migration with insufficient space
3. Migration with permission issues
4. Migration interruption (simulate crash)
5. Rollback after failed migration

### Manual Testing Scenarios

#### Test 1: Basic Migration
1. Start with fresh database
2. Add 10 todos
3. Migrate to new location
4. Verify all todos present
5. Add new todo
6. Verify it saves to new location

#### Test 2: Large Database Migration
1. Import 1000 todos
2. Migrate to new location
3. Verify integrity
4. Performance check

#### Test 3: Migration to Cloud Drive
1. Migrate to Dropbox folder
2. Verify WAL files sync
3. Close app, wait for sync
4. Open on different computer
5. Verify data accessible

#### Test 4: Failed Migration Recovery
1. Start migration
2. Kill process mid-copy
3. Restart app
4. Verify original database still works
5. Check backup cleanup

#### Test 5: Permission Issues
1. Try to migrate to read-only location
2. Verify error message
3. Verify original database untouched

## Security Considerations

1. **Path Validation**: Prevent directory traversal attacks
2. **Permissions**: Check read/write before migration
3. **Backup Retention**: Store backup until verification complete
4. **Atomic Operations**: Use temp locations and rename
5. **Config Security**: Store config in user's home directory only

## Performance Considerations

1. **Large Database**: Show progress for databases > 100MB
2. **Network Drives**: Warn user about slower migration
3. **Background Operations**: Don't block UI during copy
4. **Streaming Copy**: Use buffered I/O for large files

## Edge Cases

1. **Symlink Existing**: Detect if current path is already symlink
2. **Migration During Sync**: Warn if cloud sync in progress
3. **Multiple Instances**: Prevent migration if app running elsewhere
4. **Partial Migration**: Handle incomplete previous migrations
5. **Config Corruption**: Fallback to default path if config invalid

## Rollout Strategy

### Phase 1: Foundation (Week 1)
- Implement config system
- Add database path to config
- Update app startup to read config

### Phase 2: Migration Core (Week 2)
- Implement migration service
- Add validation logic
- Add backup/rollback

### Phase 3: UI Integration (Week 3)
- Add directory picker
- Add progress UI
- Add confirmation dialogs

### Phase 4: Testing (Week 4)
- Unit tests
- Integration tests
- Manual testing all scenarios

### Phase 5: Documentation & Release
- Update user guide
- Create migration tutorial
- Beta release to select users

## Success Metrics

- **Migration Success Rate**: > 99%
- **Data Loss**: 0 incidents
- **User Satisfaction**: Positive feedback on ease of use
- **Support Tickets**: < 5% related to migration issues

## Future Enhancements

1. **Automatic Backup**: Auto-export before migration
2. **Cloud Service Detection**: Detect Dropbox/iCloud folders automatically
3. **Migration History UI**: Show previous migrations in settings
4. **Remote Database**: Support PostgreSQL/MySQL for team usage
5. **Sync Status**: Show cloud sync status in app
6. **Conflict Resolution**: Built-in UI for handling sync conflicts

## Dependencies

- Wails v2 runtime (for directory picker)
- Go 1.21+ (for improved error handling)
- SQLite with WAL support
- Platform-specific disk space APIs (optional)

## Estimated Effort

- **Development**: 3-4 weeks
- **Testing**: 1-2 weeks
- **Documentation**: 1 week
- **Total**: 5-7 weeks for complete feature

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Data loss during migration | Low | Critical | Backup before migration, verify after |
| Insufficient disk space | Medium | High | Check space before starting |
| Permission issues | Medium | Medium | Validate permissions upfront |
| App crash during migration | Low | High | Rollback mechanism, temp locations |
| Config corruption | Low | Medium | Fallback to defaults |
| User confusion | Medium | Low | Clear UI, confirmation dialogs |

## Open Questions

1. Should we allow migration while app is running, or require restart?
   - **Recommendation**: Require restart for safety

2. Should we delete old database after successful migration?
   - **Recommendation**: Ask user, default to keep for 7 days

3. Should we support migration to network drives?
   - **Recommendation**: Support but warn about performance

4. Should we validate cloud service sync status?
   - **Recommendation**: Phase 2 feature, not MVP

5. Should config be in JSON or TOML?
   - **Recommendation**: JSON for JavaScript interop

## References

- SQLite WAL Mode: https://www.sqlite.org/wal.html
- Wails Runtime: https://wails.io/docs/reference/runtime/intro
- Go File Operations: https://golang.org/pkg/os/
- Atomic File Operations: https://github.com/google/renameio
