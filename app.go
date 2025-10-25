package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"todo-app/config"
	"todo-app/parse"
	"todo-app/store"
)

type App struct {
	ctx        context.Context
	Store      *store.Store
	needsSetup bool
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load config
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// If no database path configured, signal setup needed
	if cfg.DatabasePath == "" {
		a.needsSetup = true
		return
	}

	// Open database
	s, err := store.Open(ctx, cfg.DatabasePath)
	if err != nil {
		panic(err)
	}
	a.Store = s
}

// NeedsSetup returns true if the app needs initial database configuration
func (a *App) NeedsSetup() bool {
	return a.needsSetup
}

// GetDatabasePath returns the current configured database path
func (a *App) GetDatabasePath() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.DatabasePath == "" {
		return config.GetDefaultDatabasePath(), nil
	}
	return cfg.DatabasePath, nil
}

// GetDefaultDatabasePath returns the OS-appropriate default path
func (a *App) GetDefaultDatabasePath() string {
	return config.GetDefaultDatabasePath()
}

// SetDatabasePath validates and saves the database path, then opens the database
func (a *App) SetDatabasePath(path string) error {
	// Validate path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}

	// Check if writable
	testFile := path + "/.planck-write-test"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("directory not writable: %s", path)
	}
	os.Remove(testFile)

	// Save config
	cfg := &config.Config{DatabasePath: path}
	if err := config.Save(cfg); err != nil {
		return err
	}

	// Open database if we have a context
	if a.ctx != nil {
		s, err := store.Open(a.ctx, path)
		if err != nil {
			return err
		}
		a.Store = s
		a.needsSetup = false
	}

	return nil
}

func (a *App) CreateTodoFromLine(line string) (*store.Todo, error) {
	p := parse.ParseLine(line)
	t := &store.Todo{
		Title:     p.Title,
		Priority:  p.Priority,
		Projects:  p.Projects,
		Contexts:  p.Contexts,
		Tags:      p.Tags,
		DueAt:     p.Due,
		Threshold: p.Thresh,
		Source:    line,
	}
	return a.Store.CreateTodo(a.ctx, t)
}

const demoDataTag = "planck-demo"

func (a *App) HasDemoData() (bool, error) {
	if a.Store == nil {
		return false, nil
	}
	req := store.SearchRequest{
		Tags:     []string{demoDataTag},
		PageSize: 1,
	}
	results, err := a.Store.Search(a.ctx, req)
	if err != nil {
		return false, err
	}
	return len(results) > 0, nil
}

type todoWithMeta struct {
	line    string
	section string
	notes   string
}

