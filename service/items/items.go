// Package items is the business-logic layer for item create/update/search.
//
// Why this exists: GUI (app.go), HTTP API (api.go), CLI (cli/cli.go), and the
// AI chat bridge (chat/bridge.go) each accept item payloads in different shapes
// — Wails-encoded structs, JSON HTTP bodies, CLI flags, model-tool inputs —
// but they all need the same pre-store work: convert markdown/HTML notes to
// PM JSON via the ingest package, default the kind to "todo", default the
// section to "anytime", and so on. Before this package, each consumer
// duplicated those steps (three independent notes-input resolvers, four kind
// defaulters), making behavior drift easy and bug fixes painful (see the
// v1.3.0 silent-backfill incident in CHANGELOG).
//
// The Service is stateless and store-agnostic: each method takes the
// *store.Store the caller has already chosen for the request. Vault routing
// (HTTP's X-Vault-ID header, CLI's --vault flag, GUI's active-vs-inbox split)
// is consumer-specific and stays in the consumer.
package items

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hollis-labs/nil/ingest"
	"github.com/hollis-labs/nil/store"
)

// NotesHTMLVersion stamps the renderer version on every item the service
// creates or updates. Bump when the ingest HTML renderer changes shape in a
// way that should invalidate caches.
const NotesHTMLVersion = 1

// Service holds no state. A single zero-value Service is safe to reuse.
type Service struct{}

// New constructs a Service. Returning a pointer leaves room to add
// dependencies later (renderer registry, metrics sink) without churn at the
// call sites.
func New() *Service { return &Service{} }

// ErrInvalidNotesFormat is returned by ResolveNotesInputCLI when --format is
// not one of md|html|json.
var ErrInvalidNotesFormat = errors.New("unknown notes format (want md|html|json)")

// CreateInput is the unified input for Service.Create. Callers fill the
// fields their transport supplied; missing fields take service defaults.
//
// Notes precedence: NotesDoc > NotesMD > NotesHTML. Only one is consulted;
// the others are ignored. All three empty means the item has no body.
type CreateInput struct {
	Title      string
	Kind       string // "" → "todo"
	Section    string // "" → "anytime"
	Priority   *string
	DueAt      *string
	Threshold  *string
	Recurrence *string
	Projects   []string
	Contexts   []string
	Tags       []string
	Pinned     bool
	Inbox      bool
	APISource  string
	SourceLine string

	// ExternalRef is an optional writer-supplied idempotency key (see
	// store.Item.ExternalRef). Empty means "no external correlation" — the
	// default, unaffected path for every caller that doesn't set it. When
	// non-empty and an item with the same external_ref already exists in
	// the target vault, Create (and CreateBatch) update that existing row
	// instead of inserting a duplicate.
	ExternalRef string

	NotesDoc  string
	NotesMD   string
	NotesHTML string
}

// NullableString lets UpdatePatch distinguish "field omitted" (Set=false)
// from "clear the column to NULL" (Set=true, Value=nil) from "set to this
// value" (Set=true, Value=&v). HTTP and CLI callers map their own
// null-marker types into this shape before invoking the service.
type NullableString struct {
	Value *string
	Set   bool
}

// UpdatePatch is the partial-update input for Service.UpdatePatch. Nil
// pointer fields mean "leave unchanged"; non-nil means "replace with this".
// For slice fields, a nil *[]string means "leave unchanged" while a non-nil
// pointer (including to an empty slice) means "replace".
//
// Use NullableString fields for columns that can be NULL (Priority, DueAt,
// Threshold, Recurrence) so callers can explicitly clear them.
//
// Notes precedence matches CreateInput. If any of NotesDoc/NotesMD/NotesHTML
// is non-nil the notes are re-resolved and overwritten; otherwise notes are
// left untouched.
type UpdatePatch struct {
	Title    *string
	Kind     *string
	Section  *string
	Pinned   *bool
	Projects *[]string
	Contexts *[]string
	Tags     *[]string

	// ExternalRef lets a caller explicitly set/change/clear the item's
	// external_ref via a known ID (nil = leave unchanged, non-nil = replace,
	// including with "" to clear). This is distinct from CreateInput's
	// external_ref matching: that's for a caller that doesn't know Nil's
	// internal ID and wants create-or-update-by-external_ref; this is for a
	// caller that already has the ID and wants to (re)assign the key itself.
	ExternalRef *string

	Priority   NullableString
	DueAt      NullableString
	Threshold  NullableString
	Recurrence NullableString

	NotesDoc  *string
	NotesMD   *string
	NotesHTML *string
}

