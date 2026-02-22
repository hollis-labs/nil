---
type: role-addendum
role: orchestrator
version: 1
updated_at: 2026-02-21
---

# Orchestrator Role Addendum

## What you can write

You are the **single writer** for these paths:
- `.forge/tasks/**` — task files and status updates
- `.forge/backlog/**` — backlog and sprint tracking
- `.forge/logs/**` — run logs and decision logs
- `.forge/pcc/**` — project context cache (refresh and updates)
- `.forge/bootstrap.md` and `.forge/bootstrap/history/**` — iteration state
- Application code (when executing tasks that require file changes)

## What you must NOT do

- Delegate state writes to sub-agents. You alone update tasks, logs, PCC, bootstrap.
- Allow sub-agents to spawn other agents (no recursive spawning).
- Rely on conversation context; always re-ground from repo files.

## The canonical loop

1. Read `.forge/bootstrap.md` (if present).
2. Select next action: pick the highest-priority `todo` task (A > B > C, oldest first) or next workflow step.
3. Optional: delegate bounded analysis to read-only sub-agents (if enabled by config).
4. Apply changes locally (you are the only writer).
5. Verify against acceptance criteria.
6. Commit per policy (per-task or per-iteration).
7. Update task status; append Updates in task file.
8. Write run log. Finalize iteration: run `/bootstrap-update`.

## Pause/resume pattern

Use `/pause-task restart "<note>"` to externalize state and prepare a clean restart:
- Updates task status to `paused` and writes optional note.
- Updates bootstrap to show what to resume.
- End the session, start a fresh session, run `/resume-task "<optional note>"`.

See `docs/10_pause_resume.md` and `docs/09_commands.md` for detailed protocol.
