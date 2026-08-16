package cli

// commands_item.go holds the newer commandEnv-native command
// implementations — those registered in commandRegistry (cli.go) with a
// func(context.Context, []string, *commandEnv) signature directly, rather
// than wired in via a closure over the legacy *vault.Manager-only
// functions in commands_legacy.go.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hollis-labs/nil/contextcache"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
)

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
	externalRef := fs.String("external-ref", "", "idempotency key for a corresponding record on an external system; re-running add with the same value (in the same vault) updates the existing item instead of creating a duplicate")
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
		Title:       title,
		Kind:        *kindFlag,
		APISource:   *source,
		Tags:        appendMetaTags(splitCSV(*tags), parseKeyValuePairs(metaPairs)),
		Contexts:    splitCSV(*contexts),
		Projects:    splitCSV(*projects),
		Inbox:       toInbox,
		ExternalRef: *externalRef,
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

// cliBatchItem is one entry of the JSON array cmdAddBatch reads from a file
// or stdin. It mirrors the HTTP API's per-item batch-create shape
// (apiserver.itemCreateRequest) field-for-field, so a payload built for one
// surface works unmodified against the other. Unlike cmdAdd's flags (which
// can only describe one item per invocation), a batch naturally needs each
// item to carry its own title/kind/tags/etc., which flags can't express —
// hence a JSON array input rather than repeatable flags.
type cliBatchItem struct {
	Title       string   `json:"title"`
	Kind        string   `json:"kind"`
	Section     string   `json:"section"`
	Pinned      bool     `json:"pinned"`
	Priority    *string  `json:"priority"`
	DueAt       *string  `json:"due_at"`
	Tags        []string `json:"tags"`
	Contexts    []string `json:"contexts"`
	Projects    []string `json:"projects"`
	ExternalRef string   `json:"external_ref"`
	NotesDoc    string   `json:"notes_doc"`
	NotesMD     string   `json:"notes_md"`
	NotesHTML   string   `json:"notes_html"`
}

