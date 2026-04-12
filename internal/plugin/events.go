package plugin

import (
	"time"

	"github.com/hollis-labs/plugin"
)

// Nil Event Catalog
// These are the standard events that Nil emits for plugins to listen to.
const (
	// Ingestion events
	EventIngestCompleted = "ingest.completed"

	// Query events
	EventQueryCompleted = "query.completed"

	// Chunking events
	EventChunkCreated = "chunk.created"
)

// NewEvent creates a new plugin event with the given type, source, and data.
func NewEvent(eventType, source string, data map[string]interface{}) plugin.Event {
	return plugin.Event{
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// EmitIngestCompleted emits an ingest.completed event.
func (h *Host) EmitIngestCompleted(source string, itemCount int) {
	h.EmitEvent(NewEvent(EventIngestCompleted, "nil", map[string]interface{}{
		"source":     source,
		"item_count": itemCount,
	}))
}

// EmitQueryCompleted emits a query.completed event.
func (h *Host) EmitQueryCompleted(query string, resultCount int) {
	h.EmitEvent(NewEvent(EventQueryCompleted, "nil", map[string]interface{}{
		"query":        query,
		"result_count": resultCount,
	}))
}

// EmitChunkCreated emits a chunk.created event.
func (h *Host) EmitChunkCreated(source string, chunkIndex int) {
	h.EmitEvent(NewEvent(EventChunkCreated, "nil", map[string]interface{}{
		"source":      source,
		"chunk_index": chunkIndex,
	}))
}
