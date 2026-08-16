package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/hollis-labs/nil/apiserver"
	"github.com/hollis-labs/nil/chat"
	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/parse"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
	"github.com/hollis-labs/nil/vault"
)

// svc is the package-level items service used by every Wails-bound handler.
// Stateless; safe to share.
var svc = items.New()

type App struct {
	ctx        context.Context
	vaultMgr   *vault.Manager
	needsSetup bool
	apiServer  *http.Server
	apiMu      sync.Mutex

	// Chat addon (F5)
	chatStore       *chat.ChatStore
	chatRunner      *chat.ActionRunner
	chatBridge      *chat.Bridge
	sessionCaches   map[int64]*chat.ToolCache
	sessionCachesMu sync.Mutex
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load config (with migration and auto-defaults)
	cfg := config.LoadOrDefault()

	// Ensure API defaults (port, key) are present; save if anything changed
	if cfg.EnsureDefaults() {
		_ = config.Save(cfg)
	}

	// Initialize vault manager (opens active vault + shared inbox)
	vm, err := vault.NewManager(ctx, cfg)
	if err != nil {
		println("Failed to initialize vault manager:", err.Error())
		a.needsSetup = true
		return
	}
	a.vaultMgr = vm

	if cfg.APIEnabled {
		a.startAPIServer(cfg)
	}

	// Initialise chat addon
	cs, err := chat.Open(ctx, config.GetConfigDir())
	if err != nil {
		println("Failed to initialize chat store:", err.Error())
	} else {
		a.chatStore = cs
		a.chatRunner = chat.NewActionRunner(cs)
		a.chatBridge = &chat.Bridge{}
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.stopAPIServer()
	if a.vaultMgr != nil {
		a.vaultMgr.CloseAll()
	}
	if a.chatStore != nil {
		_ = a.chatStore.Close()
	}
}

func (a *App) startAPIServer(cfg *config.Config) {
	if a.vaultMgr == nil {
		return
	}
	a.apiMu.Lock()
	defer a.apiMu.Unlock()
	if a.apiServer != nil {
		return // already running
	}
	srv := apiserver.New(cfg, a.vaultMgr)
	a.apiServer = srv
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			println("API server error:", err.Error())
		}
	}()
}

func (a *App) stopAPIServer() {
	a.apiMu.Lock()
	defer a.apiMu.Unlock()
	if a.apiServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = a.apiServer.Shutdown(ctx)
	a.apiServer = nil
}

// --- Setup / config methods ---

// NeedsSetup returns true if the app needs initial database configuration
func (a *App) NeedsSetup() bool {
	return a.needsSetup
}

// GetDatabasePath returns the active vault's directory path
func (a *App) GetDatabasePath() (string, error) {
	if a.vaultMgr != nil {
		if v := a.vaultMgr.GetActiveVault(); v != nil {
			return v.Path, nil
		}
	}
	return config.GetDefaultDatabasePath(), nil
}

// GetDefaultDatabasePath returns the OS-appropriate default path
func (a *App) GetDefaultDatabasePath() string {
	return config.GetDefaultDatabasePath()
}

// SetDatabasePath updates the active vault's path. Kept for backward compat
// with the setup/settings UI; new code should use vault CRUD methods.
func (a *App) SetDatabasePath(path string) error {
	// Validate path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}
	testFile := path + "/.nil-write-test"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("directory not writable: %s", path)
	}
	os.Remove(testFile)

	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	// Update the active vault's path in registry
	for i := range cfg.Vaults {
		if cfg.Vaults[i].ID == cfg.ActiveVaultID {
			cfg.Vaults[i].Path = path
			break
		}
	}
	// Fallback: legacy single-path field
	if len(cfg.Vaults) == 0 {
		cfg.DatabasePath = path
	}
	return config.Save(cfg)
}

// --- Vault management methods (Wails-bound) ---

// GetVaults returns all registered vaults.
func (a *App) GetVaults() ([]config.Vault, error) {
	if a.vaultMgr == nil {
		return nil, fmt.Errorf("vault manager not initialized")
	}
	return a.vaultMgr.GetVaults(), nil
}

