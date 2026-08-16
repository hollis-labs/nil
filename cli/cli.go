package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	iofs "io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/hollis-labs/nil/apiserver"
	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/contextcache"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
	"github.com/hollis-labs/nil/vault"
)

// svc is the stateless items service shared by every CLI command.
var svc = items.New()

const cliVersion = "dev-snapshot"

type command struct {
	name    string
	aliases []string
	short   string
	usage   string
	run     func(context.Context, []string, *commandEnv)
}

type commandEnv struct {
	cfg *config.Config
	mgr *vault.Manager
}

var commandRegistry []command

func init() {
	commandRegistry = []command{
		{
			name:  "add",
			short: "Create todos/notes from inline text or files",
			usage: "nil add [title] [--file path] [--body text] [--type todo|note] [--vault id]",
			run:   cmdAdd,
		},
		{
			name:  "list",
			short: "List common views like latest, inbox, today",
			usage: "nil list [view] [--vault id] [--limit N] [--query q] [--updated-since RFC3339]",
			run:   cmdList,
		},
		{
			name:    "show",
			aliases: []string{"view"},
			short:   "Show a single item by ID",
			usage:   "nil show <id> [--vault id] [--format json|detail]",
			run:     cmdShow,
		},
		{
			name:  "context",
			short: "Emit agent-friendly context snapshots",
			usage: "nil context [topic]",
			run:   cmdContext,
		},
		{
			name:  "import",
			short: "Bulk import text/markdown files from a directory",
			usage: "nil import --dir PATH [--dry-run] [--batch-size N]",
			run:   cmdImport,
		},
		{
			name:  "update",
			short: "Batch update, archive, or delete items",
			usage: "nil update --ids 1,2 --complete --archive",
			run:   cmdUpdate,
		},
		{
			name:    "push",
			aliases: []string{"legacy-add"},
			short:   "Create an item via the legacy push flow",
			usage:   "nil push <title> [flags]",
			run: func(ctx context.Context, args []string, env *commandEnv) {
				cmdPush(ctx, args, env.mgr)
			},
		},
		{
			name:  "search",
			short: "Search items in the active or specified vault",
			usage: "nil search <keywords> [flags]",
			run: func(ctx context.Context, args []string, env *commandEnv) {
				cmdSearch(ctx, args, env.mgr)
			},
		},
		{
			name:  "inbox",
			short: "List inbox items",
			usage: "nil inbox [keywords]",
			run: func(ctx context.Context, args []string, env *commandEnv) {
				cmdInbox(ctx, args, env.mgr)
			},
		},
		{
			name:  "get",
			short: "Fetch a single item by ID",
			usage: "nil get <id> [--vault id|inbox]",
			run: func(ctx context.Context, args []string, env *commandEnv) {
				cmdGet(ctx, args, env.mgr)
			},
		},
		{
			name:  "vaults",
			short: "List configured vaults",
			usage: "nil vaults",
			run: func(ctx context.Context, args []string, env *commandEnv) {
				cmdVaults(env.cfg, env.mgr)
			},
		},
		{
			name:  "version",
			short: "Print CLI version info",
			usage: "nil version",
			run:   cmdVersion,
		},
		{
			name:  "serve-api",
			short: "Run the local HTTP API standalone, without the GUI",
			usage: "nil serve-api [--port N]",
			run:   cmdServeAPI,
		},
	}
}

// envelope wraps all CLI output.
type envelope struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// cmdHelp prints available commands or detailed usage for a specific command.
func cmdHelp(ctx context.Context, args []string, env *commandEnv) {
	if len(args) == 0 {
		fmt.Println("Usage: nil <command> [arguments]")
		fmt.Println()
		fmt.Println("Commands:")
		for _, cmd := range commandRegistry {
			if cmd.name == "help" {
				continue
			}
			fmt.Printf("  %-10s %s\n", cmd.name, cmd.short)
		}
		fmt.Println("\nRun 'nil help <command>' for details.")
		return
	}
	name := args[0]
	cmd := findCommand(name)
	if cmd == nil {
		die("help: unknown command %q", name)
	}
	fmt.Printf("Usage: %s\n\n%s\n", cmd.usage, cmd.short)
	if len(cmd.aliases) > 0 {
		fmt.Printf("Aliases: %s\n", strings.Join(cmd.aliases, ", "))
	}
}

