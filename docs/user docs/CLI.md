# NANITE CLI Quick Reference

This guide is written for humans who want to drive NANITE from a shell. Every command prints JSON so agents can consume it too, but the verbs and examples below map to the most common workflows.

## Quick start

```bash
# Show top-level help (lists all commands)
nanite help

# Add a todo directly into the inbox
nanite add "Plan launch" --body "Outline marketing tasks" --type todo --tags launch,marketing --priority A

# Import every Markdown file from a directory as notes (dry run first)
nanite import --dir ~/Notes --dry-run
```

## Core commands

| Command | When to use it | Handy flags |
|---------|----------------|-------------|
| `nanite add [title]` | Fast capture from the terminal. Accepts inline bodies or `--file` input. | `--file`, `--body`, `--type todo|note`, `--vault`, `--inbox`, `--tags`, `--contexts`, `--projects`, `--due`, `--priority`, `--meta key=value`. |
| `nanite list [view]` | Review the latest items without opening the GUI. Views: `latest`, `inbox`, `today`, `overdue`, `high`, `notes`, `todos`. | `--view`, `--limit`, `--vault`, `--query`, `--include-inbox`. |
| `nanite show <id>` | Inspect a single item (like `nanite show 42`). | `--vault id|inbox`, `--format detail|json`. |
| `nanite update --ids ...` | Batch complete, archive, delete, or retag items. | `--ids`, `--complete`, `--archive`, `--delete`, `--priority`, `--tags-add`, `--tags-remove`. |
| `nanite import --dir PATH` | Ingest entire directories of text/Markdown files (notes, exported todos, etc.). | `--dry-run`, `--type`, `--vault`, `--inbox`, `--tags`, `--contexts`, `--projects`, `--meta`. |
| `nanite search <keywords>` | Fuzzy search over the active vault (full text + filters). | `--vault`, `--type`, `--tags`, `--contexts`, `--projects`, `--page`, `--page-size`. |
| `nanite inbox [keywords]` | Quickly review inbox-only items. | `--page`, `--page-size`. |
| `nanite context [topic]` | Pull context snapshots for agents or personal reference. Topics: `app`, `schema`, `stats`, `cache`. | `--refresh` forces a rebuild of `${config}/context-cache/snapshot.json`. |

Legacy verbs (`nanite push`, `nanite get`) still work for backwards compatibility, but the new `add`/`show` commands should be used going forward.

## Practical examples

```bash
# Capture a Markdown note stored in a file
nanite add "Reading notes" --file ~/Desktop/reading.md --type note --vault myvault

# List overdue tasks (limit to 10) in the active vault
nanite list overdue --limit 10

# Complete items 12,15,18 and drop the #someday tag
nanite update --ids 12,15,18 --complete --tags-remove someday

# Refresh and print the context snapshot (great for agents)
nanite context cache --refresh | jq '.'
```

All commands run against your local vaults via the embedded Go runtime—no server process required.
