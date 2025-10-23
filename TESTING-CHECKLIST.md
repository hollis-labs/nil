# Testing Checklist: New Features

## Quick Start
Run the app and follow these test scenarios to verify all features work correctly.

---

## 1. Autocomplete Components Testing

### Test 1.1: Tags Autocomplete
- [ ] Open Quick Add modal (⌘N)
- [ ] Click in the Tags field
- [ ] Start typing a partial tag name
- [ ] Verify dropdown appears with filtered suggestions
- [ ] Use arrow keys to navigate suggestions
- [ ] Press Tab to select a suggestion
- [ ] Verify chip appears with # prefix
- [ ] Add multiple tags
- [ ] Click X on a chip to remove it
- [ ] Type in empty field and press Backspace
- [ ] Verify last tag is removed

### Test 1.2: Projects Autocomplete
- [ ] Click in the Projects field
- [ ] Type partial project name
- [ ] Verify dropdown shows filtered projects
- [ ] Press Enter to select
- [ ] Verify chip appears with + prefix
- [ ] Test keyboard navigation (arrows, Tab, Enter)
- [ ] Test click outside to close dropdown
- [ ] Test removing projects

### Test 1.3: Contexts Autocomplete
- [ ] Click in the Contexts field
- [ ] Type partial context name
- [ ] Verify dropdown shows filtered contexts
- [ ] Select multiple contexts
- [ ] Verify chips appear with @ prefix
- [ ] Test all keyboard shortcuts
- [ ] Test click-to-remove on chips

### Test 1.4: Autocomplete Edge Cases
- [ ] Type non-existent value
- [ ] Press Enter to add new value
- [ ] Verify it's added even if not in suggestions
- [ ] Type same value twice
- [ ] Verify duplicates are prevented
- [ ] Test with empty suggestions list
- [ ] Test with very long lists (10+ items)

---

## 2. Template System Testing

### Test 2.1: Creating Templates
- [ ] Open Quick Add modal
- [ ] Fill in: 2 projects, 2 contexts, 2 tags, priority B
- [ ] Click "Save Template" button
- [ ] Enter template name: "Work Tasks"
- [ ] Verify preview shows all values
- [ ] Click "Save Template"
- [ ] Verify dialog closes

### Test 2.2: Applying Templates
- [ ] Clear the form (click Clear button)
- [ ] Click "Templates" dropdown
- [ ] Verify "Work Tasks" appears
- [ ] Verify preview shows correct values
- [ ] Click on template to select
- [ ] Verify all values are populated
- [ ] Add one more tag manually
- [ ] Apply template again
- [ ] Verify new tag is preserved (merge, not replace)

### Test 2.3: Multiple Templates
- [ ] Create template "Personal" with different values
- [ ] Create template "Urgent" with priority A
- [ ] Click Templates dropdown
- [ ] Type in search box
- [ ] Verify templates are filtered by name
- [ ] Select different templates
- [ ] Verify values merge correctly

### Test 2.4: Template Edge Cases
- [ ] Try to save template with empty name
- [ ] Verify save button is disabled
- [ ] Save template with no values
- [ ] Apply it and verify nothing breaks
- [ ] Create 5+ templates
- [ ] Verify dropdown scrolls properly

---

## 3. Session Context Testing

### Test 3.1: Creating Session Profile
- [ ] Click Target icon in header (between search and +)
- [ ] Verify Session Context modal opens
- [ ] Click "New Profile"
- [ ] Enter profile name: "Work Session"
- [ ] Add 2 contexts, 2 projects, 2 tags
- [ ] Set priority to A
- [ ] Click "Save Profile"
- [ ] Verify profile appears in saved profiles table

### Test 3.2: Applying Session
- [ ] Select "Work Session" from dropdown
- [ ] Verify all fields populate
- [ ] Click "Apply Session"
- [ ] Verify modal closes
- [ ] **Check badge:** Badge should show count (e.g., "7" for 2+2+2+1)
- [ ] Open Quick Add modal
- [ ] Verify all session values are pre-filled
- [ ] Create a todo
- [ ] Verify it has all session values

### Test 3.3: Priority Order
**Setup:** Create session with tag "urgent", then search for "work"

- [ ] Open Quick Add modal
- [ ] **Expected:** Both "urgent" (from session) and "work" (from search) appear
- [ ] Verify session priority is set
- [ ] Clear priority and verify search doesn't override
- [ ] Test with default tags in settings
- [ ] Verify order: session > search > defaults

### Test 3.4: Session Management
- [ ] Create 3 different session profiles
- [ ] Switch between them
- [ ] Verify values change correctly
- [ ] Edit a profile (change name and values)
- [ ] Click "Save Profile"
- [ ] Verify changes persist
- [ ] Select a profile
- [ ] Click "Delete Profile"
- [ ] Verify it's removed from list
- [ ] Verify active session is cleared if it was the active one

### Test 3.5: Clear Session
- [ ] Apply a session
- [ ] Verify badge shows count
- [ ] Click Target icon
- [ ] Click "Clear Session"
- [ ] Verify badge disappears
- [ ] Open Quick Add modal
- [ ] Verify session values are gone
- [ ] Only search filters remain

### Test 3.6: Session Persistence
- [ ] Create and apply a session
- [ ] Close the app
- [ ] Reopen the app
- [ ] Verify badge still shows count
- [ ] Open Quick Add
- [ ] Verify session values still apply
- [ ] Open Session Context modal
- [ ] Verify profile is still selected

---