// GetActiveVault returns the currently active vault metadata.
func (a *App) GetActiveVault() (*config.Vault, error) {
	if a.vaultMgr == nil {
		return nil, fmt.Errorf("vault manager not initialized")
	}
	v := a.vaultMgr.GetActiveVault()
	if v == nil {
		return nil, fmt.Errorf("no active vault")
	}
	return v, nil
}

// CreateVault registers a new vault at the given directory path.
func (a *App) CreateVault(name, dir string) (*config.Vault, error) {
	if a.vaultMgr == nil {
		return nil, fmt.Errorf("vault manager not initialized")
	}
	return a.vaultMgr.CreateVault(name, dir)
}

// RenameVault updates the display name of a vault.
func (a *App) RenameVault(id, name string) error {
	if a.vaultMgr == nil {
		return fmt.Errorf("vault manager not initialized")
	}
	return a.vaultMgr.RenameVault(id, name)
}

// DeleteVault removes a vault from the registry (files on disk are preserved).
func (a *App) DeleteVault(id string) error {
	if a.vaultMgr == nil {
		return fmt.Errorf("vault manager not initialized")
	}
	return a.vaultMgr.DeleteVault(id)
}

// SwitchVault changes the active vault. Returns the new active vault's metadata.
func (a *App) SwitchVault(id string) (*config.Vault, error) {
	if a.vaultMgr == nil {
		return nil, fmt.Errorf("vault manager not initialized")
	}
	if err := a.vaultMgr.SwitchVault(id); err != nil {
		return nil, err
	}
	return a.vaultMgr.GetActiveVault(), nil
}

// --- API config ---

// APIConfigResult is the shape returned to the Settings UI.
type APIConfigResult struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	APIKey  string `json:"api_key"`
}

// GetAPIConfig returns the current API configuration for display in Settings.
func (a *App) GetAPIConfig() (*APIConfigResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	cfg.EnsureDefaults()
	return &APIConfigResult{
		Enabled: cfg.APIEnabled,
		Port:    cfg.APIPort,
		APIKey:  cfg.APIKey,
	}, nil
}

// SetAPIEnabled enables or disables the local HTTP API at runtime.
func (a *App) SetAPIEnabled(enabled bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.EnsureDefaults()
	cfg.APIEnabled = enabled
	if err := config.Save(cfg); err != nil {
		return err
	}
	if enabled {
		a.startAPIServer(cfg)
	} else {
		a.stopAPIServer()
	}
	return nil
}

// --- Helper: find which store holds an item by ID ---

// storeForItemID tries the active vault then the inbox store to locate an item.
func (a *App) storeForItemID(id int64) (*store.Store, error) {
	if s := a.vaultMgr.ActiveStore(); s != nil {
		if _, err := s.GetItem(a.ctx, id); err == nil {
			return s, nil
		}
	}
	if s := a.vaultMgr.InboxStore(); s != nil {
		if _, err := s.GetItem(a.ctx, id); err == nil {
			return s, nil
		}
	}
	return nil, fmt.Errorf("item %d not found", id)
}

// --- Item CRUD ---

func (a *App) CreateItemFromLine(line string) (*store.Item, error) {
	return a.createFromLine(line, "todo")
}

func (a *App) CreateNoteFromLine(line string) (*store.Item, error) {
	return a.createFromLine(line, "note")
}

// CreateScratchFromLine creates a scratch-kind item. Scratch items behave
// like notes until the scratch UX follow-up defines dedicated behavior.
func (a *App) CreateScratchFromLine(line string) (*store.Item, error) {
	return a.createFromLine(line, "scratch")
}

// createFromLine is the shared GUI create path. Parses the todo.txt-style
// line and routes blank-title items to the shared inbox store, matching the
// "capture without friction" UX from F1.
func (a *App) createFromLine(line, kind string) (*store.Item, error) {
	p := parse.ParseLine(line)
	input := items.CreateInput{
		Title:      p.Title,
		Kind:       kind,
		Priority:   p.Priority,
		DueAt:      p.Due,
		Threshold:  p.Thresh,
		Projects:   p.Projects,
		Contexts:   p.Contexts,
		Tags:       p.Tags,
		SourceLine: line,
	}
	// Only the todo kind carries scheduling fields; clear them for note/scratch
	// to preserve the previous per-kind behavior.
	if kind != "todo" {
		input.Priority = nil
		input.DueAt = nil
		input.Threshold = nil
	}
	if strings.TrimSpace(input.Title) == "" {
		input.Inbox = true
		return svc.Create(a.ctx, a.vaultMgr.InboxStore(), input)
	}
	return svc.Create(a.ctx, a.vaultMgr.ActiveStore(), input)
}

