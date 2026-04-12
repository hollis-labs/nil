# NIL CLI Quick Reference

This guide is written for humans who want to drive NIL from a shell. Every command prints JSON so agents can consume it too, but the verbs and examples below map to the most common workflows.

## Quick start

```bash
# Show top-level help (lists all commands)
nil help

# Add a todo directly into the inbox
nil add "Plan launch" --body "Outline marketing tasks" --type todo --tags launch,marketing --priority A

# Import every Markdown file from a directory as notes (dry run first)
nil import --dir ~/Notes --dry-run
```

## Core commands

| Command | When to use it | Handy flags |
|---------|----------------|-------------|
| `nil add [title]` | Fast capture from the terminal. Accepts inline bodies or `--file` input. | `--file`, `--body`, `--type todo|note`, `--vault`, `--inbox`, `--tags`, `--contexts`, `--projects`, `--due`, `--priority`, `--meta key=value`. |
| `nil list [view]` | Review the latest items without opening the GUI. Views: `latest`, `inbox`, `today`, `overdue`, `high`, `notes`, `todos`. | `--view`, `--limit`, `--vault`, `--query`, `--include-inbox`. |
| `nil show <id>` | Inspect a single item (like `nil show 42`). | `--vault id|inbox`, `--format detail|json`. |
| `nil update --ids ...` | Batch complete, archive, delete, or retag items. | `--ids`, `--complete`, `--archive`, `--delete`, `--priority`, `--tags-add`, `--tags-remove`. |
| `nil import --dir PATH` | Ingest entire directories of text/Markdown files (notes, exported todos, etc.). | `--dry-run`, `--type`, `--vault`, `--inbox`, `--tags`, `--contexts`, `--projects`, `--meta`. |
| `nil search <keywords>` | Fuzzy search over the active vault (full text + filters). | `--vault`, `--type`, `--tags`, `--contexts`, `--projects`, `--page`, `--page-size`. |
| `nil inbox [keywords]` | Quickly review inbox-only items. | `--page`, `--page-size`. |
| `nil context [topic]` | Pull context snapshots for agents or personal reference. Topics: `app`, `schema`, `stats`, `cache`. | `--refresh` forces a rebuild of `${config}/context-cache/snapshot.json`. |

Legacy verbs (`nil push`, `nil get`) still work for backwards compatibility, but the new `add`/`show` commands should be used going forward.

## Practical examples

```bash
# Capture a Markdown note stored in a file
nil add "Reading notes" --file ~/Desktop/reading.md --type note --vault myvault

# List overdue tasks (limit to 10) in the active vault
nil list overdue --limit 10

# Complete items 12,15,18 and drop the #someday tag
nil update --ids 12,15,18 --complete --tags-remove someday

# Refresh and print the context snapshot (great for agents)
nil context cache --refresh | jq '.'
```

All commands run against your local vaults via the embedded Go runtime—no server process required.
