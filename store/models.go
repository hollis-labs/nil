package store

type Item struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Priority  *string `json:"priority,omitempty"`
	Completed bool    `json:"completed"`
	Archived  bool    `json:"archived"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	DueAt     *string `json:"due_at,omitempty"`
	Threshold *string `json:"threshold_at,omitempty"`
	Recur     *string `json:"recurrence_rule,omitempty"`
	Source    string  `json:"source_line"`
	NotesMD   string  `json:"notes_md"`
	NotesText string  `json:"notes_text,omitempty"`
	Section   string  `json:"section"` // now, soon, anytime
	Pinned    bool    `json:"pinned"`
	Type      string  `json:"type"`
	Inbox     bool    `json:"inbox"`
	APISource string  `json:"api_source,omitempty"`

	Projects []string `json:"projects"`
	Contexts []string `json:"contexts"`
	Tags     []string `json:"tags"`
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
	BySection VaultBySection `json:"by_section"`
}

type SearchRequest struct {
	Query      string   `json:"query"`
	Projects   []string `json:"projects"`
	Contexts   []string `json:"contexts"`
	Tags       []string `json:"tags"`
	Statuses   []string `json:"statuses"`
	Priorities []string `json:"priorities"`
	DateFrom   *string  `json:"date_from"`
	DateTo     *string  `json:"date_to"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	SortBy       string   `json:"sort_by"`
	SortDir      string   `json:"sort_dir"`
	Type         string   `json:"type"`
	IncludeInbox bool     `json:"include_inbox"`
}