func (a *App) GetItem(id int64) (*store.Item, error) {
	// Try active store first, then inbox
	if s := a.vaultMgr.ActiveStore(); s != nil {
		if item, err := s.GetItem(a.ctx, id); err == nil {
			return item, nil
		}
	}
	if s := a.vaultMgr.InboxStore(); s != nil {
		if item, err := s.GetItem(a.ctx, id); err == nil {
			return item, nil
		}
	}
	return nil, fmt.Errorf("item %d not found", id)
}

func (a *App) GetBackrefs(id int64) ([]store.Item, error) {
	return a.vaultMgr.ActiveStore().GetBackrefs(a.ctx, id)
}

// UpdateItem routes the update to the correct store based on the item's inbox flag.
// If an item with inbox=true exists in the active vault, it is moved to the inbox store.
func (a *App) UpdateItem(t store.Item) error {
	if t.Inbox {
		// Check if item currently lives in the active vault (explicit "→ Inbox" routing)
		if s := a.vaultMgr.ActiveStore(); s != nil {
			if _, err := s.GetItem(a.ctx, t.ID); err == nil {
				// Move: create copy in inbox store, delete from active vault
				newItem := t
				newItem.ID = 0
				if _, err := a.vaultMgr.InboxStore().CreateItem(a.ctx, &newItem); err != nil {
					return fmt.Errorf("creating item in inbox: %w", err)
				}
				return s.DeleteItem(a.ctx, t.ID)
			}
		}
		// Item is already in inbox store — update in place
		return svc.Update(a.ctx, a.vaultMgr.InboxStore(), &t)
	}
	return svc.Update(a.ctx, a.vaultMgr.ActiveStore(), &t)
}

func (a *App) ToggleComplete(id int64, completed bool) error {
	s, err := a.storeForItemID(id)
	if err != nil {
		return err
	}
	return s.ToggleComplete(a.ctx, id, completed)
}

func (a *App) Archive(id int64, archived bool) error {
	s, err := a.storeForItemID(id)
	if err != nil {
		return err
	}
	return s.Archive(a.ctx, id, archived)
}

func (a *App) DeleteItem(id int64) error {
	s, err := a.storeForItemID(id)
	if err != nil {
		return err
	}
	return s.DeleteItem(a.ctx, id)
}

func (a *App) Search(req store.SearchRequest) ([]store.Item, error) {
	req.Query = strings.TrimSpace(req.Query)
	println("Backend search query:", req.Query, "statuses:", len(req.Statuses))
	results, err := svc.Search(a.ctx, a.vaultMgr.ActiveStore(), req)
	println("Backend search returned:", len(results), "results, error:", err)
	return results, err
}

type FiltersResult struct {
	Projects []string `json:"projects"`
	Contexts []string `json:"contexts"`
	Tags     []string `json:"tags"`
}

func (a *App) GetFilters() (*FiltersResult, error) {
	projects, contexts, tags, err := a.vaultMgr.ActiveStore().GetFilterValues(a.ctx)
	if err != nil {
		return nil, err
	}
	if projects == nil {
		projects = []string{}
	}
	if contexts == nil {
		contexts = []string{}
	}
	if tags == nil {
		tags = []string{}
	}
	return &FiltersResult{
		Projects: projects,
		Contexts: contexts,
		Tags:     tags,
	}, nil
}

// --- Inbox methods ---

func (a *App) GetInboxCount() (int, error) {
	return svc.InboxCount(a.ctx, a.vaultMgr.InboxStore())
}

func (a *App) GetInboxItems(req store.SearchRequest) ([]store.Item, error) {
	return svc.ListInbox(a.ctx, a.vaultMgr.InboxStore(), req)
}

