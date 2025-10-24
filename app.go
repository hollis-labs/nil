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
