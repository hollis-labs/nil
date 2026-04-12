package contextcache

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/store"
	"github.com/hollis-labs/nil/vault"
)

// Snapshot captures agent-friendly context data that can be cached on disk.
type Snapshot struct {
	GeneratedAt string         `json:"generated_at"`
	App         AppInfo        `json:"app"`
	Schema      SchemaInfo     `json:"schema"`
	Stats       StatsInfo      `json:"stats"`
	Vaults      []VaultSummary `json:"vaults"`
	Inbox       InboxSummary   `json:"inbox"`
	Notes       []string       `json:"notes,omitempty"`
}

type AppInfo struct {
	Name    string   `json:"name"`
	Purpose string   `json:"purpose"`
	Version string   `json:"version"`
	Focus   []string `json:"focus"`
	Docs    []string `json:"docs"`
}

type SchemaInfo struct {
	Version int      `json:"version"`
	Tables  []string `json:"tables"`
	Fields  []string `json:"fields"`
}

type StatsInfo struct {
	VaultCount    int        `json:"vault_count"`
	ActiveVaultID string     `json:"active_vault_id"`
	ActiveTotals  VaultStats `json:"active_totals"`
	GlobalTotals  VaultStats `json:"global_totals"`
}

type InboxSummary struct {
	Count int `json:"count"`
}

type VaultSummary struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Path   string     `json:"path"`
	Active bool       `json:"active"`
	Stats  VaultStats `json:"stats"`
}

type VaultStats struct {
	Total        int `json:"total"`
	Todos        int `json:"todos"`
	Notes        int `json:"notes"`
	Open         int `json:"open"`
	Completed    int `json:"completed"`
	Archived     int `json:"archived"`
	Overdue      int `json:"overdue"`
	HighPriority int `json:"high_priority"`
}

// Refresh rebuilds a snapshot and writes it to disk.
func Refresh(ctx context.Context, cfg *config.Config, mgr *vault.Manager) (*Snapshot, error) {
	snap, err := BuildSnapshot(ctx, cfg, mgr)
	if err != nil {
		return nil, err
	}
	if err := Save(snap); err != nil {
		return nil, err
	}
	return snap, nil
}

// BuildSnapshot computes a fresh snapshot without writing it to disk.
func BuildSnapshot(ctx context.Context, cfg *config.Config, mgr *vault.Manager) (*Snapshot, error) {
	snap := &Snapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		App: AppInfo{
			Name:    "NIL",
			Purpose: "Keyboard-first personal tasks + notes with multi-vault storage",
			Version: "cli-dev",
			Focus:   []string{"Human-friendly CLI", "Agent-ready APIs", "Local-first data ownership", "Inbox + Vault views"},
			Docs:    []string{"README.md", "docs/CLI.md", "docs/QUICKSTART.md"},
		},
		Schema: SchemaInfo{
			Tables: []string{"todos", "todo_projects", "todo_contexts", "todo_tags", "refs"},
			Fields: []string{"title", "notes_md", "notes_text", "type", "priority", "due_at", "contexts", "projects", "tags", "inbox", "pinned", "api_source"},
		},
	}

	var (
		globalTotals VaultStats
		activeTotals VaultStats
		activeID     = mgr.GetActiveVaultID()
	)

	for _, vaultMeta := range cfg.Vaults {
		storeRef, err := mgr.StoreForID(vaultMeta.ID)
		if err != nil {
			return nil, fmt.Errorf("context cache: open vault %s: %w", vaultMeta.ID, err)
		}
		stats, err := gatherVaultStats(ctx, storeRef)
		if err != nil {
			return nil, fmt.Errorf("context cache: gather stats for %s: %w", vaultMeta.ID, err)
		}
		globalTotals = globalTotals.add(stats)
		if vaultMeta.ID == activeID {
			activeTotals = stats
		}
		snap.Vaults = append(snap.Vaults, VaultSummary{
			ID:     vaultMeta.ID,
			Name:   vaultMeta.Name,
			Path:   vaultMeta.Path,
			Active: vaultMeta.ID == activeID,
			Stats:  stats,
		})
	}

	snap.Schema.Version = detectSchemaVersion(ctx, mgr)
	snap.Inbox = InboxSummary{Count: gatherInboxCount(ctx, mgr)}
	snap.Stats = StatsInfo{
		VaultCount:    len(cfg.Vaults),
		ActiveVaultID: activeID,
		ActiveTotals:  activeTotals,
		GlobalTotals:  globalTotals,
	}

	return snap, nil
}

// Load reads the cached snapshot from disk.
func Load() (*Snapshot, error) {
	path := cachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// Save writes the snapshot JSON to disk.
func Save(snap *Snapshot) error {
	if snap == nil {
		return fmt.Errorf("context cache: cannot save nil snapshot")
	}
	if err := os.MkdirAll(cacheDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath(), data, 0644)
}

func cacheDir() string {
	return filepath.Join(config.GetConfigDir(), "context-cache")
}

func cachePath() string {
	return filepath.Join(cacheDir(), "snapshot.json")
}

func detectSchemaVersion(ctx context.Context, mgr *vault.Manager) int {
	if active := mgr.ActiveStore(); active != nil {
		var version int
		if err := active.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version); err == nil {
			return version
		}
	}
	return 0
}

func gatherInboxCount(ctx context.Context, mgr *vault.Manager) int {
	if inbox := mgr.InboxStore(); inbox != nil {
		if cnt, err := inbox.GetInboxCount(ctx); err == nil {
			return cnt
		}
	}
	return 0
}

func gatherVaultStats(ctx context.Context, s *store.Store) (VaultStats, error) {
	stats := VaultStats{}
	var err error
	err = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos").Scan(&stats.Total)
	if err != nil && err != sql.ErrNoRows {
		return stats, err
	}
	_ = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos WHERE type='todo'").Scan(&stats.Todos)
	_ = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos WHERE type='note'").Scan(&stats.Notes)
	_ = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos WHERE completed=1").Scan(&stats.Completed)
	_ = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos WHERE archived=1").Scan(&stats.Archived)
	_ = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos WHERE completed=0 AND archived=0").Scan(&stats.Open)
	_ = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos WHERE completed=0 AND archived=0 AND due_at IS NOT NULL AND due_at < date('now')").Scan(&stats.Overdue)
	_ = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM todos WHERE completed=0 AND archived=0 AND priority='A'").Scan(&stats.HighPriority)
	return stats, nil
}

func (v VaultStats) add(other VaultStats) VaultStats {
	return VaultStats{
		Total:        v.Total + other.Total,
		Todos:        v.Todos + other.Todos,
		Notes:        v.Notes + other.Notes,
		Open:         v.Open + other.Open,
		Completed:    v.Completed + other.Completed,
		Archived:     v.Archived + other.Archived,
		Overdue:      v.Overdue + other.Overdue,
		HighPriority: v.HighPriority + other.HighPriority,
	}
}
