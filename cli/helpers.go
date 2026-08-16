package cli

// helpers.go holds shared helpers used by every command implementation in
// this package, regardless of which dispatch idiom (commandEnv-native vs.
// legacy *vault.Manager-only) the caller uses: flag parsing
// (parseInterspersed), vault/item resolution (chooseCreateDestination,
// selectVaultStore, findItem), CSV/ID/meta parsing, and response
// printing/exit helpers (printJSON, die).

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/hollis-labs/nil/store"
	"github.com/hollis-labs/nil/vault"
)

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
	data, err := os.ReadFile(path) //nolint:gosec // CLI tool: path is an operator-supplied --file argument, not attacker-controlled input
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