// cmdVersion emits CLI build metadata.
func cmdVersion(ctx context.Context, args []string, env *commandEnv) {
	data := map[string]string{
		"version":  cliVersion,
		"go":       runtime.Version(),
		"platform": runtime.GOOS + "/" + runtime.GOARCH,
	}
	printJSON(envelope{OK: true, Data: data})
}

// cmdServeAPI starts the same HTTP API server the GUI embeds (see
// apiserver.New), but as a standalone, long-running process with no Wails
// dependency. It reads/writes the same config.json and vault registry as
// the GUI and CLI, so it works whether or not the desktop app has ever been
// launched on this machine — EnsureDefaults seeds an API port/key on first
// run just like app.go's startup() does for the GUI.
//
// Blocks until SIGINT/SIGTERM, then shuts the HTTP server down gracefully
// and closes all open vault stores before returning.
func cmdServeAPI(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("serve-api", flag.ContinueOnError)
	port := fs.Int("port", 0, "override the configured API port for this run only (not persisted)")
	if _, err := parseInterspersed(args, fs); err != nil {
		die("serve-api: %v", err)
	}

	// Work on a copy so a --port override never mutates env.cfg or gets
	// persisted, but EnsureDefaults (port/key seeding) is saved for real —
	// otherwise a machine that has only ever used the CLI would have no
	// APIKey and every request would be rejected.
	cfg := *env.cfg
	if cfg.EnsureDefaults() {
		if err := config.Save(&cfg); err != nil {
			fmt.Fprintf(os.Stderr, "nil: warning: failed to persist API defaults: %v\n", err)
		}
	}
	if *port != 0 {
		cfg.APIPort = *port
	}

	srv := apiserver.New(&cfg, env.mgr)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	fmt.Fprintf(os.Stderr, "nil: API server listening on %s (Ctrl+C to stop)\n", srv.Addr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			die("serve-api: %v (is NIL already running with the API enabled on this port?)", err)
		}
	case <-sigCh:
		fmt.Fprintln(os.Stderr, "nil: shutting down API server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "nil: warning: API server shutdown error: %v\n", err)
		}
	}

	env.mgr.CloseAll()
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func cmdAdd(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	file := fs.String("file", "", "path to UTF-8 text/markdown file for the body")
	bodyFlag := fs.String("body", "", "inline body text")
	formatFlag := fs.String("format", "md", "body format: md|html|json (default md)")
	kindFlag := fs.String("kind", "todo", "item kind: todo|note|scratch (or registered kind)")
	vaultID := fs.String("vault", "", "vault ID to store the item")
	inbox := fs.Bool("inbox", false, "route to inbox even if --vault is set")
	titleFlag := fs.String("title", "", "explicit title override")
	source := fs.String("source", "cli", "source label stored in api_source")
	tags := fs.String("tags", "", "comma-separated tags")
	contexts := fs.String("contexts", "", "comma-separated contexts")
	projects := fs.String("projects", "", "comma-separated projects")
	priority := fs.String("priority", "", "priority letter A|B|C")
	due := fs.String("due", "", "due date YYYY-MM-DD")
	var metaPairs stringSliceFlag
	fs.Var(&metaPairs, "meta", "metadata key=value (repeatable; stored as meta:<key>=<value> tags for now)")
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("add: %v", err)
	}
	if *file != "" && *bodyFlag != "" {
		die("add: use either --file or --body, not both")
	}
	body := *bodyFlag
	if *file != "" {
		body, err = readTextFile(*file)
		if err != nil {
			die("add: %v", err)
		}
	}
	title := strings.TrimSpace(strings.Join(positional, " "))
	if *titleFlag != "" {
		title = *titleFlag
	}
	if title == "" && *file != "" {
		title = strings.TrimSuffix(filepath.Base(*file), filepath.Ext(*file))
	}
	if title == "" {
		die("add: title is required (pass as positional args or --title)")
	}
	storeDest, toInbox, err := chooseCreateDestination(env.mgr, *vaultID, *inbox)
	if err != nil {
		die("add: %v", err)
	}
	input := items.CreateInput{
		Title:     title,
		Kind:      *kindFlag,
		APISource: *source,
		Tags:      appendMetaTags(splitCSV(*tags), parseKeyValuePairs(metaPairs)),
		Contexts:  splitCSV(*contexts),
		Projects:  splitCSV(*projects),
		Inbox:     toInbox,
	}
	if *priority != "" {
		p := strings.ToUpper(*priority)
		input.Priority = &p
	}
	if *due != "" {
		d := *due
		input.DueAt = &d
	}
	doc, md, htmlBody, err := svc.NotesInputFromCLI(body, *formatFlag)
	if err != nil {
		die("add: parse body: %v", err)
	}
	input.NotesDoc = doc
	input.NotesMD = md
	input.NotesHTML = htmlBody
	created, err := svc.Create(ctx, storeDest, input)
	if err != nil {
		die("add: %v", err)
	}
	printJSON(envelope{OK: true, Data: map[string]any{
		"command": "add",
		"item":    created,
	}})
}

