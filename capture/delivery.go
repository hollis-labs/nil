package capture

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hollis-labs/libs/security/secrets"
	"github.com/hollis-labs/nil/capture/broker"
)

// CredentialCustody is native-only. Tests inject disposable fixture custody;
// the renderer never receives this interface, its reference, or a bearer.
type CredentialCustody interface {
	Get(context.Context, string) (secrets.Secret, error)
	Put(context.Context, string, secrets.Secret) error
}
type NativeCustody struct{ resolver *secrets.Resolver }

func NewNativeCustody() *NativeCustody { return &NativeCustody{resolver: secrets.New()} }
func credentialRef(key string) string {
	scheme := "keychain"
	if runtime.GOOS == "linux" {
		scheme = "file"
	}
	return scheme + "://nil-basalisk/" + key
}
func (n *NativeCustody) Get(ctx context.Context, key string) (secrets.Secret, error) {
	result, err := n.resolver.Resolve(ctx, credentialRef(key))
	return result.Secret, err
}
func (n *NativeCustody) Put(ctx context.Context, key string, value secrets.Secret) error {
	return n.resolver.Set(ctx, credentialRef(key), strings.NewReader(value.Reveal()))
}

type Delivery struct {
	store      *Store
	custody    CredentialCustody
	transport  http.RoundTripper
	management sync.Mutex
}

func NewDelivery(ctx context.Context, s *Store, custody CredentialCustody, transport http.RoundTripper) (*Delivery, error) {
	// A prior process may have lost its initial context response. Recovery must
	// distinguish 200 replay from a new late 201; never resample old drafts.
	_, err := s.db.ExecContext(ctx, `UPDATE capture_context_jobs SET state='ambiguous',code='process_restarted' WHERE state='pending'`)
	if err != nil {
		return nil, err
	}
	return &Delivery{store: s, custody: custody, transport: transport}, nil
}
func (d *Delivery) client(a Account) (*broker.Client, error) {
	return broker.New(a.Origin, a.DeviceID, func(ctx context.Context) (secrets.Secret, error) { return d.custody.Get(ctx, a.ID) }, d.transport)
}
func (d *Delivery) Pair(ctx context.Context, origin, code string) (Account, error) {
	d.management.Lock()
	defer d.management.Unlock()
	client, err := broker.New(origin, "", nil, d.transport)
	if err != nil {
		return Account{}, err
	}
	exchanged, err := client.Exchange(ctx, code)
	if err != nil {
		return Account{}, err
	}
	a := Account{ID: hash(client.Origin + "\n" + exchanged.Device.ID), Origin: client.Origin, DeviceID: exchanged.Device.ID, Label: exchanged.Device.Label, Scopes: exchanged.Device.Scopes, ExpiresAt: exchanged.Device.ExpiresAt.UTC().Format(time.RFC3339Nano)}
	if err = d.custody.Put(ctx, a.ID, secrets.FromString(exchanged.Token)); err != nil {
		return Account{}, errors.New("pairing credential could not be stored in native custody; request a new one-use code")
	}
	if err = d.store.saveAccount(ctx, a, true); err != nil {
		return Account{}, errors.New("pairing metadata could not be stored; retain the existing pairing and repair local storage")
	}
	return a, nil
}
func (d *Delivery) Renew(ctx context.Context, expectedAccount ...string) (Account, error) {
	d.management.Lock()
	defer d.management.Unlock()
	a, err := d.store.selectedAccount(ctx)
	if err != nil {
		return a, errors.New("select an existing Nil pairing before renewal")
	}
	if len(expectedAccount) > 0 && expectedAccount[0] != a.ID {
		return Account{}, errors.New("selected Nil pairing changed; reload local setup before continuing")
	}
	c, err := d.client(a)
	if err != nil {
		return a, err
	}
	device, err := c.Renew(ctx)
	if err != nil {
		return a, err
	}
	a.Scopes = device.Scopes
	a.ExpiresAt = device.ExpiresAt.UTC().Format(time.RFC3339Nano)
	return a, d.store.saveAccount(ctx, a, false)
}

type SetupEvidence struct {
	Account            Account             `json:"account"`
	Capabilities       broker.Capabilities `json:"capabilities"`
	Readiness          broker.Readiness    `json:"readiness"`
	ReadinessSupported bool                `json:"readinessSupported"`
}

