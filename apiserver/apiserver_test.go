package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/vault"
)

// testEnv wires up a real *http.Server.Handler backed by a temp-dir vault
// and inbox store, exactly like the production wiring in app.go/cli.go, but
// without binding a network listener — handlers are invoked directly via
// httptest against srv.Handler.
type testEnv struct {
	t       *testing.T
	handler http.Handler
	apiKey  string
	mgr     *vault.Manager
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	vaultDir := filepath.Join(t.TempDir(), "vault")
	inboxDir := filepath.Join(t.TempDir(), "inbox")

	cfg := &config.Config{
		ActiveVaultID: "v1",
		InboxPath:     inboxDir,
		Vaults: []config.Vault{
			{ID: "v1", Name: "Test Vault", Path: vaultDir},
		},
		APIEnabled: true,
	}
	cfg.EnsureDefaults()

	mgr, err := vault.NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(mgr.CloseAll)

	srv := New(cfg, mgr)

	return &testEnv{t: t, handler: srv.Handler, apiKey: cfg.APIKey, mgr: mgr}
}

func (e *testEnv) do(method, path string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var b []byte
	if body != nil {
		var err error
		b, err = json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("X-API-Key", e.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	e.handler.ServeHTTP(w, req)
	return w
}

// envelope mirrors apiResponse but keeps Data raw for per-test decoding.
type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, w.Body.String())
	}
	return env
}

// itemJSON is a loose decode target for asserting on individual fields
// without depending on the full ItemView/store.Item shape.
type itemJSON map[string]any

func TestHandleGetItemIncludesNotesText(t *testing.T) {
	e := newTestEnv(t)

	created := e.do("POST", "/api/v1/items", map[string]any{
		"title":    "Grocery list",
		"notes_md": "# Groceries\n\nMilk, eggs, bread.",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	createdEnv := decodeEnvelope(t, created)
	var createdItem itemJSON
	if err := json.Unmarshal(createdEnv.Data, &createdItem); err != nil {
		t.Fatalf("decode created item: %v", err)
	}
	id := int64(createdItem["id"].(float64))

	// GET /api/v1/items/{id} — the surface under test.
	got := e.do("GET", "/api/v1/items/"+strconv.FormatInt(id, 10), nil)
	if got.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", got.Code, got.Body.String())
	}
	gotEnv := decodeEnvelope(t, got)
	var item itemJSON
	if err := json.Unmarshal(gotEnv.Data, &item); err != nil {
		t.Fatalf("decode item: %v", err)
	}

	notesText, _ := item["notes_text"].(string)
	if notesText == "" {
		t.Fatalf("notes_text missing/empty in GET /api/v1/items/{id} response: %v", item)
	}
	if got, want := notesText, "Groceries"; !strings.Contains(got, want) {
		t.Errorf("notes_text=%q, want it to contain %q", got, want)
	}
	if got, want := notesText, "Milk, eggs, bread."; !strings.Contains(got, want) {
		t.Errorf("notes_text=%q, want it to contain %q", got, want)
	}

	// Existing fields are unchanged/still present alongside notes_text.
	if item["notes_doc"] == nil || item["notes_doc"] == "" {
		t.Errorf("notes_doc missing/empty — should be unchanged alongside notes_text")
	}
	if item["notes_html"] == nil || item["notes_html"] == "" {
		t.Errorf("notes_html missing/empty — should be unchanged alongside notes_text")
	}
	if item["title"] != "Grocery list" {
		t.Errorf("title=%v, want unchanged %q", item["title"], "Grocery list")
	}
}

func TestHandleSearchIncludesNotesTextAndHandlesEmptyBody(t *testing.T) {
	e := newTestEnv(t)

	withBody := e.do("POST", "/api/v1/items", map[string]any{
		"title":    "Has body",
		"notes_md": "Some real content.",
	})
	if withBody.Code != http.StatusCreated {
		t.Fatalf("create withBody status=%d body=%s", withBody.Code, withBody.Body.String())
	}
	noBody := e.do("POST", "/api/v1/items", map[string]any{
		"title": "No body",
	})
	if noBody.Code != http.StatusCreated {
		t.Fatalf("create noBody status=%d body=%s", noBody.Code, noBody.Body.String())
	}

	res := e.do("GET", "/api/v1/search?kind=all", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", res.Code, res.Body.String())
	}
	env := decodeEnvelope(t, res)
	var items []itemJSON
	if err := json.Unmarshal(env.Data, &items); err != nil {
		t.Fatalf("decode search results: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	foundWithBody, foundNoBody := false, false
	for _, it := range items {
		title, _ := it["title"].(string)
		notesText, _ := it["notes_text"].(string)
		if _, ok := it["notes_text"]; !ok {
			t.Errorf("item %v missing notes_text field entirely", it)
		}
		switch title {
		case "Has body":
			foundWithBody = true
			if !strings.Contains(notesText, "Some real content.") {
				t.Errorf("notes_text for %q = %q, want it to contain body content", title, notesText)
			}
		case "No body":
			foundNoBody = true
			if notesText != "" {
				t.Errorf("notes_text for bodyless item = %q, want \"\"", notesText)
			}
		}
	}
	if !foundWithBody || !foundNoBody {
		t.Fatalf("did not find both expected items: foundWithBody=%v foundNoBody=%v", foundWithBody, foundNoBody)
	}
}

func TestHandleListInboxIncludesNotesText(t *testing.T) {
	e := newTestEnv(t)

	created := e.do("POST", "/api/v1/inbox", map[string]any{
		"title":    "Inbox capture",
		"notes_md": "Quick thought worth keeping.",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create inbox item status=%d body=%s", created.Code, created.Body.String())
	}

	res := e.do("GET", "/api/v1/inbox", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list inbox status=%d body=%s", res.Code, res.Body.String())
	}
	env := decodeEnvelope(t, res)
	var items []itemJSON
	if err := json.Unmarshal(env.Data, &items); err != nil {
		t.Fatalf("decode inbox results: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	notesText, _ := items[0]["notes_text"].(string)
	if !strings.Contains(notesText, "Quick thought worth keeping.") {
		t.Errorf("notes_text=%q, want it to contain the inbox item's body", notesText)
	}
}