func cmdList(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	viewFlag := fs.String("view", "latest", "view name: latest|inbox|today|overdue|high|notes|todos")
	vaultID := fs.String("vault", "", "vault id override")
	limit := fs.Int("limit", 50, "max results")
	query := fs.String("query", "", "keyword query")
	includeInbox := fs.Bool("include-inbox", false, "include inbox results in non-inbox views")
	updatedSince := fs.String("updated-since", "", "only items updated on/after this RFC3339 timestamp (e.g. 2026-08-01T00:00:00Z); not honored by the inbox view")
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("list: %v", err)
	}
	view := *viewFlag
	if len(positional) > 0 {
		view = positional[0]
	}
	view = strings.ToLower(strings.TrimSpace(view))
	if view == "" {
		view = "latest"
	}
	var items []store.Item
	switch view {
	case "inbox":
		inboxStore := env.mgr.InboxStore()
		if inboxStore == nil {
			die("list: inbox store not available")
		}
		results, err := inboxStore.GetInboxItems(ctx, store.SearchRequest{Query: *query, PageSize: *limit})
		if err != nil {
			die("list: %v", err)
		}
		items = results
	default:
		req := store.SearchRequest{
			Query:        *query,
			PageSize:     *limit,
			IncludeInbox: *includeInbox,
			Kind:         "all",
			UpdatedSince: *updatedSince,
		}
		switch view {
		case "today":
			req.Statuses = []string{"today"}
		case "overdue":
			req.Statuses = []string{"overdue"}
		case "high":
			req.Priorities = []string{"A"}
		case "notes":
			req.Kind = "note"
		case "todos":
			req.Kind = "todo"
		case "scratch":
			req.Kind = "scratch"
		}
		storeDest, err := selectVaultStore(env.mgr, *vaultID)
		if err != nil {
			die("list: %v", err)
		}
		// Route through the service layer (rather than calling storeDest.Search
		// directly) so this surface gets the same defaulting/validation as
		// search/nil_search — a consistency fix that predates this change but
		// is a no-op for existing behavior here (every field below is already
		// explicitly set to what the service would default to anyway).
		results, err := svc.Search(ctx, storeDest, req)
		if err != nil {
			die("list: %v", err)
		}
		items = results
	}
	printJSON(envelope{OK: true, Data: map[string]any{
		"command": "list",
		"view":    view,
		"count":   len(items),
		"items":   items,
	}})
}

func cmdShow(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	vaultID := fs.String("vault", "", "vault id or 'inbox'")
	format := fs.String("format", "detail", "detail|json")
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("show: %v", err)
	}
	if len(positional) == 0 {
		die("show: requires <id>")
	}
	var id int64
	if _, err := fmt.Sscan(positional[0], &id); err != nil {
		die("show: invalid id %q", positional[0])
	}
	loc, err := findItem(ctx, env.mgr, id, *vaultID)
	if err != nil {
		die("show: %v", err)
	}
	data := map[string]any{
		"command": "show",
		"id":      id,
		"vault":   loc.location,
		"item":    loc.item,
	}
	if strings.ToLower(*format) == "detail" {
		printJSON(envelope{OK: true, Data: data})
		return
	}
	printJSON(envelope{OK: true, Data: loc.item})
}

