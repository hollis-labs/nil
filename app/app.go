package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"todo/internal/config"
	"todo/internal/parse"
	"todo/internal/store"
)

type App struct {
	ctx        context.Context
	Store      *store.Store
	needsSetup bool
}

func (a *App) Startup(ctx context.Context) {
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
	// Pass-through: the frontend can pre-parse; for MVP we accept raw query too
	// Example: allow plain words -> FTS query string
	req.Query = strings.TrimSpace(req.Query)
	return a.Store.Search(a.ctx, req)
}

func (a *App) GetFilters() (projects, contexts, tags []string, err error) {
	return a.Store.GetFilterValues(a.ctx)
}
