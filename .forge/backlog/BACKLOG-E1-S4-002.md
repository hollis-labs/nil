---
id: BACKLOG-E1-S4-002
title: "Semantic search: semantic_search tool + hybrid FTS5+semantic re-rank"
priority: B
epic: EPIC-E1
sprint: 4
created_at: 2026-02-22
---

## Description
Add `semantic_search` tool to the Bridge. Embeds the query, computes cosine similarity against stored item embeddings, returns top-K. Hybrid mode merges FTS5 and semantic candidates and re-ranks by combined score.

## Cosine Similarity (pure Go)
```go
func cosineSim(a, b []float32) float32 {
    var dot, normA, normB float32
    for i := range a {
        dot += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    return dot / (sqrt(normA) * sqrt(normB))
}
```

## Tool Definition
`semantic_search` input: `{query: string, type: string, limit: int}`
- Embeds `query` via API
- Loads items with embeddings (filtered by type)
- Computes cosine similarity for each
- Returns top-K sorted by score

## Hybrid Search (in search_vault tool)
Add optional `mode: "semantic" | "keyword" | "hybrid"` param to `search_vault`.
- `keyword` (default): existing FTS5
- `semantic`: cosine similarity only
- `hybrid`: union of FTS5 top-20 + semantic top-20, re-rank by `0.6*semantic + 0.4*bm25`, return top-K

## Acceptance Criteria
- [ ] `Store.GetItemsForSemanticSearch(ctx, type, status) ([]ItemWithEmbedding, error)` — returns slim items + float32 vecs
- [ ] `cosineSim` function in store package
- [ ] `semantic_search` tool registered in bridge
- [ ] `search_vault` gains optional `mode` param
- [ ] Graceful: if item has no embedding, excluded from semantic results (not an error)
- [ ] System prompt: "prefer semantic_search for conceptual/relational queries; use search_vault for keyword/filter queries"
- [ ] `go build .` clean

## Blocked by
BACKLOG-E1-S4-001