func cmdContext(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("context", flag.ContinueOnError)
	topicFlag := fs.String("topic", "app", "topic: app|schema|stats|cache|refresh")
	refreshFlag := fs.Bool("refresh", false, "refresh cached snapshot before responding")
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("context: %v", err)
	}
	topic := *topicFlag
	if len(positional) > 0 {
		topic = positional[0]
	}
	topic = strings.ToLower(topic)
	if topic == "refresh" {
		*refreshFlag = true
		topic = "cache"
	}

	var cachedSnap *contextcache.Snapshot
	loadSnapshot := func(force bool) *contextcache.Snapshot {
		if !force && cachedSnap != nil {
			return cachedSnap
		}
		var snap *contextcache.Snapshot
		var loadErr error
		if !force {
			snap, loadErr = contextcache.Load()
			if loadErr != nil {
				snap, loadErr = contextcache.Refresh(ctx, env.cfg, env.mgr)
			}
		} else {
			snap, loadErr = contextcache.Refresh(ctx, env.cfg, env.mgr)
		}
		if loadErr != nil {
			die("context: %v", loadErr)
		}
		cachedSnap = snap
		return cachedSnap
	}

	switch topic {
	case "schema":
		snap := loadSnapshot(*refreshFlag)
		printJSON(envelope{OK: true, Data: map[string]any{
			"topic":         "schema",
			"schema":        snap.Schema,
			"referenceDocs": snap.App.Docs,
		}})
	case "stats":
		snap := loadSnapshot(*refreshFlag)
		printJSON(envelope{OK: true, Data: map[string]any{
			"topic": "stats",
			"stats": snap.Stats,
			"inbox": snap.Inbox,
		}})
	case "cache":
		snap := loadSnapshot(*refreshFlag)
		printJSON(envelope{OK: true, Data: map[string]any{
			"topic":    "cache",
			"snapshot": snap,
		}})
	default:
		snap := loadSnapshot(*refreshFlag)
		cmds := make([]string, 0, len(commandRegistry))
		for _, c := range commandRegistry {
			cmds = append(cmds, c.name)
		}
		printJSON(envelope{OK: true, Data: map[string]any{
			"topic":    "app",
			"app":      snap.App,
			"inbox":    snap.Inbox,
			"vaults":   snap.Vaults,
			"commands": cmds,
		}})
	}
}

func cmdImport(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	dir := fs.String("dir", "", "directory to import (required)")
	dryRun := fs.Bool("dry-run", false, "scan without writing")
	batchSize := fs.Int("batch-size", 100, "files per logical batch (informational)")
	kindFlag := fs.String("kind", "note", "kind for imported files: todo|note|scratch")
	formatFlag := fs.String("format", "md", "body format for imported files: md|html|json")
	vaultID := fs.String("vault", "", "target vault id")
	inbox := fs.Bool("inbox", true, "route items to inbox (default)")
	tags := fs.String("tags", "", "tags to apply to every imported item")
	contexts := fs.String("contexts", "", "contexts to apply")
	projects := fs.String("projects", "", "projects to apply")
	var metaPairs stringSliceFlag
	fs.Var(&metaPairs, "meta", "metadata key=value (stored as meta:<key>=<value> tags)")
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("import: %v", err)
	}
	if *dir == "" && len(positional) > 0 {
		*dir = positional[0]
	}
	if *dir == "" {
		die("import: --dir is required")
	}
	info, err := os.Stat(*dir)
	if err != nil {
		die("import: %v", err)
	}
	if !info.IsDir() {
		die("import: %s is not a directory", *dir)
	}
	storeDest, toInbox, err := chooseCreateDestination(env.mgr, *vaultID, *inbox)
	if err != nil {
		die("import: %v", err)
	}
	baseTags := splitCSV(*tags)
	baseContexts := splitCSV(*contexts)
	baseProjects := splitCSV(*projects)
	metaTags := appendMetaTags(nil, parseKeyValuePairs(metaPairs))
	var processed []map[string]any
	var failed []map[string]string
	err = filepath.WalkDir(*dir, func(path string, d iofs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		body, readErr := readTextFile(path)
		if readErr != nil {
			failed = append(failed, map[string]string{"path": path, "error": readErr.Error()})
			return nil
		}
		rel, _ := filepath.Rel(*dir, path)
		title := strings.TrimSuffix(rel, filepath.Ext(rel))
		doc, md, htmlBody, ierr := svc.NotesInputFromCLI(body, *formatFlag)
		if ierr != nil {
			failed = append(failed, map[string]string{"path": path, "error": ierr.Error()})
			return nil
		}
		input := items.CreateInput{
			Title:     title,
			Kind:      *kindFlag,
			APISource: "cli-import",
			Tags:      append(append([]string{}, baseTags...), metaTags...),
			Contexts:  append([]string{}, baseContexts...),
			Projects:  append([]string{}, baseProjects...),
			Inbox:     toInbox,
			NotesDoc:  doc,
			NotesMD:   md,
			NotesHTML: htmlBody,
		}
		if *dryRun {
			processed = append(processed, map[string]any{"path": path, "title": input.Title, "dryRun": true})
			return nil
		}
		created, createErr := svc.Create(ctx, storeDest, input)
		if createErr != nil {
			failed = append(failed, map[string]string{"path": path, "error": createErr.Error()})
			return nil
		}
		processed = append(processed, map[string]any{"path": path, "id": created.ID, "title": created.Title})
		return nil
	})
	if err != nil {
		die("import: %v", err)
	}
	printJSON(envelope{OK: len(failed) == 0, Data: map[string]any{
		"command":   "import",
		"dir":       *dir,
		"dryRun":    *dryRun,
		"processed": processed,
		"failed":    failed,
		"metaTags":  metaTags,
		"batchSize": *batchSize,
	}})
}

