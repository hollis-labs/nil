package ingest

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTMLToDoc parses an HTML fragment (typical TipTap output: a sequence of
// block elements with no <html>/<body> wrapper) into PM JSON.
// Handles the StarterKit subset plus the WikilinkExtension's
// <span data-type="wikilink" data-id data-ref-type data-label> nodes.
func HTMLToDoc(htmlStr string) (string, error) {
	if strings.TrimSpace(htmlStr) == "" {
		return MarshalDoc(EmptyDoc())
	}
	// Parse as a fragment with <body> as the context — yields a flat list of
	// block siblings without forcing a full document wrapper.
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(htmlStr), body)
	if err != nil {
		return "", err
	}

	doc := Node{Type: "doc"}
	for _, n := range nodes {
		blocks := htmlBlocksFromNode(n)
		doc.Content = append(doc.Content, blocks...)
	}
	if len(doc.Content) == 0 {
		doc.Content = []Node{{Type: "paragraph"}}
	}
	return MarshalDoc(doc)
}

// htmlBlocksFromNode emits zero or more block nodes from a parsed HTML node.
// Top-level text nodes (whitespace only between blocks) are skipped; non-
// whitespace top-level text gets wrapped in a paragraph.
func htmlBlocksFromNode(n *html.Node) []Node {
	switch n.Type {
	case html.TextNode:
		if strings.TrimSpace(n.Data) == "" {
			return nil
		}
		return []Node{{Type: "paragraph", Content: []Node{{Type: "text", Text: n.Data}}}}
	case html.ElementNode:
		if pm, ok := htmlElementToBlock(n); ok {
			return []Node{pm}
		}
		// Unknown element at block level: flatten inline content into a paragraph.
		inline := htmlInlineFromChildren(n, nil)
		if len(inline) == 0 {
			return nil
		}
		return []Node{{Type: "paragraph", Content: inline}}
	}
	// Document/Doctype/Comment: walk children.
	var out []Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, htmlBlocksFromNode(c)...)
	}
	return out
}

func htmlElementToBlock(n *html.Node) (Node, bool) {
	switch n.DataAtom {
	case atom.P:
		return Node{Type: "paragraph", Content: htmlInlineFromChildren(n, nil)}, true
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		level := int(n.Data[1] - '0')
		return Node{
			Type:    "heading",
			Attrs:   map[string]any{"level": level},
			Content: htmlInlineFromChildren(n, nil),
		}, true
	case atom.Ul:
		return Node{Type: "bulletList", Content: htmlListItems(n)}, true
	case atom.Ol:
		attrs := map[string]any{}
		if v := getAttr(n, "start"); v != "" {
			if i, err := strconv.Atoi(v); err == nil && i != 1 {
				attrs["start"] = i
			}
		}
		out := Node{Type: "orderedList", Content: htmlListItems(n)}
		if len(attrs) > 0 {
			out.Attrs = attrs
		}
		return out, true
	case atom.Blockquote:
		var content []Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			content = append(content, htmlBlocksFromNode(c)...)
		}
		if len(content) == 0 {
			content = []Node{{Type: "paragraph"}}
		}
		return Node{Type: "blockquote", Content: content}, true
	case atom.Pre:
		// <pre><code class="language-X">...</code></pre>
		var codeNode *html.Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.DataAtom == atom.Code {
				codeNode = c
				break
			}
		}
		var lang any
		text := ""
		if codeNode != nil {
			if cls := getAttr(codeNode, "class"); strings.HasPrefix(cls, "language-") {
				lang = strings.TrimPrefix(cls, "language-")
			}
			text = collectText(codeNode)
		} else {
			text = collectText(n)
		}
		attrs := map[string]any{"language": lang}
		var content []Node
		if text != "" {
			content = []Node{{Type: "text", Text: text}}
		}
		return Node{Type: "codeBlock", Attrs: attrs, Content: content}, true
	case atom.Hr:
		return Node{Type: "horizontalRule"}, true
	}
	return Node{}, false
}

