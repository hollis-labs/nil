# Theme System Enhancements

## Overview
Enhanced the PLANCK theme system with scrollbar customization and improved theme preview UI.

## Changes Made

### 1. Scrollbar Theme Properties

**New Theme Variables:**
- `scrollbarBg` - Background color of scrollbar track
- `scrollbarThumb` - Color of scrollbar thumb (draggable part)
- `scrollbarThumbHover` - Hover state color for scrollbar thumb

**Added to All Theme Presets:**

**Default Theme:**
```typescript
scrollbarBg: '#0f131a',       // Matches panel
scrollbarThumb: '#1f2937',    // Matches border
scrollbarThumbHover: '#2d3748' // Slightly lighter
```

**Light Theme:**
```typescript
scrollbarBg: '#f1f5f9',
scrollbarThumb: '#cbd5e1',
scrollbarThumbHover: '#94a3b8'
```

**Synthwave Theme:**
```typescript
scrollbarBg: '#241b2f',       // Matches panel
scrollbarThumb: '#495495',    // Matches border
scrollbarThumbHover: '#ff7edb' // Matches accent (dramatic!)
```

**Terminal Theme:**
```typescript
scrollbarBg: '#0a0a0a',
scrollbarThumb: '#1a1a1a',
scrollbarThumbHover: '#00aa00' // Classic green
```

### 2. Custom Theme Editor Improvements

**Organized Sections:**
- **Base Colors** section (9 colors: bg, panel, fg, dim, accent, success, warn, info, border)
- **Scrollbar** section (3 colors: bg, thumb, thumb hover)

**Better Label Formatting:**
- Converts camelCase to Title Case
- `scrollbarBg` → "Bg"
- `scrollbarThumbHover` → "Thumb Hover"
- Clean, readable labels at 80px width

**Color Picker + Text Input:**
- Visual color picker (40px square)
- Text input with hex value (editable)
- Placeholder "#000000" for empty values
- Monospace font for hex codes

### 3. Theme Preview Enhancement

**Preset Cards Now Show:**
1. **Main color swatches** (9 squares)
   - All base theme colors displayed
   - 28px squares with rounded corners
   - Tooltips show property name + hex value
   
2. **Scrollbar preview row**
   - Separate row labeled "Scrollbar:"
   - 3 smaller squares (20px each)
   - Shows track → thumb → hover progression
   - Tooltips for each state

### 4. CSS Variable Integration

**Applied Throughout App:**
```css
::-webkit-scrollbar-track {
  background: var(--scrollbar-bg);
}
::-webkit-scrollbar-thumb {
  background: var(--scrollbar-thumb);
}
::-webkit-scrollbar-thumb:hover {
  background: var(--scrollbar-thumb-hover);
}
```

**Firefox Support:**
```css
* {
  scrollbar-width: thin;
  scrollbar-color: var(--scrollbar-thumb) var(--scrollbar-bg);
}
```

### 5. Fallback Handling

If scrollbar properties are missing from a theme:
```typescript
t.scrollbarBg || t.panel           // Default to panel color
t.scrollbarThumb || t.border       // Default to border color
t.scrollbarThumbHover || t.dim     // Default to dim color
```

## Visual Improvements

**Before:**
- Scrollbars used fixed accent color
- Custom theme: 9 color pickers in flat list
- Preview: Only first 5 colors shown

**After:**
- Scrollbars fully customizable per theme
- Custom theme: Organized into Base + Scrollbar sections (12 total)
- Preview: All 9 base colors + 3 scrollbar colors with labels
- Proper label formatting (Title Case, readable)

## User Experience

**Theme Designers Can Now:**
1. Match scrollbars to overall theme aesthetic
2. Create subtle or dramatic hover effects
3. See all colors at a glance in preview
4. Edit scrollbar colors independently from other UI elements

**Example Use Cases:**
- **Subtle (Default)**: Scrollbar barely visible, blends into panel
- **Accent (Synthwave)**: Hover state uses theme accent for pop
- **Classic (Terminal)**: Green hover state for retro terminal feel
- **Contrast (Light)**: Clear visual distinction for accessibility

## Testing

✅ All 4 preset themes include scrollbar properties
✅ Custom theme editor displays all 12 properties
✅ Labels properly formatted (camelCase → Title Case)
✅ Fallbacks prevent undefined CSS variables
✅ Works in Chrome, Safari (WebKit) and Firefox
✅ TypeScript compilation passes
✅ Theme switching updates scrollbars immediately

## Technical Notes

- Scrollbar properties are optional (`scrollbarBg?: string`) for backward compatibility
- `applyTheme()` function handles fallbacks automatically
- Custom theme state preserved when switching between presets
- Object.entries() iteration ensures all properties show in editor
