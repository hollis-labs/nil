// Package cli implements `nil <command> [args]`. main.go dispatches any
// bare (non-flag) first argument to Run before wails.Run ever executes.
//
// This file is the command-dispatch spine: the command/commandEnv types,
// the commandRegistry table (built in init()), Run/findCommand, and the
// two smallest built-in commands (help, version). Individual command
// implementations live alongside their dispatch idiom:
//   - commands_item.go: the newer commandEnv-native commands (add,
//     add-batch, list, show, backrefs, ids, context, import, update)
//   - commands_legacy.go: older commands wired via *vault.Manager-only
//     closures (push/legacy-add, search, inbox, get, vaults)
//   - commands_serve.go: serve-api, which pulls in apiserver as a
//     dependency
//   - helpers.go: shared helpers used across all of the above (flag
//     parsing, vault/item resolution, response envelope printing)
package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/hollis-labs/nil/config"
	"github.com/hollis-labs/nil/service/items"
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
			name:  "add-batch",
			short: "Batch-create todos/notes from a JSON array (file or stdin)",
			usage: "nil add-batch [--file path|-] [--vault id] [--inbox] [--source label]",
			run:   cmdAddBatch,
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
			name:  "backrefs",
			short: "List items that link to a given item (wikilink backlinks)",
			usage: "nil backrefs <id> [--vault id|inbox]",
			run:   cmdBackrefs,
		},
		{
			name:  "ids",
			short: "List every current item's id + updated_at (deletion/change signal for sync consumers)",
			usage: "nil ids [--vault id|inbox] [--kind todo|note|scratch|all]",
			run:   cmdIDs,
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