func cmdUpdate(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	idsArg := fs.String("ids", "", "comma-separated item IDs")
	vaultID := fs.String("vault", "", "limit updates to this vault")
	inbox := fs.Bool("inbox", false, "limit updates to inbox")
	complete := fs.Bool("complete", false, "mark items complete")
	uncomplete := fs.Bool("uncomplete", false, "mark items incomplete")
	archive := fs.Bool("archive", false, "archive items")
	unarchive := fs.Bool("unarchive", false, "unarchive items")
	deleteFlag := fs.Bool("delete", false, "delete items")
	priority := fs.String("priority", "", "set priority A|B|C, or NONE to clear")
	tagsAdd := fs.String("tags-add", "", "comma-separated tags to add")
	tagsRemove := fs.String("tags-remove", "", "comma-separated tags to remove")
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("update: %v", err)
	}
	if *idsArg == "" && len(positional) > 0 {
		*idsArg = positional[0]
	}
	ids, err := parseIDsArg(*idsArg)
	if err != nil {
		die("update: %v", err)
	}
	if len(ids) == 0 {
		die("update: provide --ids or positional ids")
	}
	if !*complete && !*uncomplete && !*archive && !*unarchive && !*deleteFlag && *priority == "" && *tagsAdd == "" && *tagsRemove == "" {
		die("update: specify at least one mutation flag")
	}
	var scope string
	if *inbox {
		scope = "inbox"
	} else if *vaultID != "" {
		scope = *vaultID
	}
	addTags := splitCSV(*tagsAdd)
	removeTags := splitCSV(*tagsRemove)
	var updated []map[string]any
	var failed []map[string]any
	for _, id := range ids {
		loc, findErr := findItem(ctx, env.mgr, id, scope)
		if findErr != nil {
			failed = append(failed, map[string]any{"id": id, "error": findErr.Error()})
			continue
		}
		if *deleteFlag {
			if err := loc.store.DeleteItem(ctx, id); err != nil {
				failed = append(failed, map[string]any{"id": id, "error": err.Error()})
				continue
			}
			updated = append(updated, map[string]any{"id": id, "action": "deleted"})
			continue
		}
		item := loc.item
		if *complete {
			item.Completed = true
		}
		if *uncomplete {
			item.Completed = false
		}
		if *archive {
			item.Archived = true
		}
		if *unarchive {
			item.Archived = false
		}
		if strings.EqualFold(*priority, "none") {
			item.Priority = nil
		} else if *priority != "" {
			p := strings.ToUpper(*priority)
			item.Priority = &p
		}
		item.Tags = applyTagMutations(item.Tags, addTags, removeTags)
		if err := svc.Update(ctx, loc.store, item); err != nil {
			failed = append(failed, map[string]any{"id": id, "error": err.Error()})
			continue
		}
		updated = append(updated, map[string]any{"id": id, "action": "updated"})
	}
	printJSON(envelope{OK: len(failed) == 0, Data: map[string]any{
		"command": "update",
		"updated": updated,
		"failed":  failed,
	}})
}

