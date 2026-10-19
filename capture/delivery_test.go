package capture

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hollis-labs/libs/security/secrets"
	"github.com/hollis-labs/nil/capture/broker"
)

type fixtureCustody map[string]secrets.Secret

func (f fixtureCustody) Get(_ context.Context, k string) (secrets.Secret, error) {
	v, ok := f[k]
	if !ok {
		return v, errors.New("fixture missing")
	}
	return v, nil
}
func (f fixtureCustody) Put(_ context.Context, k string, v secrets.Secret) error {
	f[k] = v
	return nil
}

type deliveryFixture struct {
	mu           sync.Mutex
	posts        []string
	accepted     bool
	lose         bool
	mayHaveSent  bool
	actionBodies []string
	binds        int
	bindStatus   int
	block        chan struct{}
	entered      chan struct{}
	server       *httptest.Server
}

func fixtureCaps() broker.Capabilities {
	var c broker.Capabilities
	c.SchemaVersion = 1
	c.DeviceID = "fixture-device"
	c.PairingProtocol = "1"
	c.Scopes = []string{broker.CaptureScope, broker.ContextScope}
	c.MetadataVersions = []int{1}
	c.Context.Version = 1
	c.Context.Allowed = true
	c.Recovery.Version = 1
	c.Readiness.Version = 1
	c.Limits.RequestBytes = 131072
	c.Limits.ContentBytes = 65536
	c.Limits.MetadataBytes = 16384
	c.Limits.Tags = 32
	c.Limits.References = 32
	return c
}
func (f *deliveryFixture) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer disposable-fixture" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	write := func(v any) { _ = json.NewEncoder(w).Encode(v) }
	switch r.URL.Path {
	case "/companion/capture-capabilities":
		write(fixtureCaps())
	case "/companion/fragment-connection":
		write(broker.Connection{DeviceID: "fixture-device", IntegrationID: "fixture-integration"})
	case "/companion/capture-context":
		var req struct {
			CaptureID string `json:"captureID"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.binds++
		w.WriteHeader(f.bindStatus)
		write(broker.Binding{SchemaVersion: 1, BindingID: "fixture-binding-0001", CaptureID: req.CaptureID, DeviceID: "fixture-device", Origin: f.server.URL, DraftAt: "2026-10-18T12:00:00.123456789Z", State: "active"})
	case "/companion/fragments":
		raw, _ := io.ReadAll(r.Body)
		f.posts = append(f.posts, string(raw))
		var req broker.Request
		_ = json.Unmarshal(raw, &req)
		if f.entered != nil {
			close(f.entered)
			f.entered = nil
			f.mu.Unlock()
			<-f.block
			f.mu.Lock()
		}
		if f.lose {
			f.lose = false
			http.Error(w, "lost response", http.StatusServiceUnavailable)
			return
		}
		f.accepted = true
		w.WriteHeader(202)
		write(broker.Status{CaptureID: req.CaptureID, DeviceID: "fixture-device", IntegrationID: "fixture-integration", State: "queued", MayHaveSent: f.mayHaveSent})
	default:
		id := strings.TrimPrefix(r.URL.Path, "/companion/fragments/")
		state := "queued"
		if strings.HasSuffix(id, "/stop") || strings.HasSuffix(id, "/retry") {
			raw, _ := io.ReadAll(r.Body)
			f.actionBodies = append(f.actionBodies, string(raw))
			id = strings.Split(id, "/")[0]
			state = "discarded"
		}
		if !f.accepted {
			http.NotFound(w, r)
			return
		}
		write(broker.Status{CaptureID: id, DeviceID: "fixture-device", IntegrationID: "fixture-integration", State: state, MayHaveSent: f.mayHaveSent})
	}
}
func fixtureDelivery(t *testing.T) (context.Context, *Store, *Delivery, *deliveryFixture, Draft, string) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	f := &deliveryFixture{bindStatus: 201}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	a := Account{ID: "fixture-account", Origin: f.server.URL, DeviceID: "fixture-device", Scopes: []string{broker.CaptureScope, broker.ContextScope}}
	if err = s.saveAccount(ctx, a, true); err != nil {
		t.Fatal(err)
	}
	d, err := NewDelivery(ctx, s, fixtureCustody{a.ID: secrets.FromString("disposable-fixture")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	draft, err = s.SaveContent(ctx, draft.CaptureID, draft.Revision, "# exact\r\n\nhello", "Title +work #fixture")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Queue(ctx, draft.CaptureID, draft.Revision); err != nil {
		t.Fatal(err)
	}
	return ctx, s, d, f, draft, dir
}
func TestDeliveryLostResponseRestartReplaysExactWire(t *testing.T) {
	ctx, s, d, f, draft, dir := fixtureDelivery(t)
	f.lose = true
	if _, err := d.Run(ctx, draft.CaptureID, "send", false); err == nil {
		t.Fatal("expected ambiguous response")
	}
	original, err := s.wire(ctx, draft.CaptureID)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.posts) != 1 || f.posts[0] != original.JSON {
		t.Fatal("wire not persisted before send")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err = NewDelivery(ctx, s, d.custody, nil)
	if err != nil {
		t.Fatal(err)
	}
	status, err := d.Run(ctx, draft.CaptureID, "send", false)
	if err != nil {
		t.Fatal(err)
	}
	if status.Confirmed() || status.State != "queued" || len(f.posts) != 2 || f.posts[1] != original.JSON {
		t.Fatal("replay changed bytes or overstated acceptance")
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE capture_wire SET request_json='changed'`); err == nil {
		t.Fatal("wire mutable")
	}
}
func TestDeliveryContextRecoveryNeverAttachesLateFocus(t *testing.T) {
	for _, code := range []int{200, 201} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			ctx, s, d, f, draft, _ := fixtureDelivery(t)
			started, err := d.BeginContext(ctx, draft.CaptureID)
			if err != nil || !started {
				t.Fatal(err)
			}
			_, err = s.db.ExecContext(ctx, `UPDATE capture_context_jobs SET state='ambiguous'`)
			if err != nil {
				t.Fatal(err)
			}
			f.bindStatus = code
			if _, err = d.Run(ctx, draft.CaptureID, "send", false); err != nil {
				t.Fatal(err)
			}
			var req broker.Request
			_ = json.Unmarshal([]byte(f.posts[0]), &req)
			if code == 200 {
				if req.ContextBindingID != "fixture-binding-0001" || req.DraftAt != "2026-10-18T12:00:00.123456789Z" {
					t.Fatal("original binding not exact")
				}
			} else if req.ContextBindingID != "" || req.DraftAt != draft.CreatedAt {
				t.Fatal("late focus attached to historical draft")
			}
		})
	}
}
func TestDeliveryStopWarningsEmptyBodyAndLocalTombstone(t *testing.T) {
	ctx, s, d, f, draft, _ := fixtureDelivery(t)
	f.mayHaveSent = true
	if _, err := d.Run(ctx, draft.CaptureID, "send", false); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Run(ctx, draft.CaptureID, "stop", false); err == nil || len(f.actionBodies) != 0 {
		t.Fatal("missing mayHaveSent warning")
	}
	if _, err := d.Run(ctx, draft.CaptureID, "stop", true); err != nil {
		t.Fatal(err)
	}
	if len(f.actionBodies) != 1 || f.actionBodies[0] != "" {
		t.Fatal("recovery body must be empty")
	}
	fresh, err := s.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err = s.Save(ctx, fresh.CaptureID, fresh.Revision, "local only")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Queue(ctx, fresh.CaptureID, fresh.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = d.Run(ctx, fresh.CaptureID, "stop", false); err != nil {
		t.Fatal(err)
	}
	if _, err = d.Run(ctx, fresh.CaptureID, "send", false); err == nil {
		t.Fatal("local tombstone resurrected")
	}
	if len(f.posts) != 1 {
		t.Fatal("local stop created a remote capture")
	}
}
func TestDeliveryCrossProcessLeaseAndPairingIsolation(t *testing.T) {
	ctx, s, d, f, draft, dir := fixtureDelivery(t)
	f.entered = make(chan struct{})
	entered := f.entered
	f.block = make(chan struct{})
	done := make(chan error, 1)
	go func() { _, err := d.Run(ctx, draft.CaptureID, "send", false); done <- err }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("send did not start")
	}
	other, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	second, err := NewDelivery(ctx, other, d.custody, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = second.Run(ctx, draft.CaptureID, "stop", true); err == nil {
		t.Fatal("concurrent recovery admitted")
	}
	close(f.block)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if err = s.saveAccount(ctx, Account{ID: "different", Origin: f.server.URL, DeviceID: "different-device"}, true); err != nil {
		t.Fatal(err)
	}
	if _, err = d.Run(ctx, draft.CaptureID, "send", false); err == nil {
		t.Fatal("new pairing adopted old operation")
	}
	if len(f.posts) != 1 {
		t.Fatal("duplicate submission")
	}
}
