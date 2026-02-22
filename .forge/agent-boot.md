---
type: agent-boot
version: 1
updated_at: 2026-02-21
---

# Forge Agent Boot

## What is Forge

Forge is a multi-session orchestration system for repository automation. It coordinates human-directed workflows, task execution, and knowledge artifact generation through a single-writer Orchestrator that delegates bounded read-only work to sub-agents.

## Ground truth (always read these first)

- `forge.yaml` — system configuration (roles, storage, workflows, observability)
- `.forge/bootstrap.md` — current iteration state and next actions
- `.forge/pcc/` — project context cache (agent-only, ground truth for repo info)
- `.forge/tasks/` — task backlog and execution state
- `.forge/logs/` — run logs and decision logs
- `docs/03_workflow-contracts.md` — execution contract for workflows
- `docs/08_orchestrator.md` — Orchestrator role responsibilities

## Core rules

- **Single writer**: Only the Orchestrator may modify tasks, logs, PCC, or bootstrap. Sub-agents are read-only.
- **Ground truth in files, not chat**: Always re-ground from repo artifacts, never rely on conversation context.
- **Minimal diffs**: Small, verifiable steps. Each task should produce incremental changes.
- **Bootstrap boundaries**: Finalize each iteration with `/bootstrap-update` to enable clean restarts.
- **Bounded delegation**: Sub-agents execute single scoped objectives; no spawning other agents.
- **Deterministic pause/resume**: Use `/pause-task` and `/resume-task` with state on disk, not memory.

## How to start

1. Read `.forge/bootstrap.md` (if present) for current state and next action.
2. Identify your role: check `.forge/boot/` for your role addendum (orchestrator.md, worker.md, or reviewer.md).
3. For Orchestrator: read tasks from `.forge/tasks/`, select next todo by priority, execute, verify, and log results.
4. For Workers/Reviewers: execute your single objective, return results in requested format, do not modify files.

## Your role

You are operating in Forge mode. Load the addendum for your role:
- **Orchestrator**: `.forge/boot/orchestrator.md` — you drive loops, write state, finalize iterations
- **Worker**: `.forge/boot/worker.md` — you execute scoped tasks, return results, read-only
- **Reviewer**: `.forge/boot/reviewer.md` — you scan and summarize, read-only

## Reference map

| Topic | Doc |
|---|---|
| System config | `docs/01_config.md` |
| Project context cache | `docs/02_pcc.md` |
| Workflow contracts | `docs/03_workflow-contracts.md` |
| Task model | `docs/04_task-model.md` |
| Iteration bootstrap | `docs/05_bootstrap.md` |
| Loop runner | `docs/06_loop-runner.md` |
| Sub-agents | `docs/07_subagents.md` |
| Orchestrator mode | `docs/08_orchestrator.md` |
| User commands | `docs/09_commands.md` |
| Pause/resume | `docs/10_pause_resume.md` |