## 4. Integration Testing

### Test 4.1: All Features Together
- [ ] Create a template "Sprint Planning"
- [ ] Create a session profile "Current Sprint"
- [ ] Apply the session
- [ ] Open Quick Add
- [ ] Apply the template
- [ ] Verify values merge correctly
- [ ] Add manual values
- [ ] Verify nothing is overwritten
- [ ] Create the todo
- [ ] Verify all values are saved

### Test 4.2: Edit Existing Todo
- [ ] Click edit on an existing todo
- [ ] Verify autocomplete works in edit mode
- [ ] Change tags using autocomplete
- [ ] Change projects using autocomplete
- [ ] Save changes
- [ ] Verify changes persist

### Test 4.3: Search Filters + Session
- [ ] Apply session with context "work"
- [ ] Search for "+project"
- [ ] Open Quick Add
- [ ] Verify both "work" context AND "project" appear
- [ ] Change search to different project
- [ ] Open Quick Add again
- [ ] Verify session context stays, search project updates

### Test 4.4: Modal Interactions
- [ ] Open Session Context modal
- [ ] Click outside to close (should NOT close)
- [ ] Click Close button (should close)
- [ ] Open Template Save dialog
- [ ] Click outside (should close)
- [ ] Open autocomplete dropdown
- [ ] Click outside (should close dropdown)
- [ ] Verify no z-index issues between modals

---

## 5. UI/UX Testing

### Test 5.1: Visual Consistency
- [ ] Verify all autocomplete components match theme
- [ ] Verify chips have consistent styling
- [ ] Verify dropdowns have proper blur/backdrop
- [ ] Verify modal styling is consistent
- [ ] Verify buttons use proper badge classes
- [ ] Verify hover states work on all buttons

### Test 5.2: Keyboard Navigation
- [ ] Tab through all fields in Quick Add
- [ ] Verify focus indicators are visible
- [ ] Use autocomplete with only keyboard
- [ ] Navigate template dropdown with keyboard
- [ ] Test ⌘+Enter to submit while in autocomplete
- [ ] Test Escape to close dropdowns

### Test 5.3: Badge Indicator
- [ ] Apply session with 1 context
- [ ] Verify badge shows "1"
- [ ] Add project to session
- [ ] Verify badge shows "2"
- [ ] Add priority
- [ ] Verify badge shows "3"
- [ ] Clear session
- [ ] Verify badge disappears
- [ ] Verify badge positioning doesn't break layout

### Test 5.4: Responsive Behavior
- [ ] Test with many suggestions (50+)
- [ ] Verify dropdown scrolls
- [ ] Test with long tag names
- [ ] Verify chips wrap properly
- [ ] Test with 20+ chips
- [ ] Verify input container grows

---

## 6. Error Handling

### Test 6.1: Data Validation
- [ ] Try to save template with very long name (100+ chars)
- [ ] Try to save profile with special characters
- [ ] Try to add 50+ tags via autocomplete
- [ ] Verify app doesn't crash

### Test 6.2: Backend Failures
- [ ] Disconnect from backend (if possible)
- [ ] Try to load filters
- [ ] Verify graceful failure (empty suggestions)
- [ ] Reconnect
- [ ] Verify suggestions reload

### Test 6.3: localStorage Edge Cases
- [ ] Fill localStorage with many templates (50+)
- [ ] Verify app still loads
- [ ] Clear localStorage manually
- [ ] Reload app
- [ ] Verify no errors (empty state)

---

## 7. Performance Testing

### Test 7.1: Autocomplete Performance
- [ ] Type rapidly in autocomplete field
- [ ] Verify filtering is instant
- [ ] Test with 100+ suggestions
- [ ] Verify no lag in dropdown

### Test 7.2: Template/Session Loading
- [ ] Create 20 templates
- [ ] Open template picker
- [ ] Verify instant load
- [ ] Create 20 session profiles
- [ ] Open session modal
- [ ] Verify table loads quickly

---

## 8. Regression Testing

### Test 8.1: Existing Features
- [ ] Create todo with title input (with @, +, # in title)
- [ ] Verify inline suggestions still work
- [ ] Search for todos
- [ ] Verify search still works
- [ ] Test Quick Add modal (⌘N)
- [ ] Test Edit Todo modal
- [ ] Test Notes modal
- [ ] Test Settings modal
- [ ] Test all existing tab functionality
- [ ] Test view mode switching (Scope/Date)
- [ ] Test todo completion/archive
- [ ] Test default tags setting

### Test 8.2: Backward Compatibility
- [ ] Verify todos created before update still work
- [ ] Verify existing settings are preserved
- [ ] Verify no migration issues

---

## Expected Results Summary

After completing all tests:

✅ **Autocomplete:** Fast, accurate filtering with keyboard support
✅ **Templates:** Save and apply common configurations
✅ **Session Context:** Batch operations with persistent defaults
✅ **Integration:** All features work together seamlessly
✅ **Priority Order:** Session > Search > Defaults working correctly
✅ **UI/UX:** Consistent, responsive, accessible
✅ **Performance:** No lag, instant responses
✅ **Stability:** No crashes, errors handled gracefully

---

## Bug Reporting Template

If you find issues, report them with:

**Bug Title:** [Concise description]

**Steps to Reproduce:**
1.
2.
3.

**Expected Behavior:**

**Actual Behavior:**

**Component:** [Autocomplete/Template/Session/Integration]

**Priority:** [High/Medium/Low]

**Screenshots:** (if applicable)
