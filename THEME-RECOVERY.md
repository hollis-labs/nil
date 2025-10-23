# Theme Recovery Guide

## If Your Custom Theme Got Overwritten

The theme migration had a bug that may have overwritten your custom theme. Here's how to recover:

### Quick Fix - Clear and Start Fresh

**Option 1: Use a Preset as Starting Point**
1. Open Settings → Theme
2. Select a preset that's close to what you want (Default, Light, Synthwave, or Terminal)
3. Click "Save Theme"
4. Click on "Custom"
5. Click "Edit Custom Theme" (if not already editing)
6. Adjust the 12 colors to your preference
7. Click "Save Theme"

**Option 2: Manually Set Custom Theme via Console**

If you remember your custom theme colors, you can restore them via browser console:

```javascript
// Example custom theme - replace with your colors
const myTheme = {
  bg: '#1a1b26',
  panel: '#24283b',
  fg: '#c0caf5',
  dim: '#565f89',
  accent: '#7aa2f7',
  success: '#9ece6a',
  warn: '#e0af68',
  info: '#7dcfff',
  border: '#414868',
  scrollbarBg: '#24283b',
  scrollbarThumb: '#414868',
  scrollbarThumbHover: '#545c7e'
};

// Save it
localStorage.setItem('todo.term.theme', JSON.stringify(myTheme));

// Reload the page
location.reload();
```

### Check What Theme is Currently Loaded

```javascript
// In browser console:
const current = JSON.parse(localStorage.getItem('todo.term.theme'));
console.log('Current theme:', current);
```

### Verify Scrollbar Properties are Present

```javascript
// In browser console:
const theme = JSON.parse(localStorage.getItem('todo.term.theme'));
console.log('Has scrollbar properties:', {
  scrollbarBg: theme.scrollbarBg,
  scrollbarThumb: theme.scrollbarThumb,
  scrollbarThumbHover: theme.scrollbarThumbHover
});
```

If any are `undefined`, the theme migration will add them automatically using fallbacks.

## What Went Wrong

The initial migration code did:
```typescript
// BAD - overwrites entire theme
const merged = { ...defaultTheme, ...parsed };
localStorage.setItem('todo.term.theme', JSON.stringify(merged));
```

This would overwrite any custom theme with default theme properties, then overlay the saved theme. The order was wrong!

## What's Fixed Now

New migration code only adds missing properties:
```typescript
// GOOD - only adds missing scrollbar properties
const merged = {
  ...parsed,
  scrollbarBg: parsed.scrollbarBg || defaultTheme.scrollbarBg,
  scrollbarThumb: parsed.scrollbarThumb || defaultTheme.scrollbarThumb,
  scrollbarThumbHover: parsed.scrollbarThumbHover || defaultTheme.scrollbarThumbHover,
};
// Does NOT auto-save, preserves user theme
```

## Prevention

From now on:
- Theme migration only adds missing properties
- No auto-save to localStorage during migration
- Your custom theme is preserved
- Only explicit "Save Theme" button writes to localStorage

## Apologies

This was a bug in the migration logic. Your theme should not have been overwritten. The fix is now in place to prevent this from happening again.
