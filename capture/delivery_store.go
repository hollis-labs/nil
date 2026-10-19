package capture

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hollis-labs/nil/capture/broker"
)

const deliverySchema = `
CREATE TABLE IF NOT EXISTS capture_accounts(id TEXT PRIMARY KEY,account_json TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS capture_wire(capture_id TEXT PRIMARY KEY,account_id TEXT NOT NULL,origin TEXT NOT NULL,device_id TEXT NOT NULL,integration_id TEXT NOT NULL,request_json TEXT NOT NULL,request_hash TEXT NOT NULL);
CREATE TRIGGER IF NOT EXISTS immutable_capture_wire BEFORE UPDATE ON capture_wire BEGIN SELECT RAISE(ABORT,'capture wire request is immutable'); END;
CREATE TABLE IF NOT EXISTS capture_delivery(capture_id TEXT PRIMARY KEY,status_json TEXT NOT NULL DEFAULT '',local_stopped INTEGER NOT NULL DEFAULT 0,attempted INTEGER NOT NULL DEFAULT 0,error_code TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS capture_delivery_leases(capture_id TEXT PRIMARY KEY,holder TEXT NOT NULL,expires_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS capture_context_jobs(capture_id TEXT PRIMARY KEY,account_id TEXT NOT NULL,origin TEXT NOT NULL,device_id TEXT NOT NULL,state TEXT NOT NULL,binding_json TEXT NOT NULL DEFAULT '',code TEXT NOT NULL DEFAULT '');
`

type Account struct {
	ID        string   `json:"id"`
	Origin    string   `json:"origin"`
	DeviceID  string   `json:"deviceID"`
	Label     string   `json:"label"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expiresAt"`
}
type AccountSelection struct {
	Accounts   []Account `json:"accounts"`
	SelectedID string    `json:"selectedID"`
}
type Wire struct{ CaptureID, AccountID, Origin, DeviceID, IntegrationID, JSON, Hash string }
type ContextJob struct{ CaptureID, AccountID, Origin, DeviceID, State, BindingJSON, Code string }
type ContextEvidence struct {
	State   string `json:"state"`
	Code    string `json:"code"`
	DraftAt string `json:"draftAt"`
}
type DeliveryView struct {
	CaptureID string          `json:"captureID"`
	AccountID string          `json:"accountID"`
	State     string          `json:"state"`
	Code      string          `json:"code"`
	Confirmed bool            `json:"confirmed"`
	Status    broker.Status   `json:"status"`
	Context   ContextEvidence `json:"context"`
}

