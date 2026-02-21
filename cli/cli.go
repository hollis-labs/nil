package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"nanite/config"
	"nanite/store"
	"nanite/vault"
)

// envelope wraps all CLI output.
type envelope struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// Run dispatches os.Args[1:] to the appropriate subcommand.
func Run(args []string, cfg *config.Config, mgr *vault.Manager) {
	if len(args) == 0 {
		die("usage: nanite <push|search|inbox|get|vaults> [flags]")
	}
	ctx := context.Background()
	switch args[0] {
	case "push":
		cmdPush(ctx, args[1:], mgr)
	case "search":
		cmdSearch(ctx, args[1:], mgr)
	case "inbox":
		cmdInbox(ctx, args[1:], mgr)
	case "get":
		cmdGet(ctx, args[1:], mgr)
	case "vaults":
		cmdVaults(cfg, mgr)
	default:
		die("unknown command %q — try push, search, inbox, get, or vaults", args[0])
	}
}

// cmdPush creates an item and routes it to the inbox (default) or a vault.
func cmdPush(ctx context.Context, args []string, mgr *vault.Manager) {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	var (
		source   = fs.String("source", "cli", "source label stored in api_source")
		vaultID  = fs.String("vault", "", "vault ID to push to (omit for inbox)")
		notes    = fs.String("notes", "", "notes_md HTML content")
		itemType = fs.String("type", "todo", "item type: todo|note")
		priority = fs.String("priority", "", "priority letter: A|B|C")
		tags     = fs.String("tags", "", "comma-separated tags")
		contexts = fs.String("contexts", "", "comma-separated contexts")
		projects = fs.String("projects", "", "comma-separated projects")
		due      = fs.String("due", "", "due date YYYY-MM-DD")
		toInbox  = fs.Bool("inbox", false, "force route to inbox (default when no --vault)")
	)
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("push: %v", err)
	}

	title := strings.Join(positional, " ")

	item := &store.Item{
		Title:     title,
		NotesMD:   *notes,
		Type:      *itemType,
		APISource: *source,
		Tags:      splitCSV(*tags),
		Contexts:  splitCSV(*contexts),
		Projects:  splitCSV(*projects),
	}
	if *priority != "" {
		p := strings.ToUpper(*priority)
		item.Priority = &p
	}
	if *due != "" {
		d := *due
		item.DueAt = &d
	}

	// Routing: --vault <id> sends to a specific vault; otherwise goes to inbox.
	var (
		result *store.Item
		rerr   error
	)
	if *vaultID != "" && !*toInbox {
		s, serr := mgr.StoreForID(*vaultID)
		if serr != nil {
			die("push: vault %q: %v", *vaultID, serr)
		}
		result, rerr = s.CreateItem(ctx, item)
	} else {
		item.Inbox = true
		inboxStore := mgr.InboxStore()
		if inboxStore == nil {
			die("push: inbox store not available")
		}
		result, rerr = inboxStore.CreateItem(ctx, item)
	}
	if rerr != nil {
		die("push: %v", rerr)
	}
	printJSON(envelope{OK: true, Data: result})
}

// cmdSearch searches items in the active (or specified) vault.
func cmdSearch(ctx context.Context, args []string, mgr *vault.Manager) {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	var (
		vaultID  = fs.String("vault", "", "vault ID to search (omit for active vault)")
		itemType = fs.String("type", "all", "filter by type: todo|note|all")
		tags     = fs.String("tags", "", "comma-separated tags")
		contexts = fs.String("contexts", "", "comma-separated contexts")
		projects = fs.String("projects", "", "comma-separated projects")
		page     = fs.Int("page", 0, "zero-based page index")
		pageSize = fs.Int("page-size", 50, "results per page")
	)
	positional, err := parseInterspersed(args, fs)
	if err != nil {
		die("search: %v", err)
	}

	query := strings.Join(positional, " ")

	t := *itemType
	if t == "all" {
		t = ""
	}
	req := store.SearchRequest{
		Query:    query,
		Tags:     splitCSV(*tags),
		Contexts: splitCSV(*contexts),
		Projects: splitCSV(*projects),
		Page:     *page,
		PageSize: *pageSize,
		Type:     t,
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

	results, err := s.Search(ctx, req)
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

	req := store.SearchRequest{
		Query:    query,
		PageSize: 200,
	}
	results, err := inboxStore.GetInboxItems(ctx, req)
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
		fmt.Fprintln(os.Stderr, "nanite: json error:", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}

// die writes a formatted message to stderr and exits with code 1.
func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "nanite: "+format+"\n", args...)
	os.Exit(1)
}

// parseInterspersed calls fs.Parse in a loop so that flags and positional
// arguments may be freely intermixed (e.g. nanite push "title" --source cli).
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
