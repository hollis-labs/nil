package main

import (
	"fmt"
	"strings"

	"github.com/hollis-labs/nil/parse"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
)

// --- Helper: find which store holds an item by ID ---

// storeForItemID tries the active vault then the inbox store to locate an item.
func (a *App) storeForItemID(id int64) (*store.Store, error) {
	if s := a.vaultMgr.ActiveStore(); s != nil {
		if _, err := s.GetItem(a.ctx, id); err == nil {
			return s, nil
		}
	}
	if s := a.vaultMgr.InboxStore(); s != nil {
		if _, err := s.GetItem(a.ctx, id); err == nil {
			return s, nil
		}
	}
	return nil, fmt.Errorf("item %d not found", id)
}

// --- Item CRUD ---

func (a *App) CreateItemFromLine(line string) (*store.Item, error) {
	return a.createFromLine(line, "todo")
}

func (a *App) CreateNoteFromLine(line string) (*store.Item, error) {
	return a.createFromLine(line, "note")
}

// CreateScratchFromLine creates a scratch-kind item. Scratch items behave
// like notes until the scratch UX follow-up defines dedicated behavior.
func (a *App) CreateScratchFromLine(line string) (*store.Item, error) {
	return a.createFromLine(line, "scratch")
}

// createFromLine is the shared GUI create path. Parses the todo.txt-style
// line and routes blank-title items to the shared inbox store, matching the
// "capture without friction" UX from F1.
func (a *App) createFromLine(line, kind string) (*store.Item, error) {
	p := parse.ParseLine(line)
	input := items.CreateInput{
		Title:      p.Title,
		Kind:       kind,
		Priority:   p.Priority,
		DueAt:      p.Due,
		Threshold:  p.Thresh,
		Projects:   p.Projects,
		Contexts:   p.Contexts,
		Tags:       p.Tags,
		SourceLine: line,
	}
	// Only the todo kind carries scheduling fields; clear them for note/scratch
	// to preserve the previous per-kind behavior.
	if kind != "todo" {
		input.Priority = nil
		input.DueAt = nil
		input.Threshold = nil
	}
	if strings.TrimSpace(input.Title) == "" {
		input.Inbox = true
		return svc.Create(a.ctx, a.vaultMgr.InboxStore(), input)
	}
	return svc.Create(a.ctx, a.vaultMgr.ActiveStore(), input)
}

func (a *App) GetItem(id int64) (*store.Item, error) {
	// Try active store first, then inbox
	if s := a.vaultMgr.ActiveStore(); s != nil {
		if item, err := s.GetItem(a.ctx, id); err == nil {
			return item, nil
		}
	}
	if s := a.vaultMgr.InboxStore(); s != nil {
		if item, err := s.GetItem(a.ctx, id); err == nil {
			return item, nil
		}
	}
	return nil, fmt.Errorf("item %d not found", id)
}

func (a *App) GetBackrefs(id int64) ([]store.Item, error) {
	return a.vaultMgr.ActiveStore().GetBackrefs(a.ctx, id)
}

// UpdateItem routes the update to the correct store based on the item's inbox flag.
// If an item with inbox=true exists in the active vault, it is moved to the inbox store.
func (a *App) UpdateItem(t store.Item) error {
	if t.Inbox {
		// Check if item currently lives in the active vault (explicit "→ Inbox" routing)
		if s := a.vaultMgr.ActiveStore(); s != nil {
			if _, err := s.GetItem(a.ctx, t.ID); err == nil {
				// Move: create copy in inbox store, delete from active vault
				newItem := t
				newItem.ID = 0
				if _, err := a.vaultMgr.InboxStore().CreateItem(a.ctx, &newItem); err != nil {
					return fmt.Errorf("creating item in inbox: %w", err)
				}
				return s.DeleteItem(a.ctx, t.ID)
			}
		}
		// Item is already in inbox store — update in place
		return svc.Update(a.ctx, a.vaultMgr.InboxStore(), &t)
	}
	return svc.Update(a.ctx, a.vaultMgr.ActiveStore(), &t)
}

func (a *App) ToggleComplete(id int64, completed bool) error {
	s, err := a.storeForItemID(id)
	if err != nil {
		return err
	}
	return s.ToggleComplete(a.ctx, id, completed)
}

func (a *App) Archive(id int64, archived bool) error {
	s, err := a.storeForItemID(id)
	if err != nil {
		return err
	}
	return s.Archive(a.ctx, id, archived)
}

func (a *App) DeleteItem(id int64) error {
	s, err := a.storeForItemID(id)
	if err != nil {
		return err
	}
	return s.DeleteItem(a.ctx, id)
}

func (a *App) Search(req store.SearchRequest) ([]store.Item, error) {
	req.Query = strings.TrimSpace(req.Query)
	println("Backend search query:", req.Query, "statuses:", len(req.Statuses))
	results, err := svc.Search(a.ctx, a.vaultMgr.ActiveStore(), req)
	println("Backend search returned:", len(results), "results, error:", err)
	return results, err
}

type FiltersResult struct {
	Projects []string `json:"projects"`
	Contexts []string `json:"contexts"`
	Tags     []string `json:"tags"`
}

func (a *App) GetFilters() (*FiltersResult, error) {
	projects, contexts, tags, err := a.vaultMgr.ActiveStore().GetFilterValues(a.ctx)
	if err != nil {
		return nil, err
	}
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