// Create builds a *store.Item from the input, applies kind/section defaults,
// resolves notes input via the ingest pipeline, and writes via st.CreateItem.
// The store validates kind against the kinds registry.
func (s *Service) Create(ctx context.Context, st *store.Store, input CreateInput) (*store.Item, error) {
	item, err := s.buildItem(input)
	if err != nil {
		return nil, err
	}
	return st.CreateItem(ctx, item)
}

// CreateBatch creates every input in a single all-or-nothing transaction
// (see store.Store.CreateItemsBatch for the transactional/rollback-on-any-
// failure rationale this shares). Each input gets the exact same
// kind/section defaulting, notes-input resolution, and external_ref
// dedup-or-insert behavior as a single Create call — the batch and
// single-item paths are semantically identical, just wrapped in one
// transaction. An input-resolution failure (e.g. malformed notes_md/html at
// index i) is reported before any DB work happens and aborts the whole
// batch, consistent with the all-or-nothing contract.
func (s *Service) CreateBatch(ctx context.Context, st *store.Store, inputs []CreateInput) ([]store.Item, error) {
	items := make([]store.Item, len(inputs))
	for i, input := range inputs {
		item, err := s.buildItem(input)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		items[i] = *item
	}
	return st.CreateItemsBatch(ctx, items)
}

// buildItem applies CreateInput's kind/section defaults and resolves its
// notes input into the (doc, html, version) triplet, producing the
// *store.Item that Create and CreateBatch both hand to the store layer. No
// store I/O happens here — it's pure input shaping, safe to call before a
// transaction is open (as CreateBatch does, to fail the whole batch fast on
// a bad input without touching the DB).
func (s *Service) buildItem(input CreateInput) (*store.Item, error) {
	kind := input.Kind
	if kind == "" {
		kind = "todo"
	}
	section := input.Section
	if section == "" {
		section = "anytime"
	}
	doc, htmlStr, version, err := s.ResolveNotesInput(input.NotesDoc, input.NotesMD, input.NotesHTML)
	if err != nil {
		return nil, fmt.Errorf("resolving notes input: %w", err)
	}
	return &store.Item{
		Title:            input.Title,
		Kind:             kind,
		Section:          section,
		Priority:         input.Priority,
		DueAt:            input.DueAt,
		Threshold:        input.Threshold,
		Recur:            input.Recurrence,
		Projects:         input.Projects,
		Contexts:         input.Contexts,
		Tags:             input.Tags,
		Pinned:           input.Pinned,
		Inbox:            input.Inbox,
		APISource:        input.APISource,
		ExternalRef:      input.ExternalRef,
		Source:           input.SourceLine,
		NotesDoc:         doc,
		NotesHTML:        htmlStr,
		NotesHTMLVersion: version,
	}, nil
}

// Update replaces an item wholesale. Caller has already populated every field
// (including NotesDoc / NotesHTML); the service does no notes-input resolution
// here. Use this when the caller holds a full *store.Item — typically the GUI,
// which round-trips Items through Wails-generated bindings.
//
// Kind is defaulted to "todo" if empty so wire formats that omit the field
// don't accidentally fail registry validation.
func (s *Service) Update(ctx context.Context, st *store.Store, item *store.Item) error {
	if item.Kind == "" {
		item.Kind = "todo"
	}
	return st.UpdateItem(ctx, item)
}

