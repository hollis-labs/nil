package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hollis-labs/nil/capture/broker"

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
	draft, err := a.captureStore.Create(a.ctx)
	if err != nil {
		return draft, err
	}
	if a.captureDelivery != nil {
		started, _ := a.captureDelivery.BeginContext(a.ctx, draft.CaptureID)
		if started {
			a.captureWorkers.Add(1)
			go func() {
				defer a.captureWorkers.Done()
				ctx, cancel := context.WithTimeout(a.ctx, 25*time.Second)
				defer cancel()
				_ = a.captureDelivery.SampleContext(ctx, draft.CaptureID)
			}()
		}
	}
	return draft, nil
}

func (a *App) GetCapturePresets() (capture.PresetSelection, error) {
	if err := a.captureReady(); err != nil {
		return capture.PresetSelection{}, err
	}
	return a.captureStore.Presets(a.ctx)
}

func (a *App) SaveCapturePreset(id string, revision int, name, quickAdd string) (capture.Preset, error) {
	if err := a.captureReady(); err != nil {
		return capture.Preset{}, err
	}
	return a.captureStore.SavePreset(a.ctx, id, revision, name, quickAdd)
}

func (a *App) SelectCapturePreset(id string) error {
	if err := a.captureReady(); err != nil {
		return err
	}
	return a.captureStore.SelectPreset(a.ctx, id)
}

func (a *App) QueueCaptureDraft(id string, revision int) (capture.Envelope, error) {
	if err := a.captureReady(); err != nil {
		return capture.Envelope{}, err
	}
	return a.captureStore.Queue(a.ctx, id, revision)
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

func (a *App) GetCaptureAccounts() (capture.AccountSelection, error) {
	if err := a.captureReady(); err != nil {
		return capture.AccountSelection{}, err
	}
	return a.captureStore.Accounts(a.ctx)
}
func (a *App) PairCaptureBroker(origin, code string) (capture.Account, error) {
	if err := a.deliveryReady(); err != nil {
		return capture.Account{}, err
	}
	return a.captureDelivery.Pair(a.ctx, origin, code)
}
func (a *App) SelectCaptureAccount(id string) error {
	if err := a.captureReady(); err != nil {
		return err
	}
	return a.captureStore.SelectAccount(a.ctx, id)
}
func (a *App) RenewCaptureAccount() (capture.Account, error) {
	if err := a.deliveryReady(); err != nil {
		return capture.Account{}, err
	}
	return a.captureDelivery.Renew(a.ctx)
}
func (a *App) GetCaptureEvidence() (capture.SetupEvidence, error) {
	if err := a.deliveryReady(); err != nil {
		return capture.SetupEvidence{}, err
	}
	return a.captureDelivery.Evidence(a.ctx)
}
func (a *App) ListCaptureDeliveries() ([]capture.DeliveryView, error) {
	if err := a.captureReady(); err != nil {
		return nil, err
	}
	return a.captureStore.DeliveryViews(a.ctx)
}
func (a *App) RunCaptureDelivery(id, action string, acknowledgeMayHaveSent bool) (broker.Status, error) {
	if err := a.deliveryReady(); err != nil {
		return broker.Status{}, err
	}
	return a.captureDelivery.Run(a.ctx, id, action, acknowledgeMayHaveSent)
}
func (a *App) ListRemoteCaptureStatuses(before int64) (broker.StatusPage, error) {
	if err := a.deliveryReady(); err != nil {
		return broker.StatusPage{}, err
	}
	return a.captureDelivery.RemotePage(a.ctx, before)
}
func (a *App) deliveryReady() error {
	if err := a.captureReady(); err != nil {
		return err
	}
	if a.captureDelivery == nil {
		return errors.New("capture delivery unavailable")
	}
	return nil
}
