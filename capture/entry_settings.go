package capture

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/hollis-labs/nil/capture/entry"
)

func (s *Store) EntryShortcut(ctx context.Context) (entry.Shortcut, error) {
	result := entry.Default()
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM capture_preferences WHERE key='entry_shortcut'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		return entry.Default(), errors.New("capture shortcut preference is invalid; global registration disabled")
	}
	normalized, err := entry.Normalize(result)
	if err != nil {
		return entry.Default(), err
	}
	return normalized, nil
}
func (s *Store) SaveEntryShortcut(ctx context.Context, shortcut entry.Shortcut) error {
	normalized, err := entry.Normalize(shortcut)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO capture_preferences(key,value) VALUES('entry_shortcut',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(raw))
	return err
}