// UpdatePatch fetches the current item, applies non-nil patch fields, optionally
// re-resolves notes input, and writes the result. Returns the updated item as
// re-fetched from the store (so caller sees fresh updated_at, etc.).
func (s *Service) UpdatePatch(ctx context.Context, st *store.Store, id int64, patch UpdatePatch) (*store.Item, error) {
	existing, err := st.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}

	if patch.Title != nil {
		existing.Title = *patch.Title
	}
	if patch.Kind != nil {
		existing.Kind = *patch.Kind
	}
	if patch.Section != nil {
		existing.Section = *patch.Section
	}
	if patch.Pinned != nil {
		existing.Pinned = *patch.Pinned
	}
	if patch.Projects != nil {
		existing.Projects = *patch.Projects
	}
	if patch.Contexts != nil {
		existing.Contexts = *patch.Contexts
	}
	if patch.Tags != nil {
		existing.Tags = *patch.Tags
	}
	if patch.ExternalRef != nil {
		existing.ExternalRef = *patch.ExternalRef
	}
	if patch.Priority.Set {
		existing.Priority = patch.Priority.Value
	}
	if patch.DueAt.Set {
		existing.DueAt = patch.DueAt.Value
	}
	if patch.Threshold.Set {
		existing.Threshold = patch.Threshold.Value
	}
	if patch.Recurrence.Set {
		existing.Recur = patch.Recurrence.Value
	}

	if patch.NotesDoc != nil || patch.NotesMD != nil || patch.NotesHTML != nil {
		var inDoc, inMD, inHTML string
		if patch.NotesDoc != nil {
			inDoc = *patch.NotesDoc
		}
		if patch.NotesMD != nil {
			inMD = *patch.NotesMD
		}
		if patch.NotesHTML != nil {
			inHTML = *patch.NotesHTML
		}
		doc, htmlStr, version, err := s.ResolveNotesInput(inDoc, inMD, inHTML)
		if err != nil {
			return nil, fmt.Errorf("resolving notes input: %w", err)
		}
		existing.NotesDoc = doc
		existing.NotesHTML = htmlStr
		existing.NotesHTMLVersion = version
	}

	if err := st.UpdateItem(ctx, existing); err != nil {
		return nil, err
	}
	return st.GetItem(ctx, id)
}

// ResolveNotesInput converts whichever notes input format the caller supplied
// into the (doc JSON, html cache, renderer version) tuple needed for storage.
// Precedence: notes_doc > notes_md > notes_html. Returns empty zero values
// when none are provided.
//
// This is the single source of truth for notes ingest across the GUI, HTTP
// API, CLI, and chat bridge.
func (s *Service) ResolveNotesInput(notesDoc, notesMD, notesHTML string) (string, string, int, error) {
	if strings.TrimSpace(notesDoc) != "" {
		htmlStr, err := ingest.DocToHTML(notesDoc)
		if err != nil {
			return "", "", 0, err
		}
		return notesDoc, htmlStr, NotesHTMLVersion, nil
	}
	if strings.TrimSpace(notesMD) != "" {
		doc, err := ingest.MarkdownToDoc(notesMD)
		if err != nil {
			return "", "", 0, err
		}
		htmlStr, err := ingest.DocToHTML(doc)
		if err != nil {
			return "", "", 0, err
		}
		return doc, htmlStr, NotesHTMLVersion, nil
	}
	if strings.TrimSpace(notesHTML) != "" {
		doc, err := ingest.HTMLToDoc(notesHTML)
		if err != nil {
			return "", "", 0, err
		}
		return doc, notesHTML, NotesHTMLVersion, nil
	}
	return "", "", 0, nil
}

// NotesInputFromCLI maps the CLI's --body + --format flags onto the
// notes-input triplet that CreateInput / UpdatePatch consume. Exactly one of
// the returned strings is non-empty; the others are "". Empty body produces
// all-empty output (no error). Unknown format returns ErrInvalidNotesFormat
// wrapping the offending string.
//
// This is a transport-shape helper: it does NOT resolve to PM JSON itself —
// the resolution happens inside Create / UpdatePatch via ResolveNotesInput,
// so consumers don't pay for double conversion.
func (s *Service) NotesInputFromCLI(body, format string) (doc, md, html string, err error) {
	if strings.TrimSpace(body) == "" {
		return "", "", "", nil
	}
	switch strings.ToLower(format) {
	case "", "md", "markdown":
		return "", body, "", nil
	case "html":
		return "", "", body, nil
	case "json", "doc":
		return body, "", "", nil
	default:
		return "", "", "", fmt.Errorf("%w: %q", ErrInvalidNotesFormat, format)
	}
}

// Search applies the service's canonical defaults and delegates to st.Search.
// Defaults:
//   - Kind: "all" if empty (the store would default to "todo"; the service
//     overrides because every real consumer treats "" as "no filter")
//   - PageSize: 50 if zero or negative
//   - SortBy: "created_at" if empty
//   - SortDir: "desc" if empty
//
// All other fields pass through untouched.
func (s *Service) Search(ctx context.Context, st *store.Store, req store.SearchRequest) ([]store.Item, error) {
	return st.Search(ctx, s.applySearchDefaults(req))
}

