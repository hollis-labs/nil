---
intent: pcc
updated_at: 2026-02-22
---

# 03 — Workflows

## Active Workflow: Chat Addon (F5) — Inception Loop

### Phase Summary
- **Iteration 1** (current): Inception bootstrap — install Forge, create PCC + all artifacts, define initial tasks
- **Iteration 2+**: Execute MVP tasks from `.forge/tasks/`

### Canonical Inception Loop
1. Read `.forge/bootstrap.md` for current state
2. Select next `todo` task (A > B > C, oldest first)
3. Execute: small, verifiable step
4. Verify against acceptance criteria
5. Append `Updates` to task file
6. Write run log → `.forge/logs/run-YYYYMMDD-iterNN.md`
7. Update `.forge/bootstrap.md`
8. Commit once per iteration

### Task Status Flow
`todo` → `doing` → `done` | `blocked` | `paused`

### Task File Convention
Location: `.forge/tasks/TASK-YYYYMMDD-NNN.md`
Required fields: id, title, status, priority (A|B|C), description, acceptance, verification

### Backlog Convention
Location: `.forge/backlog/BACKLOG-YYYYMMDD-NNN.md`
Promotion path: backlog → task (when prioritized for current sprint)

### Artifact Convention
Location: `.forge/artifacts/` (knowledge artifacts from investigation)
Intent: `knowledge_artifact` (frontmatter)

## Wails Dev Workflow
```bash
wails dev              # hot-reload dev mode
go build .             # test Go layer compiles
wails generate module  # regenerate TS bindings after Go method changes
cd frontend && npx tsc --noEmit  # type-check frontend
```

## Feature Deployment Path
1. Go changes → `store/store.go` + `app.go` → `go build .` verify
2. Schema changes → `store/schema.sql` + migration slice
3. New Wails methods → `wails generate module`
4. Frontend changes → hot-reload via `wails dev`
5. Commit + push → GitHub (hollis-labs/nanite)

## Evidence
- Inspected: forge.yaml, docs/13_inception-workflow.md, CLAUDE.md
- Updated: 2026-02-22
