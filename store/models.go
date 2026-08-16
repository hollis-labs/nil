package store

type Item struct {
	ID               int64   `json:"id"`
	Title            string  `json:"title"`
	Priority         *string `json:"priority,omitempty"`
	Completed        bool    `json:"completed"`
	Archived         bool    `json:"archived"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	DueAt            *string `json:"due_at,omitempty"`
	Threshold        *string `json:"threshold_at,omitempty"`
	Recur            *string `json:"recurrence_rule,omitempty"`
	Source           string  `json:"source_line"`
	NotesDoc         string  `json:"notes_doc"`           // TipTap PM JSON, source of truth
	NotesHTML        string  `json:"notes_html"`          // write-time render cache
	NotesHTMLVersion int     `json:"notes_html_version"`  // bump when renderer changes
	Section          string  `json:"section"`             // now, soon, anytime
	Pinned           bool    `json:"pinned"`
	Kind             string  `json:"kind"` // todo, note, scratch, ... (kinds registry)
	Inbox            bool    `json:"inbox"`
	APISource        string  `json:"api_source,omitempty"`

	Projects []string `json:"projects"`
	Contexts []string `json:"contexts"`
	Tags     []string `json:"tags"`
}

// Kind represents a row from the kinds registry. Core kinds (todo, note,
// scratch) have is_core=1 and a nil PluginID. Plugin-registered kinds
// reference a plugin via PluginID.
type Kind struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Icon        string  `json:"icon"`
	DefaultView string  `json:"default_view"`
	PluginID    *string `json:"plugin_id,omitempty"`
	IsCore      bool    `json:"is_core"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// TaxonomyItem holds a taxonomy name and the number of items that use it.
type TaxonomyItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// VaultBySection holds item counts per section (non-inbox, non-archived items only).
type VaultBySection struct {
	Now     int `json:"now"`
	Soon    int `json:"soon"`
	Anytime int `json:"anytime"`
}

// VaultStats holds aggregate counts for a vault.
type VaultStats struct {
	Total     int            `json:"total"`
	Open      int            `json:"open"`
	Completed int            `json:"completed"`
	Archived  int            `json:"archived"`
	Overdue   int            `json:"overdue"`
	Inbox     int            `json:"inbox"`
	Notes     int            `json:"notes"`
	Todos     int            `json:"todos"`
	Scratch   int            `json:"scratch"`
	BySection VaultBySection `json:"by_section"`
}

type SearchRequest struct {
	Query        string   `json:"query"`
	Projects     []string `json:"projects"`
	Contexts     []string `json:"contexts"`
	Tags         []string `json:"tags"`
	Statuses     []string `json:"statuses"`
	Priorities   []string `json:"priorities"`
	DateFrom     *string  `json:"date_from"`
	DateTo       *string  `json:"date_to"`
	Page         int      `json:"page"`
	PageSize     int      `json:"page_size"`
	SortBy       string   `json:"sort_by"`
	SortDir      string   `json:"sort_dir"`
	Kind         string   `json:"kind"`
	IncludeInbox bool     `json:"include_inbox"`

	// UpdatedSince filters results to items whose updated_at is on or after
	// this instant. Accepts RFC3339 (e.g. "2026-08-01T00:00:00Z" or with an
	// offset like "2026-08-01T00:00:00-07:00"); Store.Search parses it,
	// converts to UTC, and compares against the naive
	// "YYYY-MM-DD HH:MM:SS" UTC strings Nil actually stores in updated_at
	// (see schema.sql's todos_update_ts trigger, which uses SQLite's
	// datetime('now') — always UTC, no timezone suffix). RFC3339 was chosen
	// over the raw SQLite format because it's the standard external API
	// consumers expect; normalizing once at the store boundary keeps every
	// caller (HTTP API, CLI, MCP) from having to know Nil's internal storage
	// format. Empty string means no filter (default: unchanged, all rows
	// regardless of update time).
	UpdatedSince string `json:"updated_since"`
}