func htmlListItems(list *html.Node) []Node {
	var items []Node
	for c := list.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.DataAtom != atom.Li {
			continue
		}
		items = append(items, htmlListItemToNode(c))
	}
	return items
}

func htmlListItemToNode(li *html.Node) Node {
	var content []Node
	hasBlockChild := false
	for c := li.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			if _, ok := htmlElementToBlock(c); ok {
				hasBlockChild = true
				break
			}
		}
	}
	if hasBlockChild {
		for c := li.FirstChild; c != nil; c = c.NextSibling {
			content = append(content, htmlBlocksFromNode(c)...)
		}
	} else {
		// Inline content only — wrap in a paragraph (TipTap listItem expects block children).
		inline := htmlInlineFromChildren(li, nil)
		if len(inline) == 0 {
			content = []Node{{Type: "paragraph"}}
		} else {
			content = []Node{{Type: "paragraph", Content: inline}}
		}
	}
	return Node{Type: "listItem", Content: content}
}

// htmlInlineFromChildren walks inline children, producing PM inline nodes
// with the active mark stack applied to text leaves.
func htmlInlineFromChildren(n *html.Node, marks []Mark) []Node {
	var out []Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = appendHTMLInline(out, c, marks)
	}
	return out
}

func appendHTMLInline(out []Node, n *html.Node, marks []Mark) []Node {
	switch n.Type {
	case html.TextNode:
		if n.Data == "" {
			return out
		}
		node := Node{Type: "text", Text: n.Data}
		if len(marks) > 0 {
			node.Marks = append([]Mark(nil), marks...)
		}
		return append(out, node)
	case html.ElementNode:
		// Wikilink extension: <span data-type="wikilink" data-id data-ref-type data-label>
		if n.DataAtom == atom.Span && getAttr(n, "data-type") == "wikilink" {
			attrs := map[string]any{}
			if v := getAttr(n, "data-id"); v != "" {
				if i, err := strconv.ParseInt(v, 10, 64); err == nil {
					attrs["id"] = i
				} else {
					attrs["id"] = v
				}
			}
			if v := getAttr(n, "data-ref-type"); v != "" {
				attrs["refType"] = v
			}
			if v := getAttr(n, "data-label"); v != "" {
				attrs["label"] = v
			} else {
				attrs["label"] = collectText(n)
			}
			node := Node{Type: "wikilink", Attrs: attrs}
			if len(marks) > 0 {
				node.Marks = append([]Mark(nil), marks...)
			}
			return append(out, node)
		}
		switch n.DataAtom {
		case atom.Br:
			return append(out, Node{Type: "hardBreak"})
		case atom.Strong, atom.B:
			newMarks := append([]Mark(nil), marks...)
			newMarks = append(newMarks, Mark{Type: "bold"})
			return append(out, htmlInlineFromChildren(n, newMarks)...)
		case atom.Em, atom.I:
			newMarks := append([]Mark(nil), marks...)
			newMarks = append(newMarks, Mark{Type: "italic"})
			return append(out, htmlInlineFromChildren(n, newMarks)...)
		case atom.Code:
			newMarks := append([]Mark(nil), marks...)
			newMarks = append(newMarks, Mark{Type: "code"})
			return append(out, htmlInlineFromChildren(n, newMarks)...)
		case atom.S, atom.Strike, atom.Del:
			newMarks := append([]Mark(nil), marks...)
			newMarks = append(newMarks, Mark{Type: "strike"})
			return append(out, htmlInlineFromChildren(n, newMarks)...)
		case atom.A:
			attrs := map[string]any{"href": getAttr(n, "href")}
			if v := getAttr(n, "title"); v != "" {
				attrs["title"] = v
			}
			if v := getAttr(n, "target"); v != "" {
				attrs["target"] = v
			}
			newMarks := append([]Mark(nil), marks...)
			newMarks = append(newMarks, Mark{Type: "link", Attrs: attrs})
			return append(out, htmlInlineFromChildren(n, newMarks)...)
		}
		// Unknown inline element: walk children with current marks.
		return append(out, htmlInlineFromChildren(n, marks)...)
	}
	return out
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func collectText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