// ProcessInboxItem moves an inbox item to the target vault.
// If targetVaultID is empty, defaults to the currently active vault.
func (a *App) ProcessInboxItem(id int64, targetVaultID string) error {
	if targetVaultID == "" {
		targetVaultID = a.vaultMgr.GetActiveVaultID()
	}
	return a.vaultMgr.MoveItemToVault(a.ctx, id, targetVaultID)
}

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
		if err := a.vaultMgr.ActiveStore().DeleteItem(a.ctx, int64(todo.ID)); err != nil {
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

// --- Chat addon (F5) ---

// chatReady returns an error if the chat subsystem failed to initialise.
func (a *App) chatReady() error {
	if a.chatStore == nil || a.chatBridge == nil || a.chatRunner == nil {
		return fmt.Errorf("chat subsystem not initialised")
	}
	return nil
}

// StartChatSession opens a new chat session against the active vault.
func (a *App) StartChatSession() (*chat.ChatSession, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	dryRun := cfg.Chat.DryRun
	vaultID := ""
	if a.vaultMgr != nil {
		vaultID = a.vaultMgr.GetActiveVaultID()
	}
	return a.chatStore.CreateSession(a.ctx, vaultID, dryRun)
}

// getSessionCache returns (creating if needed) the in-memory tool cache for a session.
func (a *App) getSessionCache(sessionID int64) *chat.ToolCache {
	a.sessionCachesMu.Lock()
	defer a.sessionCachesMu.Unlock()
	if a.sessionCaches == nil {
		a.sessionCaches = make(map[int64]*chat.ToolCache)
	}
	if c, ok := a.sessionCaches[sessionID]; ok {
		return c
	}
	c := chat.NewToolCache(20)
	a.sessionCaches[sessionID] = c
	return c
}

// dropSessionCache frees the in-memory cache when a session ends.
func (a *App) dropSessionCache(sessionID int64) {
	a.sessionCachesMu.Lock()
	defer a.sessionCachesMu.Unlock()
	delete(a.sessionCaches, sessionID)
}

// EndChatSession closes the given session and frees its tool cache.
func (a *App) EndChatSession(sessionID int64) error {
	if err := a.chatReady(); err != nil {
		return err
	}
	a.dropSessionCache(sessionID)
	return a.chatStore.EndSession(a.ctx, sessionID)
}

// SendChatMessage sends a user message, calls the LLM, and returns the response.
// If the LLM proposes an action, the proposal is persisted and its ID returned.
func (a *App) SendChatMessage(sessionID int64, content string) (*chat.ChatResponse, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Persist the user message.
	userMsg, err := a.chatStore.AddMessage(a.ctx, &chat.ChatMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   content,
	})
	if err != nil {
		return nil, fmt.Errorf("chat: store user message: %w", err)
	}

	// Load session for context.
	sess, err := a.chatStore.GetSession(a.ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Get active vault metadata.
	vaultName := "Default"
	caps := chat.VaultCaps{Read: true}
	if a.vaultMgr != nil {
		if v := a.vaultMgr.GetActiveVault(); v != nil {
			vaultName = v.Name
			if vc, ok := cfg.Chat.VaultCaps[v.ID]; ok {
				caps = chat.VaultCaps{Read: true, Write: vc.Write, Delete: vc.Delete, DirectCreate: vc.DirectCreate}
			}
		}
	}

	// Load message history for context window.
	history, _ := a.chatStore.GetMessages(a.ctx, sessionID)
	_ = userMsg // already in history

	// Determine the active store for tool execution.
	var activeStore chat.BridgeStore
	if a.vaultMgr != nil {
		if s := a.vaultMgr.ActiveStore(); s != nil {
			activeStore = s
		}
	}

	// Call the LLM (agentic tool-use loop inside Bridge.Send).
	resp, err := a.chatBridge.Send(a.ctx, chat.BridgeRequest{
		APIKey:      cfg.Chat.APIKey,
		Model:       cfg.Chat.Model,
		VaultID:     sess.VaultID,
		VaultName:   vaultName,
		Caps:        caps,
		DryRun:      sess.DryRun,
		History:     history,
		UserMessage: content,
		Store:       activeStore,
		Templates:   a.chatStore,
		ToolCache:   a.getSessionCache(sessionID),
	})
	if err != nil {
		return nil, err
	}

	// Persist the assistant message.
	resp.Message.SessionID = sessionID
	assistantMsg, err := a.chatStore.AddMessage(a.ctx, &resp.Message)
	if err != nil {
		return nil, fmt.Errorf("chat: store assistant message: %w", err)
	}
	resp.Message = *assistantMsg

	// Persist tool calls made during this turn (best-effort; non-fatal on error).
	for _, tc := range resp.ToolCalls {
		_ = a.chatStore.PersistToolCall(a.ctx, sessionID, assistantMsg.ID, tc)
	}

	// If the LLM proposed an action, persist it.
	if resp.Proposal != nil {
		resp.Proposal.SessionID = sessionID
		resp.Proposal.MessageID = &assistantMsg.ID
		proposalID, err := a.chatRunner.Propose(a.ctx, resp.Proposal)
		if err != nil {
			return resp, fmt.Errorf("chat: persist proposal: %w", err)
		}
		resp.ProposalID = &proposalID
		// Reload full proposal with DB-assigned fields.
		full, _ := a.chatStore.GetProposal(a.ctx, proposalID)
		resp.Proposal = full
	}

	return resp, nil
}

// ApproveChatAction executes an approved ActionProposal.
func (a *App) ApproveChatAction(proposalID int64) (*chat.ActionResult, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	p, err := a.chatStore.GetProposal(a.ctx, proposalID)
	if err != nil {
		return nil, err
	}

	caps := chat.VaultCaps{Read: true}
	if vc, ok := cfg.Chat.VaultCaps[p.VaultID]; ok {
		caps = chat.VaultCaps{Read: true, Write: vc.Write, Delete: vc.Delete, DirectCreate: vc.DirectCreate}
	}

	return a.chatRunner.Approve(a.ctx, proposalID, a.vaultMgr.ActiveStore(), cfg.Chat.DryRun, caps)
}

// DenyChatAction rejects a pending ActionProposal.
func (a *App) DenyChatAction(proposalID int64) error {
	if err := a.chatReady(); err != nil {
		return err
	}
	return a.chatRunner.Deny(a.ctx, proposalID)
}

// GetChatHistory returns all messages for a session.
func (a *App) GetChatHistory(sessionID int64) ([]chat.ChatMessage, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	return a.chatStore.GetMessages(a.ctx, sessionID)
}

// GetActionAudit returns the most recent audit entries.
func (a *App) GetActionAudit(limit int) ([]chat.AuditEntry, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	return a.chatStore.GetAudit(a.ctx, limit)
}

// GetChatConfig returns the current chat configuration.
func (a *App) GetChatConfig() (*config.ChatConfig, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &cfg.Chat, nil
}

// SetChatConfig persists updated chat configuration.
func (a *App) SetChatConfig(chatCfg config.ChatConfig) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.Chat = chatCfg
	// Reload bridge with new API key/model on next send (stateless Bridge, no action needed).
	return config.Save(cfg)
}

