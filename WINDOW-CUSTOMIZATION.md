# Window Customization Guide

## Current Configuration (Chromeless/Frameless)

Planck uses a **frameless window** with a custom drag area, similar to Raycast, Spotlight, etc.

### Settings in `main.go`:

```go
Frameless: true,  // Removes default window chrome
Width: 900,
Height: 700,
MinWidth: 800,
MinHeight: 600,
```

### macOS-Specific Settings:

```go
Mac: &mac.Options{
    TitleBar: &mac.TitleBar{
        TitlebarAppearsTransparent: true,  // Transparent titlebar
        HideTitle: true,                    // No title text
        HideTitleBar: false,                // Keep titlebar for traffic lights
        FullSizeContent: true,              // Content extends under titlebar
        UseToolbar: false,                  // No toolbar
        HideToolbarSeparator: true,         // No separator line
    },
}
```

## Dragging the Window

The top 40px of the app is a **drag region** (see `App.tsx`):

```tsx
<div style={{ /* ... */ }} data-wails-drag />
```

Users can click and drag this area to move the window.

## Alternative Configurations

### 1. **Full Chrome (Standard macOS window)**
```go
Frameless: false,
// Remove Mac.TitleBar customizations
```

### 2. **Transparent Titlebar (keeps traffic lights visible)**
```go
Frameless: false,
Mac: &mac.Options{
    TitleBar: &mac.TitleBar{
        TitlebarAppearsTransparent: true,
        HideTitle: true,
        FullSizeContent: true,
    },
}
```

### 3. **Borderless Utility Window**
```go
Frameless: true,
Mac: &mac.Options{
    TitleBar: &mac.TitleBar{
        HideTitleBar: true,  // Completely hide titlebar
    },
}
```

### 4. **Fixed Size (non-resizable)**
```go
Width: 900,
Height: 700,
DisableResize: true,
```

### 5. **Always on Top**
```go
AlwaysOnTop: true,
```

## Other Useful Options

```go
// Window behavior
StartHidden: false,
HideWindowOnClose: false,
Fullscreen: false,

// Appearance
BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},

// Dev tools (production should be false)
Debug: options.Debug{
    OpenInspectorOnStartup: false,
},
```

## Cross-Platform Considerations

- `Mac.Options` only affects macOS
- Windows has `windows.Options` for platform-specific features
- Linux has `linux.Options`

Example:
```go
import (
    "github.com/wailsapp/wails/v2/pkg/options/windows"
)

Windows: &windows.Options{
    WebviewIsTransparent: true,
    WindowIsTranslucent: true,
    DisableWindowIcon: false,
},
```

## Testing Window Behavior

After changing `main.go`:
```bash
wails build
open build/bin/todo-app.app
```

Or in dev mode:
```bash
wails dev
```

## Traffic Lights (Close/Minimize/Maximize buttons)

Current config shows them in the top-left. To hide completely:
```go
Mac: &mac.Options{
    TitleBar: &mac.TitleBar{
        HideTitleBar: true,  // Removes traffic lights entirely
    },
}
```

Note: If you hide them, users can't close/minimize via UI. Consider adding custom buttons.

## Documentation

Full Wails options: https://wails.io/docs/reference/options