func (d *Delivery) Evidence(ctx context.Context, expectedAccount ...string) (SetupEvidence, error) {
	var result SetupEvidence
	a, err := d.store.selectedAccount(ctx)
	if err != nil {
		return result, errors.New("pair Nil independently with Basalisk's Capture app profile")
	}
	result.Account = a
	if len(expectedAccount) > 0 && expectedAccount[0] != a.ID {
		return result, errors.New("selected Nil pairing changed; reload local setup before continuing")
	}
	c, err := d.client(a)
	if err != nil {
		return result, err
	}
	result.Capabilities, err = c.Discover(ctx)
	if err != nil {
		return result, err
	}
	result.ReadinessSupported = result.Capabilities.Readiness.Version == 1
	if result.ReadinessSupported {
		result.Readiness, err = c.Evidence(ctx)
	}
	return result, err
}

// BeginContext does local initiation only; callers run SampleContext in a
// bounded goroutine so network availability never blocks typing. It is invoked
// only for a NEW draft, never when loading/reconnecting historical drafts.
func (d *Delivery) BeginContext(ctx context.Context, id string) (bool, error) {
	a, err := d.store.selectedAccount(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = d.store.db.ExecContext(ctx, `INSERT INTO capture_context_jobs(capture_id,account_id,origin,device_id,state) VALUES(?,?,?,?,'pending')`, id, a.ID, a.Origin, a.DeviceID)
	return err == nil, err
}
func (d *Delivery) finishContext(ctx context.Context, j ContextJob, state, code string, b *broker.Binding) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	raw := ""
	if b != nil {
		data, err := json.Marshal(b)
		if err != nil {
			return err
		}
		raw = string(data)
	}
	r, err := d.store.db.ExecContext(ctx, `UPDATE capture_context_jobs SET state=?,code=?,binding_json=? WHERE capture_id=? AND account_id=? AND state=?`, state, code, raw, j.CaptureID, j.AccountID, j.State)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("context sampling state changed; original draft retained")
	}
	return nil
}
func (d *Delivery) SampleContext(ctx context.Context, id string) error {
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_, _ = d.store.db.ExecContext(cleanup, `UPDATE capture_context_jobs SET state='ambiguous',code='invocation_interrupted' WHERE capture_id=? AND state='pending'`, id)
	}()
	j, err := d.store.contextJob(ctx, id)
	if err != nil {
		return err
	}
	if j.State != "pending" {
		return nil
	}
	a, err := d.store.account(ctx, j.AccountID)
	if err != nil {
		return err
	}
	c, err := d.client(a)
	if err != nil {
		return err
	}
	caps, err := c.Discover(ctx)
	if err != nil || caps.Context.Version != 1 || !caps.Context.Allowed {
		return d.finishContext(ctx, j, "unavailable", "context_unavailable_at_invocation", nil)
	}
	binding, _, err := c.Bind(ctx, id)
	if err != nil {
		return d.finishContext(ctx, j, "ambiguous", "binding_response_unconfirmed", nil)
	}
	return d.finishContext(ctx, j, "bound", "sampled_at_invocation", &binding)
}
func (d *Delivery) resolveContext(ctx context.Context, id string, a Account, c *broker.Client) (*broker.Binding, error) {
	j, err := d.store.contextJob(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if j.AccountID != a.ID || j.Origin != a.Origin || j.DeviceID != a.DeviceID {
		return nil, errors.New("draft context belongs to the original Nil pairing; select it before delivery")
	}
	if j.State == "pending" {
		return nil, errors.New("draft context is still being sampled; retry after it finishes")
	}
	if j.State == "ambiguous" {
		b, status, bindErr := c.Bind(ctx, id)
		if bindErr != nil {
			return nil, bindErr
		}
		if status == http.StatusCreated {
			// A new context sampled on reconnect is not evidence of historical focus.
			return nil, d.finishContext(ctx, j, "unavailable", "late_context_not_used", nil)
		}
		if err = d.finishContext(ctx, j, "bound", "recovered_original_binding", &b); err != nil {
			return nil, err
		}
		return &b, nil
	}
	if j.State != "bound" {
		return nil, nil
	}
	var b broker.Binding
	if err = json.Unmarshal([]byte(j.BindingJSON), &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (d *Delivery) prepare(ctx context.Context, id, holder string, a Account, c *broker.Client) (Wire, error) {
	var stopped bool
	err := d.store.db.QueryRowContext(ctx, `SELECT local_stopped FROM capture_delivery WHERE capture_id=?`, id).Scan(&stopped)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Wire{}, err
	}
	if stopped {
		return Wire{}, errors.New("this local capture was stopped; retain/export it, never silently replace its identity")
	}
	w, err := d.store.wire(ctx, id)
	if err == nil {
		if w.AccountID != a.ID || w.Origin != a.Origin || w.DeviceID != a.DeviceID {
			return w, errors.New("capture belongs to its original Nil pairing; re-pairing cannot adopt it")
		}
		return w, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return w, err
	}
	draft, err := d.store.Get(ctx, id)
	if err != nil {
		return w, err
	}
	env, err := d.store.Queue(ctx, id, draft.Revision)
	if err != nil {
		return w, err
	}
	caps, err := c.Discover(ctx)
	if err != nil {
		return w, err
	}
	binding, err := d.resolveContext(ctx, id, a, c)
	if err != nil {
		return w, err
	}
	connection, err := c.Connection(ctx)
	if err != nil {
		return w, err
	}
	fields := env.Draft.Fields
	if fields.ThresholdDate != nil {
		return w, errors.New("basalisk metadata v1 does not support threshold dates; retain it in Markdown or repair Quick Add before queuing a new operation")
	}
	m := broker.Metadata{SchemaVersion: 1, Client: broker.ClientMetadata{Name: "nil", Version: "2.0.0"}, Title: fields.Title, Kind: "note", Projects: fields.Projects, Contexts: fields.Contexts}
	if fields.Priority != nil {
		m.Priority = *fields.Priority
	}
	if fields.DueDate != nil {
		m.DueDate = *fields.DueDate
	}
	for _, r := range env.Draft.References {
		m.References = append(m.References, broker.Reference{Relation: r.Relation, Provider: r.Provider, Instance: r.Instance, Kind: r.Kind, ID: r.ID, Revision: r.Revision, Label: r.Label})
	}
	if p := env.Draft.Snapshot.Preset; p != nil {
		m.Preset = &broker.Preset{ID: p.ID, Revision: strconv.Itoa(p.Revision), AppliedBy: env.Draft.Snapshot.AppliedBy}
	}
	req := broker.Request{CaptureID: id, IntegrationID: connection.IntegrationID, Content: env.Content, Tags: fields.Tags, DraftAt: env.Draft.CreatedAt, SubmittedAt: time.Now().UTC().Format(time.RFC3339Nano), Metadata: m}
	if binding != nil {
		req.ContextBindingID = binding.BindingID
		req.DraftAt = binding.DraftAt
		boundAt, _ := time.Parse(time.RFC3339Nano, binding.DraftAt)
		if boundAt.After(time.Now()) {
			req.SubmittedAt = binding.DraftAt
		}
	}
	raw, err := broker.ValidateRequest(req, caps)
	if err != nil {
		return w, err
	}
	w = Wire{CaptureID: id, AccountID: a.ID, Origin: a.Origin, DeviceID: a.DeviceID, IntegrationID: req.IntegrationID, JSON: string(raw), Hash: hash(string(raw))}
	return w, d.store.putWire(ctx, w, holder)
}
func validateStatus(w Wire, s broker.Status) error {
	if s.CaptureID != w.CaptureID || s.DeviceID != w.DeviceID || s.IntegrationID != w.IntegrationID || !slices.Contains([]string{"queued", "needs_attention", "saved", "discarded"}, s.State) {
		return errors.New("basalisk status does not match the finalized capture identity")
	}
	return nil
}

// Run serializes send/status/recovery across processes with a bounded lease.
// A stop after ambiguous acceptance first reconciles/replays the ORIGINAL bytes;
// a stop404 is never treated as an initial-submission cancellation tombstone.
func (d *Delivery) Run(ctx context.Context, id, action string, acknowledgeMayHaveSent bool, expectedAccount ...string) (broker.Status, error) {
	if !broker.ValidID(id) || !slices.Contains([]string{"send", "refresh", "retry", "stop"}, action) {
		return broker.Status{}, errors.New("invalid capture delivery action")
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	holder, err := d.store.lease(ctx, id)
	if err != nil {
		return broker.Status{}, err
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer done()
		d.store.release(cleanup, id, holder)
	}()
	if action == "stop" {
		if len(expectedAccount) > 0 && expectedAccount[0] != "" {
			selected, selectionErr := d.store.selectedAccount(ctx)
			if selectionErr != nil || selected.ID != expectedAccount[0] {
				return broker.Status{}, errors.New("selected Nil pairing changed; reload local setup before continuing")
			}
		}
		stopped, stopErr := d.store.stopUnattempted(ctx, id, holder)
		if stopErr != nil {
			return broker.Status{}, stopErr
		}
		if stopped {
			return broker.Status{CaptureID: id, State: "local_stopped"}, nil
		}
	}
	a, err := d.store.selectedAccount(ctx)
	if err != nil {
		return broker.Status{}, errors.New("pair/select Nil in Basalisk before delivery")
	}
	if len(expectedAccount) > 0 && expectedAccount[0] != a.ID {
		return broker.Status{}, errors.New("selected Nil pairing changed; reload local setup before continuing")
	}
	c, err := d.client(a)
	if err != nil {
		return broker.Status{}, err
	}
	w, err := d.prepare(ctx, id, holder, a, c)
	if err != nil {
		return broker.Status{}, err
	}
	status, err := c.Status(ctx, id)
	if broker.IsStatus(err, 404) {
		if action == "refresh" {
			return status, errors.New("acceptance unconfirmed; send the original immutable operation to reconcile")
		}
		if err = d.store.intent(ctx, id, holder); err != nil {
			return status, err
		}
		status, err = c.Submit(ctx, []byte(w.JSON))
	}
	if err != nil {
		return status, err
	}
	if err = validateStatus(w, status); err != nil {
		return status, err
	}
	if err = d.store.recordStatus(ctx, w, status, holder); err != nil {
		return status, err
	}
	if action == "retry" || action == "stop" {
		caps, discoveryErr := c.Discover(ctx)
		if discoveryErr != nil {
			return status, discoveryErr
		}
		if caps.Recovery.Version != 1 {
			return status, errors.New("basalisk server update required: device-scoped recovery v1 unsupported")
		}
		if action == "stop" && status.MayHaveSent && !acknowledgeMayHaveSent {
			return status, errors.New("an FE copy may already exist; acknowledge this warning before stopping delivery (FE data is never deleted)")
		}
		if err = d.store.held(ctx, id, holder); err != nil {
			return status, err
		}
		recovered, actionErr := c.Action(ctx, id, action)
		if broker.IsStatus(actionErr, 409) {
			recovered, actionErr = c.Status(ctx, id)
		}
		if actionErr != nil {
			return status, actionErr
		}
		if err = validateStatus(w, recovered); err != nil {
			return status, err
		}
		if err = d.store.recordStatus(ctx, w, recovered, holder); err != nil {
			return status, err
		}
		status = recovered
	}
	return status, nil
}

func (d *Delivery) RemotePage(ctx context.Context, before int64, expectedAccount ...string) (broker.StatusPage, error) {
	a, err := d.store.selectedAccount(ctx)
	if err != nil {
		return broker.StatusPage{}, errors.New("select the original Nil pairing")
	}
	if len(expectedAccount) > 0 && expectedAccount[0] != a.ID {
		return broker.StatusPage{}, errors.New("selected Nil pairing changed; reload local setup before continuing")
	}
	c, err := d.client(a)
	if err != nil {
		return broker.StatusPage{}, err
	}
	caps, err := c.Discover(ctx)
	if err != nil {
		return broker.StatusPage{}, err
	}
	if caps.Recovery.Version != 1 {
		return broker.StatusPage{}, errors.New("basalisk recovery v1 unsupported")
	}
	page, err := c.List(ctx, before)
	if err != nil {
		return page, err
	}
	for _, item := range page.Items {
		if item.DeviceID != a.DeviceID {
			return broker.StatusPage{}, errors.New("basalisk list returned a different device identity")
		}
	}
	return page, nil
}