// cmdAddBatch creates multiple items from a JSON array in one call,
// all-or-nothing (see items.Service.CreateBatch / store.CreateItemsBatch):
// if any item fails, nothing is created and the error names which item (by
// index) and why. --file reads from a path; "-" (the default) reads from
// stdin, so callers can pipe generated JSON straight in
// (e.g. `some-generator | nil add-batch`) without a temp file. --vault /
// --inbox route the whole batch to one destination the same way they do for
// `nil add`; there's no per-item destination override — a caller that needs
// items split across vaults should make one add-batch call per vault.
func cmdAddBatch(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("add-batch", flag.ContinueOnError)
	file := fs.String("file", "-", "path to a JSON file containing an array of items, or '-' to read from stdin (default)")
	vaultID := fs.String("vault", "", "vault ID to store the items")
	inbox := fs.Bool("inbox", false, "route all items to inbox")
	source := fs.String("source", "cli", "source label stored in api_source for every item")
	if _, err := parseInterspersed(args, fs); err != nil {
		die("add-batch: %v", err)
	}

	var raw []byte
	var err error
	if *file == "-" || *file == "" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(*file)
	}
	if err != nil {
		die("add-batch: reading input: %v", err)
	}

	var batchItems []cliBatchItem
	if err = json.Unmarshal(raw, &batchItems); err != nil {
		die("add-batch: parsing JSON array: %v", err)
	}
	if len(batchItems) == 0 {
		die("add-batch: input must be a non-empty JSON array of items")
	}
	for i, it := range batchItems {
		if strings.TrimSpace(it.Title) == "" {
			die("add-batch: item %d: title is required", i)
		}
	}

	storeDest, toInbox, err := chooseCreateDestination(env.mgr, *vaultID, *inbox)
	if err != nil {
		die("add-batch: %v", err)
	}

	inputs := make([]items.CreateInput, len(batchItems))
	for i, it := range batchItems {
		inputs[i] = items.CreateInput{
			Title:       it.Title,
			Kind:        it.Kind,
			Section:     it.Section,
			Pinned:      it.Pinned,
			Priority:    it.Priority,
			DueAt:       it.DueAt,
			Tags:        it.Tags,
			Contexts:    it.Contexts,
			Projects:    it.Projects,
			APISource:   *source,
			ExternalRef: it.ExternalRef,
			Inbox:       toInbox,
			NotesDoc:    it.NotesDoc,
			NotesMD:     it.NotesMD,
			NotesHTML:   it.NotesHTML,
		}
	}

	created, err := svc.CreateBatch(ctx, storeDest, inputs)
	if err != nil {
		die("add-batch: %v", err)
	}
	printJSON(envelope{OK: true, Data: map[string]any{
		"command": "add-batch",
		"count":   len(created),
		"items":   svc.WithTextSlice(created),
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
		"items":   svc.WithTextSlice(items),
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
	if _, scanErr := fmt.Sscan(positional[0], &id); scanErr != nil {
		die("show: invalid id %q", positional[0])
	}
	loc, err := findItem(ctx, env.mgr, id, *vaultID)
	if err != nil {
		die("show: %v", err)
	}
	itemView := svc.WithText(loc.item)
	data := map[string]any{
		"command": "show",
		"id":      id,
		"vault":   loc.location,
		"item":    itemView,
	}
	if strings.ToLower(*format) == "detail" {
		printJSON(envelope{OK: true, Data: data})
		return
	}
	printJSON(envelope{OK: true, Data: itemView})
}

// cmdBackrefs lists items that link to <id> via a wikilink (backlinks /
// "what links here"). It is a dedicated command rather than a flag on
// show/get: backrefs returns a different shape (a list of items) than show's
// single-item response, and the CLI already has a one-command-per-read-op
// pattern (show, get, search, inbox, vaults) rather than overloading one
// command's flags to change its output shape. Mirrors GetBackrefs' existing
// GUI contract (app.go's App.GetBackrefs); notes_text is layered on top the
// same way `show`/`get` already add it via svc.WithText.
func cmdBackrefs(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("backrefs", flag.ContinueOnError)
	vaultID := fs.String("vault", "", "vault id or 'inbox'")
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("backrefs: %v", err)
	}
	if len(positional) == 0 {
		die("backrefs: requires <id>")
	}
	var id int64
	if _, scanErr := fmt.Sscan(positional[0], &id); scanErr != nil {
		die("backrefs: invalid id %q", positional[0])
	}
	loc, err := findItem(ctx, env.mgr, id, *vaultID)
	if err != nil {
		die("backrefs: %v", err)
	}
	backrefs, err := loc.store.GetBackrefs(ctx, id)
	if err != nil {
		die("backrefs: %v", err)
	}
	printJSON(envelope{OK: true, Data: map[string]any{
		"command":  "backrefs",
		"id":       id,
		"vault":    loc.location,
		"backrefs": svc.WithTextSlice(backrefs),
	}})
}

// cmdIDs lists every current item's id + updated_at in the resolved vault —
// no title, no notes body, no taxonomy. This is the deletion/change-signal
// primitive: Nil has no soft-delete/tombstone concept (DeleteItem is a real,
// hard DELETE), so an external sync consumer has no other way to learn an
// item was removed. It fetches this full current-ID set on each sync cycle
// and diffs it against its own known-ID set; any previously-seen ID that's
// missing here has been genuinely deleted. See Store.ListItemIDs and the
// GET /api/v1/items/ids handler in apiserver.go for the full filter-support
// reasoning (kind defaults to "all"; archived/completed/inbox items are
// always included; updated_since is deliberately not supported — it would
// hide currently-existing IDs and produce false deletion signals). A
// dedicated command rather than a `show`/`search` flag, mirroring the
// backrefs command's one-command-per-read-op pattern above.
func cmdIDs(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("ids", flag.ContinueOnError)
	vaultID := fs.String("vault", "", "vault id or 'inbox' (omit for active vault)")
	kind := fs.String("kind", "all", "filter by kind: todo|note|scratch|all")
	if _, err := parseInterspersed(args, fs); err != nil {
		die("ids: %v", err)
	}

	var s *store.Store
	switch *vaultID {
	case "inbox":
		s = env.mgr.InboxStore()
	case "":
		s = env.mgr.ActiveStore()
	default:
		var err error
		s, err = env.mgr.StoreForID(*vaultID)
		if err != nil {
			die("ids: vault %q: %v", *vaultID, err)
		}
	}
	if s == nil {
		die("ids: vault not available")
	}

	ids, err := s.ListItemIDs(ctx, *kind)
	if err != nil {
		die("ids: %v", err)
	}
	printJSON(envelope{OK: true, Data: ids})
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
			return nil //nolint:nilerr // per-file failure: record it and keep walking the rest of the tree
		}
		rel, _ := filepath.Rel(*dir, path)
		title := strings.TrimSuffix(rel, filepath.Ext(rel))
		doc, md, htmlBody, ierr := svc.NotesInputFromCLI(body, *formatFlag)
		if ierr != nil {
			failed = append(failed, map[string]string{"path": path, "error": ierr.Error()})
			return nil //nolint:nilerr // per-file failure: record it and keep walking the rest of the tree
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
			return nil //nolint:nilerr // per-file failure: record it and keep walking the rest of the tree
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
