package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
	"github.com/hollis-labs/nil/vault"
)

type apiHandler struct {
	cfg      *config.Config
	vaultMgr *vault.Manager
	svc      *items.Service
}

// NewAPIServer constructs an *http.Server bound to 127.0.0.1 only.
func NewAPIServer(cfg *config.Config, vm *vault.Manager) *http.Server {
	h := &apiHandler{cfg: cfg, vaultMgr: vm, svc: items.New()}

	mux := http.NewServeMux()

	// Inbox routes
	mux.Handle("POST /api/v1/inbox", h.auth(h.handleCreateInbox))
	mux.Handle("GET /api/v1/inbox", h.auth(h.handleListInbox))
	mux.Handle("POST /api/v1/inbox/{id}/process", h.auth(h.handleProcessInboxItem))

	// Item CRUD routes
	mux.Handle("POST /api/v1/items", h.auth(h.handleCreateItem))
	mux.Handle("GET /api/v1/items/{id}", h.auth(h.handleGetItem))
	mux.Handle("PUT /api/v1/items/{id}", h.auth(h.handleUpdateItem))
	mux.Handle("DELETE /api/v1/items/{id}", h.auth(h.handleDeleteItem))

	// Item action routes
	mux.Handle("POST /api/v1/items/{id}/complete", h.auth(h.handleToggleComplete))
	mux.Handle("POST /api/v1/items/{id}/archive", h.auth(h.handleArchive))

	// Search and taxonomy
	mux.Handle("GET /api/v1/search", h.auth(h.handleSearch))
	mux.Handle("GET /api/v1/taxonomy", h.auth(h.handleTaxonomy))

	// Vault listing
	mux.Handle("GET /api/v1/vaults", h.auth(h.handleListVaults))

	return &http.Server{
		Addr:         net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.APIPort)),
		Handler:      h.cors(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
}

// storeForRequest resolves the correct store from the X-Vault-ID header.
// If the header is absent, returns the active vault store.
// If the header value is "inbox", returns the inbox store.
// Otherwise, looks up the vault by ID.
func (h *apiHandler) storeForRequest(r *http.Request) *store.Store {
	vaultID := r.Header.Get("X-Vault-ID")
	switch vaultID {
	case "":
		return h.vaultMgr.ActiveStore()
	case "inbox":
		return h.vaultMgr.InboxStore()
	default:
		s, err := h.vaultMgr.StoreForID(vaultID)
		if err != nil {
			return h.vaultMgr.ActiveStore()
		}
		return s
	}
}

// --- Middleware ---

func (h *apiHandler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Agent-Source, X-Vault-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *apiHandler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != h.cfg.APIKey {
			writeError(w, http.StatusUnauthorized, "invalid or missing API key")
			return
		}
		next(w, r)
	}
}

// --- Response helpers ---

type apiResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{OK: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{OK: false, Error: msg})
}

// --- Request structs ---

// Notes input fields on create/update requests accept exactly one of:
//   - notes_doc:  TipTap PM JSON (source of truth, no conversion)
//   - notes_md:   markdown source (server converts via ingest)
//   - notes_html: HTML source (server converts via ingest)
// Precedence when multiple are provided: notes_doc > notes_md > notes_html.

type inboxCreateRequest struct {
	Title     string   `json:"title"`
	NotesDoc  string   `json:"notes_doc"`
	NotesMD   string   `json:"notes_md"`
	NotesHTML string   `json:"notes_html"`
	Priority  *string  `json:"priority"`
	DueAt     *string  `json:"due_at"`
	Kind      string   `json:"kind"`
	Tags      []string `json:"tags"`
	Contexts  []string `json:"contexts"`
	Projects  []string `json:"projects"`
}

type itemCreateRequest struct {
	Title     string   `json:"title"`
	NotesDoc  string   `json:"notes_doc"`
	NotesMD   string   `json:"notes_md"`
	NotesHTML string   `json:"notes_html"`
	Priority  *string  `json:"priority"`
	DueAt     *string  `json:"due_at"`
	Kind      string   `json:"kind"`
	Section   string   `json:"section"`
	Pinned    bool     `json:"pinned"`
	Tags      []string `json:"tags"`
	Contexts  []string `json:"contexts"`
	Projects  []string `json:"projects"`
}

// nullableString distinguishes "field omitted" (IsSet=false) from "field set to null"
// (IsSet=true, Value=nil) and "field set to a value" (IsSet=true, Value=non-nil).
// This is needed because encoding/json decodes both omitted and explicit null to nil
// for *string fields, making it impossible to tell whether the caller intends to clear
// a nullable column or simply leave it unchanged.
type nullableString struct {
	Value *string
	IsSet bool
}