func parseIDsArg(arg string) ([]int64, error) {
	if strings.TrimSpace(arg) == "" {
		return nil, nil
	}
	parts := strings.FieldsFunc(arg, func(r rune) bool { return r == ',' || r == ' ' })
	var ids []int64
	for _, part := range parts {
		if part == "" {
			continue
		}
		var id int64
		if _, err := fmt.Sscan(part, &id); err != nil {
			return nil, fmt.Errorf("invalid id %q", part)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func applyTagMutations(existing []string, add, remove []string) []string {
	set := map[string]bool{}
	for _, tag := range existing {
		set[tag] = true
	}
	for _, tag := range add {
		if tag == "" {
			continue
		}
		set[tag] = true
	}
	for _, tag := range remove {
		set[tag] = false
	}
	var out []string
	for tag, include := range set {
		if include {
			out = append(out, tag)
		}
	}
	slices.Sort(out)
	return out
}

func chooseCreateDestination(mgr *vault.Manager, vaultID string, toInbox bool) (*store.Store, bool, error) {
	if toInbox || vaultID == "" {
		if inbox := mgr.InboxStore(); inbox != nil {
			return inbox, true, nil
		}
	}
	if vaultID == "" {
		if active := mgr.ActiveStore(); active != nil {
			return active, false, nil
		}
		if inbox := mgr.InboxStore(); inbox != nil {
			return inbox, true, nil
		}
		return nil, false, fmt.Errorf("no active vault or inbox available")
	}
	s, err := mgr.StoreForID(vaultID)
	if err != nil {
		return nil, false, err
	}
	return s, false, nil
}

func selectVaultStore(mgr *vault.Manager, vaultID string) (*store.Store, error) {
	if vaultID != "" {
		return mgr.StoreForID(vaultID)
	}
	if active := mgr.ActiveStore(); active != nil {
		return active, nil
	}
	return nil, fmt.Errorf("no active vault; use --vault to select one")
}

type itemLocation struct {
	item     *store.Item
	store    *store.Store
	location string
}

func findItem(ctx context.Context, mgr *vault.Manager, id int64, hint string) (*itemLocation, error) {
	if hint == "inbox" {
		inbox := mgr.InboxStore()
		if inbox == nil {
			return nil, fmt.Errorf("inbox store not available")
		}
		item, err := inbox.GetItem(ctx, id)
		if err != nil {
			return nil, err
		}
		return &itemLocation{item: item, store: inbox, location: "inbox"}, nil
	}
	if hint != "" {
		s, err := mgr.StoreForID(hint)
		if err != nil {
			return nil, err
		}
		item, err := s.GetItem(ctx, id)
		if err != nil {
			return nil, err
		}
		return &itemLocation{item: item, store: s, location: hint}, nil
	}
	if active := mgr.ActiveStore(); active != nil {
		if item, err := active.GetItem(ctx, id); err == nil {
			return &itemLocation{item: item, store: active, location: mgr.GetActiveVaultID()}, nil
		}
	}
	if inbox := mgr.InboxStore(); inbox != nil {
		if item, err := inbox.GetItem(ctx, id); err == nil {
			return &itemLocation{item: item, store: inbox, location: "inbox"}, nil
		}
	}
	return nil, fmt.Errorf("item %d not found in active vault or inbox", id)
}

func readTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) || bytes.ContainsRune(data, '\x00') {
		return "", fmt.Errorf("%s must be UTF-8 text or markdown", path)
	}
	return string(data), nil
}

func appendMetaTags(existing []string, meta map[string]string) []string {
	if len(meta) == 0 {
		return existing
	}
	out := append([]string{}, existing...)
	for k, v := range meta {
		if k == "" {
			continue
		}
		tag := fmt.Sprintf("meta:%s=%s", k, v)
		out = append(out, tag)
	}
	return out
}

func parseKeyValuePairs(pairs []string) map[string]string {
	if len(pairs) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		result[key] = value
	}
	return result
}