// ListChatTemplates returns all templates stored in chat.db.
func (a *App) ListChatTemplates() ([]chat.Template, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	tmpl, err := a.chatStore.ListTemplates(a.ctx)
	if tmpl == nil {
		tmpl = []chat.Template{}
	}
	return tmpl, err
}

// SaveChatTemplate creates or updates a template.
// If a template with the same slug already exists it is updated; otherwise it is created.
func (a *App) SaveChatTemplate(t chat.Template) (*chat.Template, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	// Ensure defaults.
	if t.Parameters == "" {
		t.Parameters = "[]"
	}
	if t.OutputFormat == "" {
		t.OutputFormat = "markdown"
	}
	if t.Type == "" {
		t.Type = "generation"
	}
	// Upsert: check if slug exists.
	existing, _ := a.chatStore.GetTemplate(a.ctx, t.Slug)
	if existing != nil {
		if err := a.chatStore.UpdateTemplate(a.ctx, &t); err != nil {
			return nil, err
		}
		return a.chatStore.GetTemplate(a.ctx, t.Slug)
	}
	return a.chatStore.CreateTemplate(a.ctx, &t)
}

// DeleteChatTemplate removes a template by slug.
func (a *App) DeleteChatTemplate(slug string) error {
	if err := a.chatReady(); err != nil {
		return err
	}
	return a.chatStore.DeleteTemplate(a.ctx, slug)
}

// Restart restarts the application
func (a *App) Restart() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	cmd := exec.Command(executable, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart: %w", err)
	}
	os.Exit(0)
	return nil
}
