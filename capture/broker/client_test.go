package broker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hollis-labs/libs/security/pairing"
	"github.com/hollis-labs/libs/security/secrets"
)

func TestPairingNarrowScopesAndRedirectRefusal(t *testing.T) {
	var scopes []string
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { targetCalls++; w.WriteHeader(200) }))
	defer target.Close()
	redirect := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if redirect {
			http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
			return
		}
		if r.Header.Get(pairing.ProtocolHeader) != "1" || r.URL.Path != pairing.ExchangePath || r.Header.Get("Authorization") != "" {
			t.Error("incorrect shared pairing protocol")
		}
		var req pairing.ExchangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		scopes = req.Scopes
		if req.Code != "disposable-code" || req.Audience != "basalisk" {
			t.Error("incorrect exchange")
		}
		_ = json.NewEncoder(w).Encode(pairing.Exchange{Device: pairing.Device{ID: "fixture-device", Audience: "basalisk", Scopes: []string{CaptureScope, ContextScope}, ExpiresAt: time.Now().Add(time.Hour)}, Token: "disposable-token", TokenType: "Bearer"})
	}))
	defer server.Close()
	c, err := New(server.URL, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Exchange(context.Background(), "disposable-code"); err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 2 || scopes[0] != ContextScope || scopes[1] != CaptureScope {
		t.Fatal("scope escalation")
	}
	redirect = true
	if _, err = c.Exchange(context.Background(), "disposable-code"); err == nil || strings.Contains(err.Error(), "disposable") {
		t.Fatal("redirect accepted or diagnostic leaked fixture")
	}
	if targetCalls != 0 {
		t.Fatal("redirect followed")
	}
	for _, origin := range []string{"http://remote.example", "https://example.test/path", "https://user:password@example.test", "https://example.test?token=value"} {
		if _, err = New(origin, "", nil, nil); err == nil {
			t.Fatal("unsafe origin accepted")
		}
	}
}
func TestCapabilitiesIdentityAndScopeDenial(t *testing.T) {
	for _, body := range []string{
		`{"schemaVersion":1,"deviceID":"other","pairingProtocol":"1","scopes":["basalisk.fragments.create"]}`,
		`{"schemaVersion":1,"deviceID":"fixture-device","pairingProtocol":"1","scopes":["basalisk.activity.read"]}`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer disposable-fixture" {
					t.Error("credential missing")
				}
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			c, err := New(server.URL, "fixture-device", func(context.Context) (secrets.Secret, error) { return secrets.FromString("disposable-fixture"), nil }, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = c.Discover(context.Background()); err == nil {
				t.Fatal("identity/scope mismatch accepted")
			}
		})
	}
}
func TestMetadataUnsupportedNeverSilentlyDropsFields(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	req := Request{CaptureID: "fixture-capture-0001", IntegrationID: "fixture-integration", Content: "source", DraftAt: now, SubmittedAt: now, Metadata: Metadata{SchemaVersion: 1, Client: ClientMetadata{Name: "nil", Version: "2"}, Title: "retained title"}}
	caps := Capabilities{MetadataVersions: []int{2}, Limits: CaptureLimits{RequestBytes: 131072, ContentBytes: 65536, MetadataBytes: 16384, Tags: 32, References: 32}}
	if _, err := ValidateRequest(req, caps); err == nil {
		t.Fatal("unsupported metadata accepted")
	}
	caps.MetadataVersions = []int{1}
	raw, err := ValidateRequest(req, caps)
	if err != nil || !strings.Contains(string(raw), "retained title") {
		t.Fatal("metadata dropped", err)
	}
	req.Metadata.Title = strings.Repeat("x", 201)
	if _, err = ValidateRequest(req, caps); err == nil {
		t.Fatal("overlong title accepted")
	}
	req.Metadata.Title = "valid"
	req.ContextBindingID = "bad/path"
	if _, err = ValidateRequest(req, caps); err == nil {
		t.Fatal("invalid binding accepted")
	}
}
