package store

type Todo struct {
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
	Section   string  `json:"section"` // now, soon, anytime
	Pinned    bool    `json:"pinned"`
	Type      string  `json:"type"`

	Projects []string `json:"projects"`
	Contexts []string `json:"contexts"`
	Tags     []string `json:"tags"`
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
	SortBy     string   `json:"sort_by"`
	SortDir    string   `json:"sort_dir"`
	Type       string   `json:"type"`
}