func (s *Store) saveAccount(ctx context.Context, a Account, selectIt bool) error {
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `INSERT INTO capture_accounts(id,account_json) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET account_json=excluded.account_json`, a.ID, string(raw))
	if err != nil {
		return err
	}
	if selectIt {
		if _, err = tx.ExecContext(ctx, `INSERT INTO capture_preferences(key,value) VALUES('broker_account',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, a.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) Accounts(ctx context.Context) (AccountSelection, error) {
	result := AccountSelection{Accounts: []Account{}}
	rows, err := s.db.QueryContext(ctx, `SELECT account_json FROM capture_accounts ORDER BY id`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var raw string
		var a Account
		if err = rows.Scan(&raw); err == nil {
			err = json.Unmarshal([]byte(raw), &a)
		}
		if err != nil {
			_ = rows.Close()
			return result, err
		}
		result.Accounts = append(result.Accounts, a)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return result, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT value FROM capture_preferences WHERE key='broker_account'`).Scan(&result.SelectedID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return result, err
}
func (s *Store) account(ctx context.Context, id string) (Account, error) {
	var a Account
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT account_json FROM capture_accounts WHERE id=?`, id).Scan(&raw)
	if err != nil {
		return a, err
	}
	err = json.Unmarshal([]byte(raw), &a)
	return a, err
}
func (s *Store) selectedAccount(ctx context.Context) (Account, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM capture_preferences WHERE key='broker_account'`).Scan(&id)
	if err != nil {
		return Account{}, err
	}
	return s.account(ctx, id)
}
func (s *Store) SelectAccount(ctx context.Context, id string) error {
	if _, err := s.account(ctx, id); err != nil {
		return errors.New("pairing not available in native custody")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO capture_preferences(key,value) VALUES('broker_account',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, id)
	return err
}

func (s *Store) lease(ctx context.Context, id string) (string, error) {
	holder := uuid.NewString()
	now := time.Now().Unix()
	r, err := s.db.ExecContext(ctx, `INSERT INTO capture_delivery_leases(capture_id,holder,expires_at) VALUES(?,?,?) ON CONFLICT(capture_id) DO UPDATE SET holder=excluded.holder,expires_at=excluded.expires_at WHERE capture_delivery_leases.expires_at<=?`, id, holder, now+120, now)
	if err != nil {
		return "", err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return "", err
	}
	if n != 1 {
		return "", errors.New("capture operation is already in progress; refresh after it finishes")
	}
	return holder, nil
}
func (s *Store) release(ctx context.Context, id, holder string) {
	_, _ = s.db.ExecContext(ctx, `DELETE FROM capture_delivery_leases WHERE capture_id=? AND holder=?`, id, holder)
}
func (s *Store) held(ctx context.Context, id, holder string) error {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM capture_delivery_leases WHERE capture_id=? AND holder=? AND expires_at>?`, id, holder, time.Now().Unix()).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("capture operation lease expired; refresh before retry")
	}
	return nil
}
func (s *Store) wire(ctx context.Context, id string) (Wire, error) {
	var w Wire
	err := s.db.QueryRowContext(ctx, `SELECT capture_id,account_id,origin,device_id,integration_id,request_json,request_hash FROM capture_wire WHERE capture_id=?`, id).Scan(&w.CaptureID, &w.AccountID, &w.Origin, &w.DeviceID, &w.IntegrationID, &w.JSON, &w.Hash)
	if err == nil && hash(w.JSON) != w.Hash {
		err = errors.New("finalized capture request integrity check failed")
	}
	return w, err
}
func (s *Store) putWire(ctx context.Context, w Wire, holder string) error {
	if err := s.held(ctx, w.CaptureID, holder); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO capture_wire(capture_id,account_id,origin,device_id,integration_id,request_json,request_hash) VALUES(?,?,?,?,?,?,?)`, w.CaptureID, w.AccountID, w.Origin, w.DeviceID, w.IntegrationID, w.JSON, w.Hash)
	return err
}
func (s *Store) intent(ctx context.Context, id, holder string) error {
	if err := s.held(ctx, id, holder); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO capture_delivery(capture_id,attempted) VALUES(?,1) ON CONFLICT(capture_id) DO UPDATE SET attempted=1`, id)
	return err
}
func (s *Store) recordStatus(ctx context.Context, w Wire, status broker.Status, holder string) error {
	if err := s.held(ctx, w.CaptureID, holder); err != nil {
		return err
	}
	raw, err := json.Marshal(status)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO capture_delivery(capture_id,status_json) VALUES(?,?) ON CONFLICT(capture_id) DO UPDATE SET status_json=excluded.status_json,error_code=''`, w.CaptureID, string(raw))
	return err
}
func (s *Store) contextJob(ctx context.Context, id string) (ContextJob, error) {
	var j ContextJob
	err := s.db.QueryRowContext(ctx, `SELECT capture_id,account_id,origin,device_id,state,binding_json,code FROM capture_context_jobs WHERE capture_id=?`, id).Scan(&j.CaptureID, &j.AccountID, &j.Origin, &j.DeviceID, &j.State, &j.BindingJSON, &j.Code)
	return j, err
}
func (s *Store) contextEvidence(ctx context.Context, id string) ContextEvidence {
	j, err := s.contextJob(ctx, id)
	if err != nil {
		return ContextEvidence{State: "unavailable", Code: "not_sampled_at_invocation"}
	}
	evidence := ContextEvidence{State: j.State, Code: j.Code}
	if j.BindingJSON != "" {
		var b broker.Binding
		if json.Unmarshal([]byte(j.BindingJSON), &b) == nil {
			evidence.State = b.State
			evidence.DraftAt = b.DraftAt
		}
	}
	return evidence
}
func (s *Store) DeliveryViews(ctx context.Context) ([]DeliveryView, error) {
	// Read only local custody rows. This method never resolves credentials/probes.
	rows, err := s.db.QueryContext(ctx, `SELECT q.capture_id,COALESCE(w.account_id,''),COALESCE(d.status_json,''),COALESCE(d.local_stopped,0),COALESCE(d.error_code,'') FROM capture_local_queue q LEFT JOIN capture_wire w ON w.capture_id=q.capture_id LEFT JOIN capture_delivery d ON d.capture_id=q.capture_id ORDER BY q.rowid DESC`)
	if err != nil {
		return nil, err
	}
	result := []DeliveryView{}
	for rows.Next() {
		var v DeliveryView
		var raw string
		var stopped bool
		if err = rows.Scan(&v.CaptureID, &v.AccountID, &raw, &stopped, &v.Code); err != nil {
			_ = rows.Close()
			return nil, err
		}
		v.State = "local_queued"
		if stopped {
			v.State = "local_stopped"
		}
		if raw != "" {
			if err = json.Unmarshal([]byte(raw), &v.Status); err != nil {
				_ = rows.Close()
				return nil, err
			}
			v.State = v.Status.State
			v.Confirmed = v.Status.Confirmed()
		}
		result = append(result, v)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Context = s.contextEvidence(ctx, result[i].CaptureID)
	}
	return result, nil
}

// stopUnattempted leaves a local tombstone without creating a remote operation.
// An attempted POST, including a lost response, must use broker reconciliation.
func (s *Store) stopUnattempted(ctx context.Context, id, holder string) (bool, error) {
	if err := s.held(ctx, id, holder); err != nil {
		return false, err
	}
	var queued int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM capture_local_queue WHERE capture_id=?`, id).Scan(&queued); err != nil {
		return false, err
	}
	if queued != 1 {
		return false, errors.New("queue the draft locally before a delivery action")
	}
	r, err := s.db.ExecContext(ctx, `INSERT INTO capture_delivery(capture_id,local_stopped) VALUES(?,1) ON CONFLICT(capture_id) DO UPDATE SET local_stopped=1 WHERE capture_delivery.attempted=0 AND capture_delivery.status_json=''`, id)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	return n == 1, err
}
