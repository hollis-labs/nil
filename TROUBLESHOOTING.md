# Troubleshooting Guide

## Scrollbar Colors Not Showing in Custom Theme Editor

### Quick Fix
The theme system has been updated with automatic migration. Simply:
1. Reload the app (Cmd+R or F5)
2. The theme will automatically update with new scrollbar properties
3. Open Settings → Theme → Custom
4. You should see 12 color inputs total (9 base + 3 scrollbar)

### If That Doesn't Work

**Check Browser Console:**
1. Open DevTools (F12 or Cmd+Option+I)
2. Go to Console tab  
3. Look for messages like:
   - `[Theme] Loaded and migrated theme with properties: ...`
   - `[Settings] Full theme properties: ...`
   - `[Settings] Scrollbar properties: { scrollbarBg: '...', ... }`

**Expected Output:**
```
[Theme] Loaded and migrated theme with properties: (12) ['bg', 'panel', 'fg', 'dim', 'accent', 'success', 'warn', 'info', 'border', 'scrollbarBg', 'scrollbarThumb', 'scrollbarThumbHover']
```

**If scrollbar properties are missing:**
```bash
# In browser console:
localStorage.removeItem('todo.term.theme')
# Then reload the app
```

## Gradient Colors Not Showing on Tags/Projects

### Verify You Have Multiple Tags/Projects

Create a test todo:
```
Test todo +proj1 +proj2 +proj3 #tag1 #tag2 #tag3
```

### Expected Result:
- **Tags (#tag1, #tag2, #tag3):** 
  - #tag1: Bright gray (#b4b8c0)
  - #tag2: Medium gray (#999da5)  
  - #tag3: Dark gray (#7e8289)
  - #tag4+: Same as #tag3

- **Projects (+proj1, +proj2, +proj3):**
  - +proj1: Bright version of theme accent
  - +proj2: Normal theme accent
  - +proj3: Dark version of theme accent  
  - +proj4+: Same as +proj3

### Test Different Themes:
1. Settings → Theme → Synthwave
2. Save Theme
3. Create todo with multiple projects
4. Projects should now show pink gradient (synthwave accent)

## Debug Checklist

✅ App reloaded after update  
✅ Browser console shows theme migration logs  
✅ Settings → Theme → Custom shows 12 color inputs  
✅ "Scrollbar" section appears below "Base Colors"  
✅ Multiple tags show different gray shades  
✅ Multiple projects show gradient based on theme accent  

## Still Having Issues?

1. **Clear all app data:**
   ```javascript
   // In browser console:
   localStorage.clear()
   ```

2. **Hard reload:**
   - Chrome/Edge: Cmd+Shift+R (Mac) or Ctrl+Shift+R (Windows)
   - Safari: Cmd+Option+R

3. **Check theme file directly:**
   - File: `frontend/src/theme/theme.ts`
   - Should have `scrollbarBg`, `scrollbarThumb`, `scrollbarThumbHover` in all 4 presets

4. **Verify component code:**
   - File: `frontend/src/components/TerminalList.tsx`
   - Line ~197: Should use `getProjectColor(i, theme.accent)`
   - Line ~198: Should use `getTagColor(i)`

## Technical Details

**Theme Migration:**
- Old themes in localStorage are automatically merged with new default theme
- This ensures new properties (scrollbar colors) are always present
- Migration happens on app load in `ThemeProvider.tsx`

**Gradient Functions:**
- `getTagColor(index)`: Returns fixed gray gradient
- `getProjectColor(index, accentColor)`: Calculates gradient from theme accent
- Both use `Math.min(index, 2)` to cap at 3 colors

**Console Logging:**
- Theme loading: `[Theme] ...`
- Settings modal: `[Settings] ...`
- Look for these to verify theme has all properties
