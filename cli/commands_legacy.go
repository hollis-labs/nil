package cli

// commands_legacy.go holds the older command implementations that predate
// the commandEnv-native registry idiom: bare functions taking a
// *vault.Manager (or *config.Config) directly rather than *commandEnv.
// They're wired into commandRegistry (cli.go) via small closures that
// unpack env.mgr/env.cfg and forward. push's alias is literally
// "legacy-add"; search/inbox/get/vaults are the other legacy
// implementations.

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/service/items"
	"github.com/hollis-labs/nil/store"
	"github.com/hollis-labs/nil/vault"
)

// cmdPush creates an item and routes it to the inbox (default) or a vault.
func cmdPush(ctx context.Context, args []string, mgr *vault.Manager) {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	var (
		source   = fs.String("source", "cli", "source label stored in api_source")
		vaultID  = fs.String("vault", "", "vault ID to push to (omit for inbox)")
		notes    = fs.String("notes", "", "notes body (markdown by default; use --notes-format)")
		notesFmt = fs.String("notes-format", "md", "notes body format: md|html|json")
		kindFlag = fs.String("kind", "todo", "item kind: todo|note|scratch")
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
		var serr error
		s, serr = mgr.StoreForID(*vaultID)
		if serr != nil {
			die("search: vault %q: %v", *vaultID, serr)
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
	printJSON(envelope{OK: true, Data: svc.WithTextSlice(results)})
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
	printJSON(envelope{OK: true, Data: svc.WithTextSlice(results)})
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
	printJSON(envelope{OK: true, Data: svc.WithText(item)})
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