func (n *nullableString) UnmarshalJSON(data []byte) error {
	n.IsSet = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n.Value = &s
	return nil
}

// itemUpdateRequest uses pointers for all optional fields so we can distinguish
// "not provided" from "set to zero value". For slice fields, *[]string lets callers
// send null (leave unchanged) vs [] (clear). Priority and DueAt use nullableString
// so that an explicit JSON null can clear the column.
type itemUpdateRequest struct {
	Title     *string        `json:"title"`
	NotesDoc  *string        `json:"notes_doc"`
	NotesMD   *string        `json:"notes_md"`
	NotesHTML *string        `json:"notes_html"`
	Priority  nullableString `json:"priority"`
	DueAt     nullableString `json:"due_at"`
	Kind      *string        `json:"kind"`
	Section   *string        `json:"section"`
	Pinned    *bool          `json:"pinned"`
	Tags      *[]string      `json:"tags"`
	Contexts  *[]string      `json:"contexts"`
	Projects  *[]string      `json:"projects"`
}

type toggleCompleteRequest struct {
	Completed bool `json:"completed"`
}

type archiveRequest struct {
	Archived bool `json:"archived"`
}

// --- Route handlers ---

// POST /api/v1/inbox — create an inbox item.
// Always routes to the inbox store regardless of X-Vault-ID.
func (h *apiHandler) handleCreateInbox(w http.ResponseWriter, r *http.Request) {
	var req inboxCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	created, err := h.svc.Create(r.Context(), h.vaultMgr.InboxStore(), items.CreateInput{
		Title:     req.Title,
		Kind:      req.Kind,
		Priority:  req.Priority,
		DueAt:     req.DueAt,
		Tags:      req.Tags,
		Contexts:  req.Contexts,
		Projects:  req.Projects,
		Inbox:     true,
		APISource: r.Header.Get("X-Agent-Source"),
		NotesDoc:  req.NotesDoc,
		NotesMD:   req.NotesMD,
		NotesHTML: req.NotesHTML,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create item: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// GET /api/v1/inbox — list inbox items.
// Always reads from the inbox store regardless of X-Vault-ID.
func (h *apiHandler) handleListInbox(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	req := store.SearchRequest{
		Query:    q.Get("q"),
		Page:     page,
		PageSize: pageSize,
	}

	listed, err := h.svc.ListInbox(r.Context(), h.vaultMgr.InboxStore(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list inbox items")
		return
	}
	writeJSON(w, http.StatusOK, listed)
}

// POST /api/v1/inbox/{id}/process — mark an inbox item as processed.
// Calls ProcessInboxItem on the inbox store; responds 204 on success.
func (h *apiHandler) handleProcessInboxItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	inbox := h.vaultMgr.InboxStore()
	if _, err := inbox.GetItem(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch item")
		return
	}
	if err := inbox.ProcessInboxItem(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process inbox item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/items — create a regular (non-inbox) item in the vault resolved by X-Vault-ID.
func (h *apiHandler) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req itemCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	s := h.storeForRequest(r)
	if s == nil {
		writeError(w, http.StatusServiceUnavailable, "vault not available")
		return
	}

	created, err := h.svc.Create(r.Context(), s, items.CreateInput{
		Title:     req.Title,
		Kind:      req.Kind,
		Section:   req.Section,
		Priority:  req.Priority,
		DueAt:     req.DueAt,
		Pinned:    req.Pinned,
		Tags:      req.Tags,
		Contexts:  req.Contexts,
		Projects:  req.Projects,
		APISource: r.Header.Get("X-Agent-Source"),
		NotesDoc:  req.NotesDoc,
		NotesMD:   req.NotesMD,
		NotesHTML: req.NotesHTML,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create item: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// GET /api/v1/items/{id} — fetch a single item from the vault resolved by X-Vault-ID.
func (h *apiHandler) handleGetItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	s := h.storeForRequest(r)
	if s == nil {
		writeError(w, http.StatusServiceUnavailable, "vault not available")
		return
	}

	item, err := s.GetItem(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get item")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// PUT /api/v1/items/{id} — partial update of an existing item.
// Only non-nil fields in the request body are applied to the stored item.
// For slice fields, null means "leave unchanged" and [] means "clear".
func (h *apiHandler) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	var req itemUpdateRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	s := h.storeForRequest(r)
	if s == nil {
		writeError(w, http.StatusServiceUnavailable, "vault not available")
		return
	}

	patch := items.UpdatePatch{
		Title:     req.Title,
		Kind:      req.Kind,
		Section:   req.Section,
		Pinned:    req.Pinned,
		Tags:      req.Tags,
		Contexts:  req.Contexts,
		Projects:  req.Projects,
		Priority:  items.NullableString{Set: req.Priority.IsSet, Value: req.Priority.Value},
		DueAt:     items.NullableString{Set: req.DueAt.IsSet, Value: req.DueAt.Value},
		NotesDoc:  req.NotesDoc,
		NotesMD:   req.NotesMD,
		NotesHTML: req.NotesHTML,
	}

	updated, err := h.svc.UpdatePatch(r.Context(), s, id, patch)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update item: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// DELETE /api/v1/items/{id} — permanently delete an item; responds 204.
func (h *apiHandler) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	s := h.storeForRequest(r)
	if s == nil {
		writeError(w, http.StatusServiceUnavailable, "vault not available")
		return
	}

	if _, err := s.GetItem(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch item")
		return
	}
	if err := s.DeleteItem(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/items/{id}/complete — toggle the completion state of an item.
// Body: {"completed": bool}
func (h *apiHandler) handleToggleComplete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	var req toggleCompleteRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	s := h.storeForRequest(r)
	if s == nil {
		writeError(w, http.StatusServiceUnavailable, "vault not available")
		return
	}

	if _, fetchErr := s.GetItem(r.Context(), id); fetchErr != nil {
		if errors.Is(fetchErr, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch item")
		return
	}
	if toggleErr := s.ToggleComplete(r.Context(), id, req.Completed); toggleErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle completion")
		return
	}

	item, err := s.GetItem(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated item")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// POST /api/v1/items/{id}/archive — archive or unarchive an item.
// Body: {"archived": bool}
func (h *apiHandler) handleArchive(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	var req archiveRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	s := h.storeForRequest(r)
	if s == nil {
		writeError(w, http.StatusServiceUnavailable, "vault not available")
		return
	}

	if _, fetchErr := s.GetItem(r.Context(), id); fetchErr != nil {
		if errors.Is(fetchErr, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch item")
		return
	}
	if archErr := s.Archive(r.Context(), id, req.Archived); archErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to archive item")
		return
	}

	item, err := s.GetItem(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated item")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// GET /api/v1/search — search items in the vault resolved by X-Vault-ID header.
func (h *apiHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	s := h.storeForRequest(r)
	if s == nil {
		writeError(w, http.StatusServiceUnavailable, "vault not available")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	// Accept both `kind` (preferred) and legacy `type` for backward compat.
	kindFilter := q.Get("kind")
	if kindFilter == "" {
		kindFilter = q.Get("type")
	}

	req := store.SearchRequest{
		Query:    q.Get("q"),
		Kind:     kindFilter,
		Tags:     splitMultiParam(q["tags"]),
		Contexts: splitMultiParam(q["contexts"]),
		Projects: splitMultiParam(q["projects"]),
		Page:     page,
		PageSize: pageSize,
	}

	results, err := h.svc.Search(r.Context(), s, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search items")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// GET /api/v1/taxonomy — return projects, contexts, and tags with item counts.
// Vault resolved by X-Vault-ID header.
func (h *apiHandler) handleTaxonomy(w http.ResponseWriter, r *http.Request) {
	s := h.storeForRequest(r)
	if s == nil {
		writeError(w, http.StatusServiceUnavailable, "vault not available")
		return
	}

	projects, contexts, tags, err := s.GetTaxonomyWithCounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get taxonomy")
		return
	}

	type taxonomyResponse struct {
		Projects []store.TaxonomyItem `json:"projects"`
		Contexts []store.TaxonomyItem `json:"contexts"`
		Tags     []store.TaxonomyItem `json:"tags"`
	}

	// Ensure nil slices serialize as [] rather than null
	if projects == nil {
		projects = []store.TaxonomyItem{}
	}
	if contexts == nil {
		contexts = []store.TaxonomyItem{}
	}
	if tags == nil {
		tags = []store.TaxonomyItem{}
	}

	writeJSON(w, http.StatusOK, taxonomyResponse{
		Projects: projects,
		Contexts: contexts,
		Tags:     tags,
	})
}

// vaultInfo extends config.Vault with an Active flag indicating the current vault.
type vaultInfo struct {
	config.Vault
	Active bool `json:"active"`
}

// GET /api/v1/vaults — list all registered vaults with an active flag.
func (h *apiHandler) handleListVaults(w http.ResponseWriter, r *http.Request) {
	vaults := h.vaultMgr.GetVaults()
	activeID := h.vaultMgr.GetActiveVaultID()

	result := make([]vaultInfo, len(vaults))
	for i, v := range vaults {
		result[i] = vaultInfo{
			Vault:  v,
			Active: v.ID == activeID,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// splitMultiParam handles both comma-separated values and repeated query params.
func splitMultiParam(values []string) []string {
	var result []string
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}
