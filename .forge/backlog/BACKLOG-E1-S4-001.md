---
id: BACKLOG-E1-S4-001
title: "Semantic search: background embeddings pipeline"
priority: B
epic: EPIC-E1
sprint: 4
created_at: 2026-02-22
---

## Description
Generate and store text embeddings for vault items. Embeddings enable semantic search — finding conceptually related items that don't share keywords. Background generation is non-blocking and resilient.

See ADR-004 for storage approach and model decisions.

## Implementation

### DB (migration v6)
```sql
ALTER TABLE todos ADD COLUMN notes_embedding BLOB;
ALTER TABLE todos ADD COLUMN embedding_model TEXT;
ALTER TABLE todos ADD COLUMN embedded_at TEXT;
```

### Embedding Worker
- Goroutine started in `startup()`, listens on a buffered channel
- On `CreateItem`/`UpdateItem`: send item ID to channel
- Worker calls Anthropic (or configured) embeddings API with `notes_text`
- Stores `float32` array as JSON blob, records model name and timestamp
- Rate-limited: max N concurrent requests, exponential backoff on 429

### `Store.GetEmbedding(ctx, id)` + `Store.SetEmbedding(ctx, id, model, vec)`

### Backfill (migration time)
Query all items where `notes_embedding IS NULL`, enqueue for embedding.

## Acceptance Criteria
- [ ] Migration v6: 3 new columns on `todos`
- [ ] Embedding worker goroutine in `vault` package or `app.go`
- [ ] `Store.SetEmbedding()` + `Store.GetItemsWithEmbeddings()` methods
- [ ] Channel-based queue: non-blocking enqueue from CreateItem/UpdateItem
- [ ] Configurable: embedding model in `ChatConfig.EmbeddingModel`
- [ ] Graceful: no embeddings = no semantic search (FTS5 fallback)
- [ ] `go build .` clean

## Blocked by
TASK-20260222-013 (needs notes_text as embedding input)