// Run dispatches os.Args[1:] to the appropriate subcommand.
func Run(args []string, cfg *config.Config, mgr *vault.Manager) {
	ctx := context.Background()
	env := &commandEnv{cfg: cfg, mgr: mgr}
	if len(args) == 0 {
		cmdHelp(ctx, nil, env)
		os.Exit(1)
	}
	name := args[0]
	if name == "help" || name == "--help" || name == "-h" {
		cmdHelp(ctx, args[1:], env)
		return
	}
	if name == "--version" || name == "-v" {
		cmdVersion(ctx, nil, env)
		return
	}
	cmd := findCommand(name)
	if cmd == nil {
		die("unknown command %q — run 'nil help' for a list", name)
	}
	cmd.run(ctx, args[1:], env)
}

func findCommand(name string) *command {
	for i := range commandRegistry {
		cmd := &commandRegistry[i]
		if cmd.name == name {
			return cmd
		}
		for _, alias := range cmd.aliases {
			if alias == name {
				return cmd
			}
		}
	}
	return nil
}

// cmdPush creates an item and routes it to the inbox (default) or a vault.
func cmdPush(ctx context.Context, args []string, mgr *vault.Manager) {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	var (
		source     = fs.String("source", "cli", "source label stored in api_source")
		vaultID    = fs.String("vault", "", "vault ID to push to (omit for inbox)")
		notes      = fs.String("notes", "", "notes body (markdown by default; use --notes-format)")
		notesFmt   = fs.String("notes-format", "md", "notes body format: md|html|json")
		kindFlag   = fs.String("kind", "todo", "item kind: todo|note|scratch")
		priority   = fs.String("priority", "", "priority letter: A|B|C")
		tags       = fs.String("tags", "", "comma-separated tags")
		contexts   = fs.String("contexts", "", "comma-separated contexts")
		projects   = fs.String("projects", "", "comma-separated projects")
		due        = fs.String("due", "", "due date YYYY-MM-DD")
		toInbox    = fs.Bool("inbox", false, "force route to inbox (default when no --vault)")
	)
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("push: %v", err)
	}

	title := strings.Join(positional, " ")

	doc, md, htmlBody, err := svc.NotesInputFromCLI(*notes, *notesFmt)
	if err != nil {
		die("push: parse notes: %v", err)
	}

	input := items.CreateInput{
		Title:     title,
		Kind:      *kindFlag,
		APISource: *source,
		Tags:      splitCSV(*tags),
		Contexts:  splitCSV(*contexts),
		Projects:  splitCSV(*projects),
		NotesDoc:  doc,
		NotesMD:   md,
		NotesHTML: htmlBody,
	}
	if *priority != "" {
		p := strings.ToUpper(*priority)
		input.Priority = &p
	}
	if *due != "" {
		d := *due
		input.DueAt = &d
	}

	// Routing: --vault <id> sends to a specific vault; otherwise goes to inbox.
	var (
		storeDest *store.Store
		result    *store.Item
		rerr      error
	)
	if *vaultID != "" && !*toInbox {
		s, serr := mgr.StoreForID(*vaultID)
		if serr != nil {
			die("push: vault %q: %v", *vaultID, serr)
		}
		storeDest = s
	} else {
		input.Inbox = true
		storeDest = mgr.InboxStore()
		if storeDest == nil {
			die("push: inbox store not available")
		}
	}
	result, rerr = svc.Create(ctx, storeDest, input)
	if rerr != nil {
		die("push: %v", rerr)
	}
	printJSON(envelope{OK: true, Data: result})
}

// cmdSearch searches items in the active (or specified) vault.
func cmdSearch(ctx context.Context, args []string, mgr *vault.Manager) {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	var (
		vaultID      = fs.String("vault", "", "vault ID to search (omit for active vault)")
		kindFlag     = fs.String("kind", "all", "filter by kind: todo|note|scratch|all")
		tags         = fs.String("tags", "", "comma-separated tags")
		contexts     = fs.String("contexts", "", "comma-separated contexts")
		projects     = fs.String("projects", "", "comma-separated projects")
		page         = fs.Int("page", 0, "zero-based page index")
		pageSize     = fs.Int("page-size", 50, "results per page")
		updatedSince = fs.String("updated-since", "", "only items updated on/after this RFC3339 timestamp (e.g. 2026-08-01T00:00:00Z)")
	)
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("search: %v", err)
	}

	query := strings.Join(positional, " ")

	req := store.SearchRequest{
		Query:        query,
		Tags:         splitCSV(*tags),
		Contexts:     splitCSV(*contexts),
		Projects:     splitCSV(*projects),
		Page:         *page,
		PageSize:     *pageSize,
		Kind:         *kindFlag,
		UpdatedSince: *updatedSince,
	}

	var s *store.Store
	if *vaultID != "" {
		var err error
		s, err = mgr.StoreForID(*vaultID)
		if err != nil {
			die("search: vault %q: %v", *vaultID, err)
		}
	} else {
		s = mgr.ActiveStore()
	}
	if s == nil {
		die("search: no active store")
	}

	results, err := svc.Search(ctx, s, req)
	if err != nil {
		die("search: %v", err)
	}
	if results == nil {
		results = []store.Item{}
	}
	printJSON(envelope{OK: true, Data: results})
}