// ListInbox lists inbox items with the canonical inbox default (PageSize=200
// if unset). Other fields pass through to st.GetInboxItems, which enforces
// inbox=1 and archived=0 regardless of the request.
func (s *Service) ListInbox(ctx context.Context, st *store.Store, req store.SearchRequest) ([]store.Item, error) {
	if req.PageSize <= 0 {
		req.PageSize = 200
	}
	return st.GetInboxItems(ctx, req)
}

// InboxCount returns the count of unprocessed inbox items.
func (s *Service) InboxCount(ctx context.Context, st *store.Store) (int, error) {
	return st.GetInboxCount(ctx)
}

// ProcessInbox clears the inbox flag on an item without moving it between
// vaults. Vault-to-vault relocation is consumer-specific (the GUI moves
// inbox items into the active vault on triage; the HTTP API just clears the
// flag); callers that need a move should do it themselves around this call.
func (s *Service) ProcessInbox(ctx context.Context, st *store.Store, id int64) error {
	return st.ProcessInboxItem(ctx, id)
}

// ItemView augments a store.Item with a plaintext rendering of its body
// (NotesText) for API/CLI/MCP responses. It embeds store.Item by value, so
// every existing field/JSON tag is preserved unchanged — this is a purely
// additive wrapper, not a replacement shape. Callers on every transport
// (HTTP API, CLI, and MCP via its raw-body passthrough of the HTTP API)
// should serialize this type instead of a bare store.Item/[]store.Item once
// they need notes_text in the response.
type ItemView struct {
	store.Item
	NotesText string `json:"notes_text"`
}

// WithText wraps a single item for response shaping, computing notes_text
// via PlainText. This is the single reuse point every surface should call
// instead of invoking ingest.DocToPlainText (or writing its own PM-JSON
// walker) independently.
func (s *Service) WithText(item *store.Item) ItemView {
	return ItemView{Item: *item, NotesText: s.PlainText(item.NotesDoc)}
}

// WithTextSlice applies WithText across a list response (search, inbox).
func (s *Service) WithTextSlice(list []store.Item) []ItemView {
	out := make([]ItemView, len(list))
	for i := range list {
		out[i] = ItemView{Item: list[i], NotesText: s.PlainText(list[i].NotesDoc)}
	}
	return out
}

// PlainText renders a stored notes_doc (TipTap/ProseMirror JSON) to plain
// text by reusing ingest.DocToPlainText directly — no new conversion logic.
// DocToPlainText already powers FTS5 indexing (see ingest/doc.go and
// store/schema.sql's notes_text comment); this method is the single place
// every external-facing surface (HTTP API, CLI, and MCP via its HTTP
// passthrough) goes through, so none of them need to maintain their own
// PM-JSON walker just to get readable text out of a note.
//
// Error handling: an empty notes_doc is NOT an error (UnmarshalDoc treats ""
// as the canonical empty doc, common for inbox-capture items with no body,
// and yields ""). DocToPlainText only errors on genuinely malformed JSON. In
// that case this method swallows the error and returns "" rather than
// propagating it — one corrupt item must not fail an entire list/search
// response for every other item alongside it, and the raw notes_doc is still
// returned unchanged so a caller that cares about the corruption can still
// detect it via that field.
//
// DocToMarkdown: deliberately NOT added alongside this. Plaintext is
// sufficient for today's actual consumers (search/ingest/sync); a
// round-trippable markdown exporter is separate, real work (list/heading/
// mark serialization, escaping) that no concrete consumer has asked for.
// Revisit only if one does.
func (s *Service) PlainText(notesDoc string) string {
	text, err := ingest.DocToPlainText(notesDoc)
	if err != nil {
		return ""
	}
	return text
}

func (s *Service) applySearchDefaults(req store.SearchRequest) store.SearchRequest {
	if req.Kind == "" {
		req.Kind = "all"
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	if req.SortBy == "" {
		req.SortBy = "created_at"
	}
	if req.SortDir == "" {
		req.SortDir = "desc"
	}
	return req
}
