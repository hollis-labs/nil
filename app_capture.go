package main

import (
	"errors"

	"github.com/hollis-labs/nil/capture"
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
