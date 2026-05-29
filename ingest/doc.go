// Package ingest converts between TipTap/ProseMirror document JSON and the
// transport formats accepted at NIL's edges (markdown, HTML). The PM JSON
// produced here aims to match TipTap StarterKit + tiptap-markdown +
// WikilinkExtension output closely enough that the live editor accepts it
// without normalization on first save. Exact byte-parity with TipTap output
// is not guaranteed — the frontend backfill pass uses the real editor for
// content that was authored interactively.
package ingest

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Node is the on-wire shape of a ProseMirror node. It deliberately uses
// generic map types so the same struct can represent the doc, block, and
// inline levels without per-node Go types.
type Node struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []Node         `json:"content,omitempty"`
	Marks   []Mark         `json:"marks,omitempty"`
	Text    string         `json:"text,omitempty"`
}

// Mark is an inline annotation (bold, italic, code, link, ...).
type Mark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

// EmptyDoc returns the canonical empty TipTap doc — a doc with one empty
// paragraph. Matches what the editor produces for a blank document.
func EmptyDoc() Node {
	return Node{
		Type:    "doc",
		Content: []Node{{Type: "paragraph"}},
	}
}

// MarshalDoc returns the JSON encoding of a doc node, suitable for storing
// in todos.notes_doc.
func MarshalDoc(n Node) (string, error) {
	b, err := json.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("ingest: marshal doc: %w", err)
	}
	return string(b), nil
}

// UnmarshalDoc parses notes_doc JSON back into a Node tree.
func UnmarshalDoc(jsonStr string) (Node, error) {
	var n Node
	if jsonStr == "" {
		return EmptyDoc(), nil
	}
	if err := json.Unmarshal([]byte(jsonStr), &n); err != nil {
		return Node{}, fmt.Errorf("ingest: unmarshal doc: %w", err)
	}
	if n.Type == "" {
		return EmptyDoc(), nil
	}
	return n, nil
}

// DocToPlainText walks a doc tree and concatenates text content with paragraph
// breaks. Used to derive the FTS5 indexable text and for AI context excerpts.
func DocToPlainText(jsonStr string) (string, error) {
	n, err := UnmarshalDoc(jsonStr)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	writePlainText(&b, n, true)
	out := strings.TrimSpace(b.String())
	// Collapse 3+ newlines down to 2 for cleaner FTS indexing.
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return out, nil
}

func writePlainText(b *strings.Builder, n Node, isRoot bool) {
	switch n.Type {
	case "text":
		b.WriteString(n.Text)
	case "hardBreak":
		b.WriteString("\n")
	case "wikilink":
		// Render as the visible label so search picks up wikilink targets by name.
		if label, ok := n.Attrs["label"].(string); ok && label != "" {
			b.WriteString(label)
		}
	case "horizontalRule":
		b.WriteString("\n")
	default:
		for _, c := range n.Content {
			writePlainText(b, c, false)
		}
		// Block-level nodes get a paragraph break after their content.
		if isBlockNode(n.Type) && !isRoot {
			b.WriteString("\n\n")
		}
	}
}

func isBlockNode(t string) bool {
	switch t {
	case "paragraph", "heading", "blockquote", "codeBlock",
		"bulletList", "orderedList", "listItem", "horizontalRule":
		return true
	}
	return false
}
