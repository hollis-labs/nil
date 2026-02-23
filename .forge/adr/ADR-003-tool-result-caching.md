---
id: ADR-003
title: "In-session tool result caching strategy"
status: accepted
date: 2026-02-22
deciders: [engineering]
---

# ADR-003 — Tool Result Caching Strategy

## Context

The agentic loop calls `executeSearchVault` on every user message that requires data. When a user asks a follow-up like "can you sort that by title instead?" Claude will call `search_vault` again with different sort parameters. Without caching, this hits the SQLite store every time. More importantly, a re-sort of already-fetched data doesn't need a new DB query at all — the same result set can be sorted in memory.

Additionally, all tool invocations should be logged for audit, debugging, and future RAG over conversation history.

## Decision

Two-layer approach:

### Layer 1: In-memory session cache (`ToolCache`)
- Scoped per chat session (`sessionID → *ToolCache`)
- Keyed by `toolName + ":" + inputJSON` (exact match)
- FIFO eviction, default max 20 entries
- Lives in `app.go` `sessionCaches map[int64]*ToolCache`
- Created on first use, freed on `EndChatSession`

### Layer 2: Persistent tool call log (`chat_tool_calls` table)
- Written to `chat.db` after every tool call (cache hit or miss)
- Fields: session_id, message_id, tool_name, input_json, result_json, cache_hit, created_at
- Best-effort (non-fatal on write error)
- Queryable via `GetRecentToolCalls(sessionID, limit)`

## Cache Scope: Exact Input Match Only

The cache uses the exact JSON input as the key. This means:
- `search_vault({sort_by: "created_at", limit: 10})` hits cache for identical repeat calls
- `search_vault({sort_by: "title", limit: 10})` is a cache miss (different input)
- "Sort by title" follow-ups still hit the DB — Claude generates a new tool call with different params

This is intentional. Client-side re-sorting of cached result sets (without any DB or API call) would require detecting user intent independently of Claude, which is fragile. The better UX is:
- Cache hit eliminates DB query for identical calls (e.g. user asks same question twice)
- Cache miss still saves an API round-trip because Claude uses tool results from earlier in the same conversation context window

## What the Cache Does NOT Do

- Does not re-sort in-memory (a cache miss with different sort parameters hits the DB)
- Does not invalidate on vault writes (acceptable: chat sessions are short, eventual consistency is fine)
- Does not persist across sessions (intentional: stale data between sessions is worse than a fresh query)

## Future Enhancement

Sprint 4+ could add a "result set cache" separate from the tool input cache: store the raw rows from the last N searches, and when Claude requests a re-sort, apply the sort in Go without hitting the DB. This is the "simple sort" optimization referenced in the ideation discussion. Deferred until the simpler approach proves insufficient.

## Consequences

- `ToolCache` is in `chat/models.go`, exported (app.go creates it, bridge.go uses it)
- Bridge signature: `executeTool(ctx, block, store, cache) (result string, cacheHit bool)`
- `BridgeRequest.ToolCache *ToolCache` is nil-safe (disabled when nil)
- Logging is fire-and-forget (`_ = a.chatStore.PersistToolCall(...)`)
