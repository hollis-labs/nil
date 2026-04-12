package main

import (
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/store"
	"github.com/hollis-labs/nil/vault"
)

type apiHandler struct {
	cfg      *config.Config
	vaultMgr *vault.Manager
}

// NewAPIServer constructs an *http.Server bound to 127.0.0.1 only.
func NewAPIServer(cfg *config.Config, vm *vault.Manager) *http.Server {
	h := &apiHandler{cfg: cfg, vaultMgr: vm}

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/inbox", h.auth(h.handleCreateInbox))
	mux.Handle("GET /api/v1/inbox", h.auth(h.handleListInbox))
	mux.Handle("GET /api/v1/search", h.auth(h.handleSearch))
	mux.Handle("GET /api/v1/items/{id}", h.auth(h.handleGetItem))

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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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

// --- Route handlers ---

type inboxCreateRequest struct {
	Title    string   `json:"title"`
	NotesMD  string   `json:"notes_md"`
	Priority *string  `json:"priority"`
	DueAt    *string  `json:"due_at"`
	Type     string   `json:"type"`
	Tags     []string `json:"tags"`
	Contexts []string `json:"contexts"`
	Projects []string `json:"projects"`
}

// POST /api/v1/inbox — create an inbox item.
// Always routes to the inbox store regardless of X-Vault-ID.
func (h *apiHandler) handleCreateInbox(w http.ResponseWriter, r *http.Request) {
	var req inboxCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	itemType := req.Type
	if itemType == "" {
		itemType = "todo"
	}

	item := &store.Item{
		Title:     req.Title,
		NotesMD:   req.NotesMD,
		Priority:  req.Priority,
		DueAt:     req.DueAt,
		Type:      itemType,
		Tags:      req.Tags,
		Contexts:  req.Contexts,
		Projects:  req.Projects,
		Inbox:     true,
		APISource: r.Header.Get("X-Agent-Source"),
	}

	created, err := h.vaultMgr.InboxStore().CreateItem(r.Context(), item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create item")
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

	items, err := h.vaultMgr.InboxStore().GetInboxItems(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list inbox items")
		return
	}
	writeJSON(w, http.StatusOK, items)
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

	typeFilter := q.Get("type")
	if typeFilter == "" {
		typeFilter = "all"
	}

	req := store.SearchRequest{
		Query:    q.Get("q"),
		Type:     typeFilter,
		Tags:     splitMultiParam(q["tags"]),
		Contexts: splitMultiParam(q["contexts"]),
		Projects: splitMultiParam(q["projects"]),
		Page:     page,
		PageSize: pageSize,
	}

	items, err := s.Search(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search items")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// GET /api/v1/items/{id} — fetch a single item from the vault resolved by X-Vault-ID header.
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
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get item")
		return
	}
	writeJSON(w, http.StatusOK, item)
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
