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

// TestHandleGetBackrefs covers the GET /api/v1/items/{id}/backrefs route
// added for CW-20260816-0038: linking item found, notes_text present on it,
// empty-but-200 when nothing links to an existing item, and 404 when the
// target item itself doesn't exist (mirroring handleGetItem's convention).
func TestHandleGetBackrefs(t *testing.T) {
	e := newTestEnv(t)

	target := e.do("POST", "/api/v1/items", map[string]any{
		"title": "Target note",
	})
	if target.Code != http.StatusCreated {
		t.Fatalf("create target status=%d body=%s", target.Code, target.Body.String())
	}
	targetEnv := decodeEnvelope(t, target)
	var targetItem itemJSON
	if err := json.Unmarshal(targetEnv.Data, &targetItem); err != nil {
		t.Fatalf("decode target item: %v", err)
	}
	targetID := int64(targetItem["id"].(float64))

	// A wikilink node pointing at targetID, matching the shape the frontend's
	// WikilinkExtension produces and ingest.ExtractRefIDs consumes.
	linkerDoc := map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": "See "},
					map[string]any{
						"type": "wikilink",
						"attrs": map[string]any{
							"id":      targetID,
							"label":   "Target note",
							"refType": "note",
						},
					},
				},
			},
		},
	}
	docJSON, err := json.Marshal(linkerDoc)
	if err != nil {
		t.Fatalf("marshal linker doc: %v", err)
	}

	linker := e.do("POST", "/api/v1/items", map[string]any{
		"title":     "Linking note",
		"notes_doc": string(docJSON),
	})
	if linker.Code != http.StatusCreated {
		t.Fatalf("create linker status=%d body=%s", linker.Code, linker.Body.String())
	}
	linkerEnv := decodeEnvelope(t, linker)
	var linkerItem itemJSON
	if err := json.Unmarshal(linkerEnv.Data, &linkerItem); err != nil {
		t.Fatalf("decode linker item: %v", err)
	}
	linkerID := int64(linkerItem["id"].(float64))

	// An unrelated item must not appear in target's backrefs.
	unrelated := e.do("POST", "/api/v1/items", map[string]any{"title": "Unrelated note"})
	if unrelated.Code != http.StatusCreated {
		t.Fatalf("create unrelated status=%d body=%s", unrelated.Code, unrelated.Body.String())
	}

	// Found case: backrefs for target returns exactly the linking item, with
	// notes_text present.
	got := e.do("GET", "/api/v1/items/"+strconv.FormatInt(targetID, 10)+"/backrefs", nil)
	if got.Code != http.StatusOK {
		t.Fatalf("backrefs status=%d body=%s", got.Code, got.Body.String())
	}
	gotEnv := decodeEnvelope(t, got)
	var backrefs []itemJSON
	if err := json.Unmarshal(gotEnv.Data, &backrefs); err != nil {
		t.Fatalf("decode backrefs: %v", err)
	}
	if len(backrefs) != 1 {
		t.Fatalf("got %d backrefs, want 1; backrefs=%v", len(backrefs), backrefs)
	}
	if id := int64(backrefs[0]["id"].(float64)); id != linkerID {
		t.Errorf("backref id=%d, want %d", id, linkerID)
	}
	if _, ok := backrefs[0]["notes_text"]; !ok {
		t.Errorf("backref item missing notes_text field: %v", backrefs[0])
	}
	if backrefs[0]["title"] != "Linking note" {
		t.Errorf("backref title=%v, want %q", backrefs[0]["title"], "Linking note")
	}

	// Empty-but-200 case: an item that exists but has no backrefs.
	emptyRes := e.do("GET", "/api/v1/items/"+strconv.FormatInt(linkerID, 10)+"/backrefs", nil)
	if emptyRes.Code != http.StatusOK {
		t.Fatalf("empty backrefs status=%d body=%s", emptyRes.Code, emptyRes.Body.String())
	}
	emptyEnv := decodeEnvelope(t, emptyRes)
	var empty []itemJSON
	if err := json.Unmarshal(emptyEnv.Data, &empty); err != nil {
		t.Fatalf("decode empty backrefs: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("got %d backrefs for linker (which nothing links to), want 0", len(empty))
	}

	// Not-found case: the target item ID doesn't exist at all.
	missing := e.do("GET", "/api/v1/items/999999/backrefs", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing-target backrefs status=%d, want 404; body=%s", missing.Code, missing.Body.String())
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

// createItemID posts a minimal item and returns its ID, failing the test on
// any error along the way.
func createItemID(t *testing.T, e *testEnv, path string, body map[string]any) int64 {
	t.Helper()
	res := e.do("POST", path, body)
	if res.Code != http.StatusCreated {
		t.Fatalf("create (%s) status=%d body=%s", path, res.Code, res.Body.String())
	}
	env := decodeEnvelope(t, res)
	var item itemJSON
	if err := json.Unmarshal(env.Data, &item); err != nil {
		t.Fatalf("decode created item: %v", err)
	}
	return int64(item["id"].(float64))
}

// TestHandleListItemIDs is the deletion/change-signal endpoint's HTTP-level
// contract test: create items across every "still exists but not plain"
// state (archived, completed, inbox) plus a plain item and one that's about
// to be hard-deleted, then confirm GET /api/v1/items/ids reflects exactly
// what currently exists — the deleted item's ID is absent, everything else
// (regardless of archived/completed/inbox status) is present — and that
// each entry in the response is the minimal {id, updated_at} shape with no
// title/notes/taxonomy fields leaking through.
func TestHandleListItemIDs(t *testing.T) {
	e := newTestEnv(t)

	plainID := createItemID(t, e, "/api/v1/items", map[string]any{"title": "plain item"})

	archivedID := createItemID(t, e, "/api/v1/items", map[string]any{"title": "will be archived"})
	if r := e.do("POST", "/api/v1/items/"+strconv.FormatInt(archivedID, 10)+"/archive", map[string]any{"archived": true}); r.Code != http.StatusOK {
		t.Fatalf("archive status=%d body=%s", r.Code, r.Body.String())
	}

	completedID := createItemID(t, e, "/api/v1/items", map[string]any{"title": "will be completed"})
	if r := e.do("POST", "/api/v1/items/"+strconv.FormatInt(completedID, 10)+"/complete", map[string]any{"completed": true}); r.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", r.Code, r.Body.String())
	}

	inboxID := createItemID(t, e, "/api/v1/inbox", map[string]any{"title": "inbox capture"})

	deletedID := createItemID(t, e, "/api/v1/items", map[string]any{"title": "about to be deleted"})
	if r := e.do("DELETE", "/api/v1/items/"+strconv.FormatInt(deletedID, 10), nil); r.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", r.Code, r.Body.String())
	}

	res := e.do("GET", "/api/v1/items/ids", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list ids status=%d body=%s", res.Code, res.Body.String())
	}
	env := decodeEnvelope(t, res)
	var stamps []itemJSON
	if err := json.Unmarshal(env.Data, &stamps); err != nil {
		t.Fatalf("decode ids result: %v", err)
	}

	seen := map[int64]bool{}
	for _, s := range stamps {
		id := int64(s["id"].(float64))
		seen[id] = true

		// Minimal shape: exactly {id, updated_at}, nothing else.
		if len(s) != 2 {
			t.Errorf("id %d: entry has %d fields, want exactly 2 (id, updated_at); entry=%v", id, len(s), s)
		}
		if _, ok := s["updated_at"]; !ok {
			t.Errorf("id %d: entry missing updated_at; entry=%v", id, s)
		}
		for _, leaked := range []string{"title", "notes_doc", "notes_html", "notes_text", "kind", "tags", "projects", "contexts"} {
			if _, ok := s[leaked]; ok {
				t.Errorf("id %d: entry unexpectedly includes %q; entry=%v", id, leaked, s)
			}
		}
	}

	for _, want := range []int64{plainID, archivedID, completedID, inboxID} {
		if !seen[want] {
			t.Errorf("id %d (still exists) missing from /api/v1/items/ids result: %v", want, stamps)
		}
	}
	if seen[deletedID] {
		t.Errorf("deleted id %d still present in /api/v1/items/ids result: %v", deletedID, stamps)
	}
}

