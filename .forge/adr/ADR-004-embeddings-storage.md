---
id: ADR-004
title: "Embeddings storage approach for semantic search"
status: proposed
date: 2026-02-22
deciders: [engineering]
---

# ADR-004 — Embeddings Storage for Semantic Search

## Context

FTS5 keyword search fails for conceptually related queries ("find everything about project planning" when notes say "roadmap", "milestones", "sprint"). Semantic search via embeddings is needed for Sprint 4. The stack constraint is `modernc.org/sqlite` (pure Go, no CGo), which rules out sqlite-vec (C extension) and most Go vector libraries.

## Options Considered

### Option A: sqlite-vec extension
Fast C extension for SQLite vector operations.
**Rejected**: Requires CGo or dlopen. Incompatible with modernc.org/sqlite.

### Option B: External vector database (Chroma, Qdrant, Weaviate)
Purpose-built for vector search.
**Rejected**: External process/service dependency. Adds infrastructure burden for a personal desktop app. Defeats the "no external deps" principle.

### Option C: Anthropic Embeddings API + pure Go cosine similarity + SQLite BLOB storage
Generate embeddings via `api.anthropic.com/v1/embeddings` (or OpenAI). Store as JSON-encoded float32 arrays in a `notes_embedding` BLOB column. Implement cosine similarity in Go (~10 lines).

**Selected.**

### Option D: BM25 improvements (enhanced FTS5)
Tune FTS5 weights, add stemming, improve tokenization.
**Partial mitigation only**: Cannot bridge semantic gaps (roadmap ≠ planning).

## Decision: Option C

Store embeddings as JSON float32 arrays in SQLite. Compute cosine similarity in Go at query time.

## Implementation Plan (Sprint 4)

1. **Migration**: `ALTER TABLE todos ADD COLUMN notes_embedding BLOB` + `notes_embedding_model TEXT` (track which model generated it).
2. **Background job**: on `CreateItem`/`UpdateItem`, enqueue embedding generation. Worker calls embeddings API with `notes_text` (see ADR-002), stores result as JSON blob.
3. **Cosine similarity in Go**:
   ```go
   func cosineSim(a, b []float32) float32 { /* dot / (|a| * |b|) */ }
   ```
4. **`semantic_search` tool**: loads candidate items (filtered by type/status), computes similarity against query embedding, returns top-K.
5. **Hybrid search**: FTS5 candidates union semantic candidates, re-ranked by combined score. Prefer semantic for "find related", FTS5 for exact keyword.

## Performance

For vaults up to ~10k items (reasonable personal use):
- Loading 10k float32[1536] arrays: ~60MB RAM — acceptable
- Cosine similarity over 10k vectors: <10ms on modern hardware
- Scales poorly above 100k items — external vector index needed at that point (future concern)

## Embedding Model

Default: `voyage-3-lite` via Anthropic API (cheapest, sufficient for personal vaults). Fallback: OpenAI `text-embedding-3-small`. Model stored per-row so re-embedding on model change is scoped to changed items only.

## Consequences

- Requires embedding API key (can reuse Anthropic key or separate config)
- Embeddings are regenerated when `notes_text` changes (background, non-blocking)
- Cold start: first vault open triggers background embedding of all existing items
- No semantic search until embeddings are generated (graceful fallback to FTS5 only)
