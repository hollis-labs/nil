# Theme Picker - Enhanced Color Customization

## New Features

### Interactive Color Picker Modal
- **Click any color field** to expand Tailwind color palette
- **240 preset colors** from Tailwind CSS (all shades)
- **Custom hex input** - type any color like `#ffd700`
- **Live color picker** - native color selector for each field
- **Hover preview** - colors enlarge on hover for better visibility

### Tailwind Color Palette
The picker includes all Tailwind CSS colors:
- Slate, Gray, Zinc (neutrals)
- Red, Orange, Amber, Yellow (warm)
- Lime, Green, Emerald, Teal (green spectrum)
- Cyan, Sky, Blue, Indigo (blues)
- Violet, Purple, Fuchsia, Pink, Rose (purples/pinks)

Each color family has 10 shades from lightest to darkest.

### Customizable Elements
- **Background** - Main app background
- **Panel** - Terminal card background
- **Text** - Primary text color
- **Dim Text** - Secondary/muted text
- **Accent** - Links and highlights
- **Success** - Checkmarks, success badges
- **Warning** - High priority badge
- **Info** - Information badges
- **Border** - Card and input borders

### How to Use

1. **Open Theme Settings**:
   - Click "Theme" button in top bar
   - Modal appears with all color options

2. **Choose Colors**:
   - **Quick Select**: Click a color field → Choose from Tailwind palette
   - **Color Picker**: Click color square → Use native picker
   - **Custom Hex**: Type directly like `#ffd700` or `#1a2b3c`

3. **Preview & Apply**:
   - Changes are shown in real-time
   - Click "Apply Theme" to save
   - Click "Cancel" to discard changes
   - "Reset to Current" reverts to last saved theme

### Examples

**Gold Accent Theme**:
- Accent: `#ffd700` (gold)
- Warning: `#ff6b35` (orange-red)
- Success: `#4ade80` (green)

**Ocean Theme**:
- Background: `#0c4a6e` (dark blue)
- Panel: `#075985` (blue)
- Accent: `#38bdf8` (sky blue)
- Success: `#2dd4bf` (teal)

**Cyberpunk Theme**:
- Background: `#18181b` (zinc-900)
- Accent: `#d946ef` (fuchsia)
- Warning: `#facc15` (yellow)
- Success: `#34d399` (emerald)

### Technical Details

**Color Picker Features**:
- Grid layout: 10 colors per row
- Smooth hover scaling effect
- Click outside modal to close
- Persistent storage via localStorage
- Real-time CSS variable updates

**Modal Design**:
- Terminal-themed overlay
- Scrollable for small screens
- Click-outside-to-close
- Stop propagation on modal content
- Max height: 90vh for mobile support

### UI Improvements
- Monospace font for hex inputs
- Native color picker integration
- Organized color sections
- Helper text for guidance
- Action buttons: Reset, Cancel, Apply

The theme is saved to `localStorage` and persists across sessions!
