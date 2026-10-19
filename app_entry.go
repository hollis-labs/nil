package main

import (
	"runtime"

	"github.com/hollis-labs/nil/capture/entry"
	"github.com/wailsapp/wails/v2/pkg/menu"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) captureMenu() *menu.Menu {
	result := menu.NewMenu()
	if runtime.GOOS == "darwin" {
		result.Append(menu.AppMenu())
		result.Append(menu.EditMenu())
	}
	captureMenu := result.AddSubmenu("Capture")
	// No local accelerator: Command-I stays entirely with Frag until explicitly
	// enabled in Nil. Menu and status-item invocation work while it is disabled.
	captureMenu.AddText("Open Capture", nil, func(*menu.CallbackData) { a.captureEntry.Request() })
	return result
}
func (a *App) raiseCapture() {
	if a.ctx == nil {
		return
	}
	wailsruntime.Show(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.WindowShow(a.ctx) // Wails macOS Show makes key + activates NSApp.
	wailsruntime.EventsEmit(a.ctx, "nil:capture-invoke")
}
func (a *App) CaptureEntryReady()                  { a.captureEntry.Ready() }
func (a *App) GetCaptureEntryStatus() entry.Status { return a.captureEntry.Status() }
func (a *App) SetCaptureShortcut(shortcut entry.Shortcut) (entry.Status, error) {
	return a.captureEntry.Apply(shortcut)
}
