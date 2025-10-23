# Clear Theme Cache

If you don't see the new scrollbar color inputs in the Custom Theme editor, you may have old theme data in localStorage.

## How to Clear

### Option 1: Browser DevTools
1. Open the app
2. Open DevTools (F12 or Cmd+Option+I)
3. Go to Console tab
4. Run: `localStorage.removeItem('todo.term.theme')`
5. Refresh the app (Cmd+R or F5)

### Option 2: Settings UI
1. Open Settings → Theme
2. Select any preset theme (Default, Light, Synthwave, or Terminal)
3. Click "Save Theme"
4. Now select "Custom" 
5. You should see all 12 color inputs including the 3 scrollbar colors

## What You Should See in Custom Theme

**Base Colors (9):**
- Bg
- Panel
- Fg
- Dim
- Accent
- Success
- Warn
- Info
- Border

**Scrollbar (3):**
- Bg
- Thumb
- Thumb Hover

## Verifying Gradient Colors

To see the gradient colors on tags and projects:

1. Create a todo with multiple tags: `Test todo #tag1 #tag2 #tag3`
2. Create a todo with multiple projects: `Test todo +project1 +project2 +project3`
3. You should see the colors fade from bright → medium → dark

**Tags:** Gray gradient (#b4b8c0 → #999da5 → #7e8289)
**Projects:** Theme accent gradient (bright → normal → dark)
