package store

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hollis-labs/nil/ingest"
)

// reHTMLBR matches <br> / <br/> variants.
var reHTMLBR = regexp.MustCompile(`(?i)<br\s*/?>`)

// reHTMLBlock matches closing block-level tags whose content ends a "paragraph".
var reHTMLBlock = regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|blockquote)>`)

// reHTMLTag strips any remaining HTML tag.
var reHTMLTag = regexp.MustCompile(`<[^>]+>`)

// reMultiNL collapses 3+ consecutive newlines to 2.
var reMultiNL = regexp.MustCompile(`\n{3,}`)

// stripHTML converts TipTap HTML to plain text for FTS5 indexing and AI context.
// It preserves paragraph breaks and handles common HTML entities. No CGo required.
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	s = reHTMLBR.ReplaceAllString(s, "\n")
	s = reHTMLBlock.ReplaceAllString(s, "\n\n")
	s = reHTMLTag.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = reMultiNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// updateFTSTx writes (or replaces) the FTS5 row for an item. Called after
// CreateItem and UpdateItem since v8 dropped the trigger-based sync in favor
// of standalone FTS5 with app-layer writes.
func updateFTSTx(ctx context.Context, dbx dbtx, id int64, title, text string) error {
	if _, err := dbx.ExecContext(ctx, "DELETE FROM todos_fts WHERE rowid = ?", id); err != nil {
		return fmt.Errorf("updateFTS delete: %w", err)
	}
	if _, err := dbx.ExecContext(ctx, "INSERT INTO todos_fts(rowid, title, notes_text) VALUES (?,?,?)", id, title, text); err != nil {
		return fmt.Errorf("updateFTS insert: %w", err)
	}
	return nil
}

// derivePlainText returns the text projection used for FTS5 indexing. Walks
// the PM JSON doc tree concatenating text nodes; returns empty for an empty
// or unparseable doc (FTS row will then match by title only).
func derivePlainText(notesDoc string) string {
	if notesDoc == "" {
		return ""
	}
	text, err := ingest.DocToPlainText(notesDoc)
	if err != nil {
		return ""
	}
	return text
}
