---
id: ADR-001
title: "Direct Anthropic API vs CLI headless tools (Claude Code, opencode)"
status: accepted
date: 2026-02-22
deciders: [product, engineering]
---

# ADR-001 — Direct Anthropic API vs CLI Headless Tools

## Context

NANITE needs an AI agent that can search vault data, propose mutations, and eventually run scheduled automations. The question is whether to integrate via the Anthropic API directly, or by shelling out to a CLI agent tool (Claude Code in headless mode, opencode, etc.).

## Options Considered

### Option A: Direct Anthropic API (current approach)
Call `api.anthropic.com/v1/messages` from Go with `net/http`. Define tools explicitly. Run the agentic loop in-process.

### Option B: Claude Code CLI in headless mode
Spawn a subprocess, communicate via stdin/stdout, rely on Claude Code's built-in tool ecosystem.

### Option C: opencode CLI
Similar subprocess-based approach, different tool set.

## Decision

**Option A: Direct Anthropic API.**

## Rationale

1. **Data is in-process.** The vault lives in an SQLite file opened by the Wails backend. No external process can access it without either a socket server (complexity) or file-system reads (brittleness). The direct API approach keeps data access in-process where it belongs.

2. **CLI tools are developer tools, not app integration tools.** Claude Code and opencode are designed for file systems and shell commands in development environments. Their built-in tools (file read/write, bash exec) are irrelevant to NANITE's domain. You'd be fighting the abstraction layer.

3. **Full control over the tool surface.** NANITE-specific tools (`search_vault`, `list_taxonomy`, `get_vault_stats`) can be precisely defined with correct schemas. No translation layer between NANITE concepts and generic CLI affordances.

4. **No subprocess lifecycle management.** Spawning, communicating with, and killing subprocesses across macOS/Windows/Linux adds significant complexity and surface area for platform-specific bugs.

5. **Same infrastructure for automation.** Scheduled headless agent runs (weekly reviews, auto-classifiers) can reuse the same Bridge/Store architecture. No different integration path for interactive vs automated use.

## Consequences

- We own the full tool execution loop. Any new capability requires a new tool definition + store method.
- Dependency: Anthropic API key required (already captured in Settings → Chat).
- Must implement tool use, caching, and context management ourselves (done).
- Future: could wrap the API with a higher-level agent SDK if one becomes stable — but the architecture doesn't require it.
