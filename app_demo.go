package main

import (
	"fmt"
	"strings"

	"github.com/hollis-labs/nil/parse"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
)

// --- Demo data ---

const demoDataTag = "nil-demo"

func (a *App) HasDemoData() (bool, error) {
	if a.vaultMgr == nil || a.vaultMgr.ActiveStore() == nil {
		return false, nil
	}
	req := store.SearchRequest{
		Tags:     []string{demoDataTag},
		PageSize: 1,
		Kind:     "todo",
	}
	results, err := svc.Search(a.ctx, a.vaultMgr.ActiveStore(), req)
	if err != nil {
		return false, err
	}
	return len(results) > 0, nil
}

type todoWithMeta struct {
	line    string
	section string
	notes   string
}

func (a *App) SeedDemoData() error {
	if a.vaultMgr == nil || a.vaultMgr.ActiveStore() == nil {
		return fmt.Errorf("database not initialized")
	}

	demos := []todoWithMeta{
		{
			line:    "(A) Welcome to NIL! Click me to see notes due:2025-11-01 +tutorial @getting-started #now",
			section: "now",
			notes:   "# Welcome!\n\nNIL is a todo.txt task manager.\n\n**Quick Start:**\n- Click any todo to view/edit notes\n- Press ⌘N to add todos\n- Right-click for actions\n- Search with +project @context #tag",
		},
		{
			line:    "(A) Learn todo.txt syntax - (A)=high +project @context #tag due:2025-10-25 +tutorial @getting-started #now",
			section: "now",
			notes:   "# Todo.txt Format\n\nFormat: `(A) Title +project @context #tag due:2025-12-31`\n\n**Priority:** (A), (B), (C)\n**+project** - Group related tasks\n**@context** - Location/tool (@home, @computer)\n**#tag** - Flexible labels\n**due:YYYY-MM-DD** - Deadlines",
		},
		{
			line:    "(A) Try search: type keywords, +project, @context, #tag, pri:A due:2025-10-26 +tutorial @getting-started #now",
			section: "now",
			notes:   "# Search Power\n\n**Examples:**\n- `welcome` - keyword search\n- `+tutorial` - project filter\n- `#now` - tag filter\n- `pri:A` - priority\n- `+work -meeting` - exclude with minus\n\nTry searching for `#now` right now!",
		},
		{
			line:    "(B) Explore Now/Soon/Anytime scope views due:2025-10-28 +tutorial @features #soon",
			section: "soon",
			notes:   "# Scope Views\n\n**Now** - Current focus (3-5 items)\n**Soon** - Upcoming tasks\n**Anytime** - Backlog\n\nRight-click any todo → Move to... → Pick section",
		},
		{
			line:    "(B) Set Session Context (🎯 icon) for focused work due:2025-10-30 +tutorial @features #soon",
			section: "soon",
			notes:   "# Session Context\n\nTemporary filter for deep work!\n\n1. Click 🎯 icon\n2. Select filters (e.g., +work @computer)\n3. Check \"Use as Filter Tab\"\n4. Work on just those tasks\n\nGreat for: focus blocks, errands, end-of-day priority items",
		},
		{
			line:    "(B) Create custom tabs in Settings for saved views due:2025-11-05 +tutorial @features #soon",
			section: "soon",
			notes:   "# Tabs\n\nSave common searches as tabs!\n\n**Examples:**\n- Work: `+work`\n- Today: `due<=today`\n- Urgent: `#urgent`\n\nSettings ⚙️ → Tabs section → Add Tab",
		},
		{
			line:    "(B) Use due:YYYY-MM-DD and t:YYYY-MM-DD (threshold) dates due:2025-11-08 +tutorial @time-management #soon",
			section: "soon",
			notes:   "# Dates\n\n**due:** - Deadline\n**t:** - Hide until (threshold)\n\nExample: `Buy gifts due:2025-12-20 t:2025-12-01` hides until December\n\nSwitch between Scope and Date views with buttons",
		},
		{
			line:    "(C) Sync with iCloud Drive across devices due:2025-12-01 +tutorial @sync #icloud #anytime",
			section: "anytime",
			notes:   "# iCloud Sync\n\n1. Create folder: `~/Library/Mobile Documents/com~apple~CloudDocs/NIL`\n2. Settings → Database Location → Change\n3. Paste path, restart\n4. On other devices: Use Existing Database\n\nWorks with Dropbox/Google Drive too!",
		},
		{
			line:    "Keyboard shortcuts: ⌘N=new, Esc=clear +tutorial @shortcuts #anytime",
			section: "anytime",
			notes:   "# Shortcuts\n\n- **⌘N** - Quick add\n- **Esc** - Clear/close\n- **Enter** - Search\n\nMore coming soon!",
		},
		{
			line:    "Customize themes in Settings ⚙️ +tutorial @settings #anytime",
			section: "anytime",
			notes:   "# Themes\n\n20+ themes available!\n\n- Catppuccin (4 variants)\n- Gruvbox\n- Tokyo Night\n- Nord\n- Dracula\n- Solarized\n\nSettings → Theme section",
		},
		{
			line:    "Right-click for Complete, Edit, Notes, Move, Archive, Delete +tutorial @features #anytime",
			section: "anytime",
			notes:   "# Right-Click Menu\n\n- Complete/Uncomplete\n- Edit todo\n- Add markdown notes\n- Move to Now/Soon/Anytime\n- Archive (hide completed)\n- Delete\n\nTry it on this todo!",
		},
		{
			line:    "Export/import todo.txt in Settings due:2025-12-15 +tutorial @backup #anytime",
			section: "anytime",
			notes:   "# Import & Export\n\nSettings → Import/Export section\n\n**Export** - Backup as todo.txt\n**Import** - Migrate from other apps\n\nStandard todo.txt format works everywhere!",
		},
		{
			line:    "Remove all tutorial data with header button when ready +tutorial @getting-started #anytime",
			section: "anytime",
			notes:   "# Clean Up\n\nWhen comfortable:\n\n1. Click \"Remove Tutorial\" button in header\n2. All tutorial todos deleted\n3. Your todos stay!\n4. Start fresh\n\nGood luck! 🚀",
		},
	}

	for _, demo := range demos {
		p := parse.ParseLine(demo.line)
		input := items.CreateInput{
			Title:      p.Title,
			Priority:   p.Priority,
			Projects:   p.Projects,
			Contexts:   p.Contexts,
			Tags:       append(p.Tags, demoDataTag),
			DueAt:      p.Due,
			Threshold:  p.Thresh,
			SourceLine: demo.line,
			Section:    demo.section,
			NotesMD:    demo.notes,
		}
		if _, err := svc.Create(a.ctx, a.vaultMgr.ActiveStore(), input); err != nil {
			return err
		}
	}
	return nil
}