// cmdInbox lists items in the shared inbox store.
func cmdInbox(ctx context.Context, args []string, mgr *vault.Manager) {
	fs := flag.NewFlagSet("inbox", flag.ContinueOnError)
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("inbox: %v", err)
	}

	query := strings.Join(positional, " ")

	inboxStore := mgr.InboxStore()
	if inboxStore == nil {
		die("inbox: inbox store not available")
	}

	req := store.SearchRequest{Query: query}
	results, err := svc.ListInbox(ctx, inboxStore, req)
	if err != nil {
		die("inbox: %v", err)
	}
	if results == nil {
		results = []store.Item{}
	}
	printJSON(envelope{OK: true, Data: results})
}

// cmdGet fetches a single item by ID.
func cmdGet(ctx context.Context, args []string, mgr *vault.Manager) {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	var vaultID = fs.String("vault", "", "vault ID (use 'inbox' for inbox store)")
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("get: %v", err)
	}
	if len(positional) == 0 {
		die("get: requires <id>")
	}

	var id int64
	if _, err := fmt.Sscan(positional[0], &id); err != nil {
		die("get: invalid id %q: %v", positional[0], err)
	}

	var (
		item *store.Item
		gerr error
	)

	switch {
	case *vaultID == "inbox":
		inboxStore := mgr.InboxStore()
		if inboxStore == nil {
			die("get: inbox store not available")
		}
		item, gerr = inboxStore.GetItem(ctx, id)
	case *vaultID != "":
		s, serr := mgr.StoreForID(*vaultID)
		if serr != nil {
			die("get: vault %q: %v", *vaultID, serr)
		}
		item, gerr = s.GetItem(ctx, id)
	default:
		// Try active vault first, then inbox.
		if s := mgr.ActiveStore(); s != nil {
			item, gerr = s.GetItem(ctx, id)
		}
		if gerr != nil || item == nil {
			if s := mgr.InboxStore(); s != nil {
				item, gerr = s.GetItem(ctx, id)
			}
		}
	}

	if gerr != nil || item == nil {
		die("get: item %d not found", id)
	}
	printJSON(envelope{OK: true, Data: item})
}

// vaultsData is the shape returned by the vaults command.
type vaultsData struct {
	Active string         `json:"active"`
	Vaults []config.Vault `json:"vaults"`
}

// cmdVaults lists all registered vaults.
func cmdVaults(cfg *config.Config, mgr *vault.Manager) {
	printJSON(envelope{
		OK: true,
		Data: vaultsData{
			Active: cfg.ActiveVaultID,
			Vaults: mgr.GetVaults(),
		},
	})
}

// printJSON marshals v as indented JSON and writes it to stdout.
func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nil: json error:", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}

// die writes a formatted message to stderr and exits with code 1.
func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "nil: "+format+"\n", args...)
	os.Exit(1)
}

// parseInterspersed calls fs.Parse in a loop so that flags and positional
// arguments may be freely intermixed (e.g. nil push "title" --source cli).
// Returns the collected positional arguments.
func parseInterspersed(args []string, fs *flag.FlagSet) ([]string, error) {
	var positional []string
	for len(args) > 0 {
		if args[0] == "--" {
			positional = append(positional, args[1:]...)
			return positional, nil
		}
		if !strings.HasPrefix(args[0], "-") {
			positional = append(positional, args[0])
			args = args[1:]
			continue
		}
		// Flags from here — parse until the next positional.
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
	}
	return positional, nil
}

// splitCSV splits a comma-separated string into a slice, trimming whitespace.
// Returns nil for an empty string.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