// TestHandleListItemIDsKindFilter confirms ?kind= narrows the result, and
// that the default (kind omitted) returns every kind — unlike
// /api/v1/search, whose store-level default is kind=todo.
func TestHandleListItemIDsKindFilter(t *testing.T) {
	e := newTestEnv(t)

	todoID := createItemID(t, e, "/api/v1/items", map[string]any{"title": "a todo"})
	createItemID(t, e, "/api/v1/items", map[string]any{"title": "a note", "kind": "note"})

	res := e.do("GET", "/api/v1/items/ids?kind=todo", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("list ids status=%d body=%s", res.Code, res.Body.String())
	}
	env := decodeEnvelope(t, res)
	var stamps []itemJSON
	if err := json.Unmarshal(env.Data, &stamps); err != nil {
		t.Fatalf("decode ids result: %v", err)
	}
	if len(stamps) != 1 || int64(stamps[0]["id"].(float64)) != todoID {
		t.Fatalf("kind=todo got %v, want exactly [%d]", stamps, todoID)
	}

	all := e.do("GET", "/api/v1/items/ids", nil)
	if all.Code != http.StatusOK {
		t.Fatalf("list ids (no filter) status=%d body=%s", all.Code, all.Body.String())
	}
	allEnv := decodeEnvelope(t, all)
	var allStamps []itemJSON
	if err := json.Unmarshal(allEnv.Data, &allStamps); err != nil {
		t.Fatalf("decode ids result: %v", err)
	}
	if len(allStamps) != 2 {
		t.Fatalf("no-filter got %d ids, want 2 (default is every kind); stamps=%v", len(allStamps), allStamps)
	}
}