// ListKinds returns all registered kinds for the kind switcher UI.
func (a *App) ListKinds() ([]store.Kind, error) {
	if a.vaultMgr == nil || a.vaultMgr.ActiveStore() == nil {
		return []store.Kind{}, nil
	}
	return a.vaultMgr.ActiveStore().ListKinds(a.ctx)
}

func (a *App) RemoveDemoData() error {
	if a.vaultMgr == nil || a.vaultMgr.ActiveStore() == nil {
		return fmt.Errorf("database not initialized")
	}
	req := store.SearchRequest{
		Tags:     []string{demoDataTag},
		PageSize: 500,
		Kind:     "todo",
	}
	results, err := svc.Search(a.ctx, a.vaultMgr.ActiveStore(), req)
	if err != nil {
		return err
	}
	for _, todo := range results {
		if err := a.vaultMgr.ActiveStore().DeleteItem(a.ctx, todo.ID); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) ExportTodoTxt() (string, error) {
	req := store.SearchRequest{
		Query:    "",
		Page:     0,
		PageSize: 10000,
		SortBy:   "created_at",
		SortDir:  "asc",
		Kind:     "todo",
	}
	todos, err := svc.Search(a.ctx, a.vaultMgr.ActiveStore(), req)
	if err != nil {
		return "", err
	}
	var lines []string
	for _, t := range todos {
		lines = append(lines, t.Source)
	}
	return strings.Join(lines, "\n"), nil
}

func (a *App) ImportTodoTxt(content string) error {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, err := a.CreateItemFromLine(line); err != nil {
			return err
		}
	}
	return nil
}
