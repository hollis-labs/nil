# Immediate Fix - Restore Your Theme

## Your custom theme was overwritten with the Terminal theme

The console shows you're currently using terminal theme colors (all black/green).

## Quick Recovery Steps

### Step 1: Clear the Broken Theme
Open browser console (F12 or Cmd+Option+I) and run:
```javascript
localStorage.removeItem('todo.term.theme');
location.reload();
```

This will reload with the default dark theme.

### Step 2: Recreate Your Custom Theme

**Option A: Start from Default Theme**
1. The app should now show the default dark blue theme
2. Settings → Theme → Custom → Edit colors
3. Adjust the 12 colors to your preference
4. Save Theme

**Option B: Start from a Different Preset**
1. Settings → Theme → Select "Light" or "Synthwave"
2. Save Theme  
3. Click Custom → Adjust colors
4. Save Theme

**Option C: If You Remember Your Colors**
Run this in console (replace with your colors):
```javascript
const myCustomTheme = {
  bg: '#0b0e14',      // Your background
  panel: '#0f131a',   // Your panel
  fg: '#e6edf3',      // Your text
  dim: '#8b949e',     // Your dimmed text
  accent: '#7aa2f7',  // Your accent color
  success: '#22c55e', // Your success color
  warn: '#eab308',    // Your warning color
  info: '#60a5fa',    // Your info color
  border: '#1f2937',  // Your border color
  scrollbarBg: '#0f131a',
  scrollbarThumb: '#1f2937',
  scrollbarThumbHover: '#2d3748'
};

localStorage.setItem('todo.term.theme', JSON.stringify(myCustomTheme));
location.reload();
```

## About the Scrollbars

The scrollbars not changing is likely a **macOS/browser** issue, not the theme. The CSS is correct.

### Fix for macOS Scrollbars:

**System Settings:**
1. Open System Settings → Appearance
2. Find "Show scroll bars"
3. Change from "Automatically" to **"Always"**
4. Restart browser

**OR** test in a different browser to verify CSS is working.

### Verify Scrollbar CSS is Working:

Open DevTools → Elements → Select `<html>` → Check computed styles:
- Look for `--scrollbar-bg` should be `#0a0a0a` (or your theme color)
- Look for `--scrollbar-thumb` should be `#1a1a1a` (or your theme color)

If those CSS variables are set correctly but scrollbars still look default, it's a browser/OS issue.

## Why This Happened

The migration code I wrote had a bug - it overwrote your entire theme instead of just adding scrollbar properties. This has been fixed, but the damage was already done to your saved theme.

I sincerely apologize for this!
