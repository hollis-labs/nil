---
id: BACKLOG-E1-S5-001
title: "Automation: job scheduler + headless agent sessions"
priority: B
epic: EPIC-E1
sprint: 5
created_at: 2026-02-22
---

## Description
Enable scheduled agentic workflows ("send me a weekly review every Sunday"). A cron-style scheduler runs inside the Wails backend. Jobs trigger headless agent sessions (same Bridge/Store infrastructure, no UI). Output is delivered as a note in the inbox or via OS notification.

## Components

### Job Scheduler
- `robfig/cron` or a simple ticker-based scheduler in Go
- Jobs stored in a `scheduled_jobs` table in `chat.db`
- Schema: `{id, name, cron_expr, template_slug, params_json, vault_id, delivery, enabled, last_run, next_run}`

### Headless Agent Session
- `RunHeadlessJob(ctx, job) error` in `chat` package
- Creates a synthetic chat session (no UI binding)
- Runs the tool-use loop against the specified template
- On completion: delivers output via configured delivery method

### Delivery Methods
- `inbox`: write output as a new note to vault inbox
- `notification`: OS-level notification (Wails runtime notify)
- `email`: SMTP (future, requires config)

## Acceptance Criteria
- [ ] `scheduled_jobs` table in `chat.db`
- [ ] Job CRUD: Wails-bound `ListJobs`, `CreateJob`, `UpdateJob`, `DeleteJob`, `RunJobNow`
- [ ] Scheduler goroutine started in `startup()`, evaluates cron expressions
- [ ] `RunHeadlessJob` executes full tool-use loop, no UI binding
- [ ] Delivery: inbox note (minimum viable), notification (nice to have)
- [ ] Built-in job: "weekly-review" template every Sunday at 08:00 (disabled by default)
- [ ] Settings UI: Jobs tab (list, enable/disable, run now)
- [ ] `go build .` clean

## Blocked by
BACKLOG-E1-S3-001, BACKLOG-E1-S3-002 (templates needed for job definitions)
