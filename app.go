package main

import (
	"context"
	"fmt"
	"os"
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

func (a *App) SeedDemoData() error {
	if a.Store == nil {
		return fmt.Errorf("database not initialized")
	}

	demoTodos := []string{
		"(A) Welcome to Planck - your quantum todo manager! +tutorial @getting-started",
		"(A) Try searching with keywords, like 'welcome' or 'syntax' +tutorial @getting-started",
		"(B) Use +project tags to organize by project +tutorial @organization",
		"(B) Use @context tags for location or tool (like @home @computer @phone) +tutorial @organization",
		"(B) Use #hashtags for flexible categorization #learning #productivity +tutorial",
		"(C) Set priorities with (A) (B) (C) at the start of your todo +tutorial @organization",
		"Add due dates with due:2025-12-31 for deadlines +tutorial @time-management",
		"Add threshold dates with t:2025-11-01 to hide tasks until ready +tutorial @time-management",
		"Combine filters: search for '+work @office pri:A' to find high-priority office work +tutorial @advanced",
		"Use negative filters: search '-meeting' to exclude todos with 'meeting' +tutorial @advanced",
		"(A) Click the target icon to set Session Context - temporary filters for focus mode +tutorial @features",
		"Use tabs (configure in Settings) to create saved filter views +tutorial @features",
		"Switch between Scope view (Now/Soon/Anytime) and Date view (calendar) +tutorial @features",
		"Press ⌘N to quickly add new todos from anywhere in the app +tutorial @shortcuts",
		"Click the ⚙️ Settings button to customize themes, tabs, and preferences +tutorial @features",
		"Right-click todos for quick actions: archive, delete, move sections +tutorial @features",
		"(A) Store your database in iCloud Drive to sync across devices! +tutorial @sync #icloud",
		"Export to todo.txt format via Settings → Import/Export +tutorial @backup",
		"x Completed todos look like this - they're marked with 'x' in todo.txt format +tutorial @getting-started",
		"This is a note-taking example - click any todo to add detailed markdown notes +tutorial @features",
	}

	for _, line := range demoTodos {
		p := parse.ParseLine(line)
		todo := &store.Todo{
			Title:     p.Title,
			Priority:  p.Priority,
			Projects:  p.Projects,
			Contexts:  p.Contexts,
			Tags:      append(p.Tags, demoDataTag),
			DueAt:     p.Due,
			Threshold: p.Thresh,
			Source:    line,
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
