package main

import (
	"errors"
	"strings"

	"github.com/hollis-labs/nil/capture"
	"github.com/hollis-labs/nil/store"
)

func (a *App) captureReady() error {
	if a.captureErr != nil {
		return a.captureErr
	}
	if a.captureStore == nil {
		return errors.New("capture storage unavailable")
	}
	return nil
}

func (a *App) SaveCaptureContent(id string, revision int, markdown, quickAdd string) (capture.Draft, error) {
	if err := a.captureReady(); err != nil {
		return capture.Draft{}, err
	}
	return a.captureStore.SaveContent(a.ctx, id, revision, markdown, quickAdd)
}

// SearchCaptureReferences is a bounded local-vault adapter, never a URL fetch.
func (a *App) SearchCaptureReferences(query string) ([]capture.Suggestion, error) {
	out := []capture.Suggestion{}
	if a.vaultMgr == nil {
		return out, nil
	}
	v := a.vaultMgr.GetActiveVault()
	if v == nil {
		return out, nil
	}
	st, err := a.vaultMgr.StoreForID(v.ID)
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if len(query) > 200 {
		return out, nil
	}
	items, err := svc.Search(a.ctx, st, store.SearchRequest{Query: query, Kind: "all", PageSize: 8})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		out = append(out, capture.ItemSuggestion(v.ID, item.ID, item.UpdatedAt, item.Title))
	}
	return out, nil
}

func (a *App) CreateCaptureDraft() (capture.Draft, error) {
	if err := a.captureReady(); err != nil {
		return capture.Draft{}, err
	}
	return a.captureStore.Create(a.ctx)
}

func (a *App) ListCaptureDrafts() ([]capture.Draft, error) {
	if err := a.captureReady(); err != nil {
		return nil, err
	}
	return a.captureStore.List(a.ctx)
}

func (a *App) SaveCaptureDraft(id string, revision int, markdown string) (capture.Draft, error) {
	if err := a.captureReady(); err != nil {
		return capture.Draft{}, err
	}
	return a.captureStore.Save(a.ctx, id, revision, markdown)
}