func (a *App) SeedDemoData() error {
	if a.Store == nil {
		return fmt.Errorf("database not initialized")
	}

	// Tutorial todos organized by section with markdown notes
	demos := []todoWithMeta{
		// NOW section - Top priorities to start
		{
			line:    "(A) Welcome to Planck! Click me to see notes due:2025-11-01 +tutorial @getting-started #now",
			section: "now",
			notes:   "# Welcome!\n\nPlanck is a todo.txt task manager.\n\n**Quick Start:**\n- Click any todo to view/edit notes\n- Press ⌘N to add todos\n- Right-click for actions\n- Search with +project @context #tag",
		},
		{
			line:    "(A) Learn todo.txt syntax - (A)=high +project @context #tag due:2025-10-25 +tutorial @getting-started #now",
			section: "now",
			notes:   "# Todo.txt Format\n\nFormat: `(A) Title +project @context #tag due:2025-12-31`\n\n**Priority:** (A), (B), (C)\n**+project** - Group related tasks\n**@context** - Location/tool (@home, @computer)\n**#tag** - Flexible labels\n**due:YYYY-MM-DD** - Deadlines",
		},
		{
			line:    "(A) Try search: type keywords, +project, @context, #tag, pri:A due:2025-10-26 +tutorial @getting-started #now",
			section: "now",
			notes:   "# Search Power\n\n**Examples:**\n- `welcome` - keyword search\n- `+tutorial` - project filter\n- `#now` - tag filter\n- `pri:A` - priority\n- `+work -meeting` - exclude with minus\n\nTry searching for `#now` right now!",
		},

		// SOON section
		{
			line:    "(B) Explore Now/Soon/Anytime scope views due:2025-10-28 +tutorial @features #soon",
			section: "soon",
			notes:   "# Scope Views\n\n**Now** - Current focus (3-5 items)\n**Soon** - Upcoming tasks\n**Anytime** - Backlog\n\nRight-click any todo → Move to... → Pick section",
		},
		{
			line:    "(B) Set Session Context (🎯 icon) for focused work due:2025-10-30 +tutorial @features #soon",
			section: "soon",
			notes:   "# Session Context\n\nTemporary filter for deep work!\n\n1. Click 🎯 icon\n2. Select filters (e.g., +work @computer)\n3. Check \"Use as Filter Tab\"\n4. Work on just those tasks\n\nGreat for: focus blocks, errands, end-of-day priority items",
		},
		{
			line:    "(B) Create custom tabs in Settings for saved views due:2025-11-05 +tutorial @features #soon",
			section: "soon",
			notes:   "# Tabs\n\nSave common searches as tabs!\n\n**Examples:**\n- Work: `+work`\n- Today: `due<=today`\n- Urgent: `#urgent`\n\nSettings ⚙️ → Tabs section → Add Tab",
		},
		{
			line:    "(B) Use due:YYYY-MM-DD and t:YYYY-MM-DD (threshold) dates due:2025-11-08 +tutorial @time-management #soon",
			section: "soon",
			notes:   "# Dates\n\n**due:** - Deadline\n**t:** - Hide until (threshold)\n\nExample: `Buy gifts due:2025-12-20 t:2025-12-01` hides until December\n\nSwitch between Scope and Date views with buttons",
		},

		// ANYTIME section
		{
			line:    "(C) Sync with iCloud Drive across devices due:2025-12-01 +tutorial @sync #icloud #anytime",
			section: "anytime",
			notes:   "# iCloud Sync\n\n1. Create folder: `~/Library/Mobile Documents/com~apple~CloudDocs/Planck`\n2. Settings → Database Location → Change\n3. Paste path, restart\n4. On other devices: Use Existing Database\n\nWorks with Dropbox/Google Drive too!",
		},
		{
			line:    "Keyboard shortcuts: ⌘N=new, Esc=clear +tutorial @shortcuts #anytime",
			section: "anytime",
			notes:   "# Shortcuts\n\n- **⌘N** - Quick add\n- **Esc** - Clear/close\n- **Enter** - Search\n\nMore coming soon!",
		},
		{
			line:    "Customize themes in Settings ⚙️ +tutorial @settings #anytime",
			section: "anytime",
			notes:   "# Themes\n\n20+ themes available!\n\n- Catppuccin (4 variants)\n- Gruvbox\n- Tokyo Night\n- Nord\n- Dracula\n- Solarized\n\nSettings → Theme section",
		},
		{
			line:    "Right-click for Complete, Edit, Notes, Move, Archive, Delete +tutorial @features #anytime",
			section: "anytime",
			notes:   "# Right-Click Menu\n\n- Complete/Uncomplete\n- Edit todo\n- Add markdown notes\n- Move to Now/Soon/Anytime\n- Archive (hide completed)\n- Delete\n\nTry it on this todo!",
		},
		{
			line:    "Export/import todo.txt in Settings due:2025-12-15 +tutorial @backup #anytime",
			section: "anytime",
			notes:   "# Import & Export\n\nSettings → Import/Export section\n\n**Export** - Backup as todo.txt\n**Import** - Migrate from other apps\n\nStandard todo.txt format works everywhere!",
		},
		{
			line:    "Priority tips: (A)=critical (B)=important (C)=nice-to-have +tutorial @organization #anytime",
			section: "anytime",
			notes:   "# Priorities\n\n**(A)** - Must do\n**(B)** - Should do (use most)\n**(C)** - Could do\n**None** - Someday/maybe\n\nDon't overuse (A)!",
		},
		{
			line:    "Negative filters with minus: +work -meeting excludes meetings +tutorial @advanced #anytime",
			section: "anytime",
			notes:   "# Exclude with Minus\n\n`+work -meeting` - work, no meetings\n`pri:A -#waiting` - priority A, not waiting\n`@computer -+personal` - computer work tasks\n\nCombine any filters!",
		},
		{
			line:    "Add markdown notes to any todo (like this one!) +tutorial @features #anytime",
			section: "anytime",
			notes:   "# Markdown Support\n\n**Formatting:**\n- Headers: # ## ###\n- **Bold**, *italic*\n- Lists: - item\n- Links: [text](url)\n- Code: `inline` or blocks\n\nClick any todo to add notes!",
		},
		{
			line:    "x Completed format: checkmark or 'x' prefix in todo.txt +tutorial @getting-started #anytime",
			section: "anytime",
			notes:   "# Completed Todos\n\nThis one is done!\n\nFormat: `x 2025-10-23 Task name`\n\nToggle with checkbox or Settings → Show Completed",
		},
		{
			line:    "Try the Date view (calendar icon) vs Scope view due:2025-11-10 +tutorial @features #anytime",
			section: "anytime",
			notes:   "# View Modes\n\n**Scope** - Groups by Now/Soon/Anytime\n**Date** - Groups by due dates\n\nToggle with buttons near tabs",
		},
		{
			line:    "Create your first real todo with ⌘N now! due:2025-10-24 +tutorial @shortcuts #anytime",
			section: "anytime",
			notes:   "# Practice Time\n\nPress ⌘N and try:\n\n`(A) Call dentist @phone due:2025-10-25`\n`Buy groceries @errands #personal`\n`(B) Review report +work @computer`\n\nUse what you learned!",
		},
		{
			line:    "Remove all tutorial data with header button when ready +tutorial @getting-started #anytime",
			section: "anytime",
			notes:   "# Clean Up\n\nWhen comfortable:\n\n1. Click \"Remove Tutorial\" button in header\n2. All tutorial todos deleted\n3. Your todos stay!\n4. Start fresh\n\nGood luck! 🚀",
		},
	}

	for _, demo := range demos {
		p := parse.ParseLine(demo.line)
		todo := &store.Todo{
			Title:     p.Title,
			Priority:  p.Priority,
			Projects:  p.Projects,
			Contexts:  p.Contexts,
			Tags:      append(p.Tags, demoDataTag),
			DueAt:     p.Due,
			Threshold: p.Thresh,
			Source:    demo.line,
			NotesMD:   demo.notes,
			Section:   demo.section,
		}
		_, err := a.Store.CreateTodo(a.ctx, todo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) RemoveDemoData() error {
	if a.Store == nil {
		return fmt.Errorf("database not initialized")
	}

	req := store.SearchRequest{
		Tags:     []string{demoDataTag},
		PageSize: 500,
	}
	results, err := a.Store.Search(a.ctx, req)
	if err != nil {
		return err
	}

	for _, todo := range results {
		err = a.Store.DeleteTodo(a.ctx, int64(todo.ID))
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) UpdateTodo(t store.Todo) error {
	return a.Store.UpdateTodo(a.ctx, &t)
}

func (a *App) ToggleComplete(id int64, completed bool) error {
	return a.Store.ToggleComplete(a.ctx, id, completed)
}

func (a *App) Archive(id int64, archived bool) error {
	return a.Store.Archive(a.ctx, id, archived)
}

func (a *App) DeleteTodo(id int64) error {
	return a.Store.DeleteTodo(a.ctx, id)
}

func (a *App) Search(req store.SearchRequest) ([]store.Todo, error) {
	req.Query = strings.TrimSpace(req.Query)
	println("Backend search query:", req.Query, "statuses:", len(req.Statuses))
	results, err := a.Store.Search(a.ctx, req)
	println("Backend search returned:", len(results), "results, error:", err)
	return results, err
}

type FiltersResult struct {
	Projects []string `json:"projects"`
	Contexts []string `json:"contexts"`
	Tags     []string `json:"tags"`
}

func (a *App) GetFilters() (*FiltersResult, error) {
	projects, contexts, tags, err := a.Store.GetFilterValues(a.ctx)
	if err != nil {
		return nil, err
	}

	// Ensure we return empty arrays instead of nil
	if projects == nil {
		projects = []string{}
	}
	if contexts == nil {
		contexts = []string{}
	}
	if tags == nil {
		tags = []string{}
	}

	return &FiltersResult{
		Projects: projects,
		Contexts: contexts,
		Tags:     tags,
	}, nil
}

func (a *App) ExportTodoTxt() (string, error) {
	req := store.SearchRequest{
		Query:    "",
		Page:     0,
		PageSize: 10000,
		SortBy:   "created_at",
		SortDir:  "asc",
	}
	todos, err := a.Store.Search(a.ctx, req)
	if err != nil {
		return "", err
	}

	var lines []string
	for _, t := range todos {
		lines = append(lines, t.Source)
	}
	return strings.Join(lines, "\n"), nil
}

func (a *App) ImportTodoTxt(content string) error {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		_, err := a.CreateTodoFromLine(line)
		if err != nil {
			return err
		}
	}
	return nil
}

// Restart restarts the application
func (a *App) Restart() error {
	// Get the executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Start a new instance
	cmd := exec.Command(executable, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart: %w", err)
	}

	// Exit current process
	os.Exit(0)
	return nil
}
