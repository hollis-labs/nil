package ingest

import (
	"fmt"
	"html"
	"strings"
)

// DocToHTML renders PM JSON to HTML matching what TipTap's StarterKit +
// WikilinkExtension serialization produces. Used to populate the notes_html
// cache when input came in as markdown/HTML (the GUI save path computes its
// own HTML via TipTap and sends both).
func DocToHTML(jsonStr string) (string, error) {
	n, err := UnmarshalDoc(jsonStr)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, c := range n.Content {
		writeBlockHTML(&b, c)
	}
	return b.String(), nil
}

func writeBlockHTML(b *strings.Builder, n Node) {
	switch n.Type {
	case "paragraph":
		b.WriteString("<p>")
		writeInlineHTML(b, n.Content)
		b.WriteString("</p>")
	case "heading":
		level := 1
		if v, ok := n.Attrs["level"].(float64); ok {
			level = int(v)
		} else if v, ok := n.Attrs["level"].(int); ok {
			level = v
		}
		if level < 1 || level > 6 {
			level = 1
		}
		fmt.Fprintf(b, "<h%d>", level)
		writeInlineHTML(b, n.Content)
		fmt.Fprintf(b, "</h%d>", level)
	case "bulletList":
		b.WriteString("<ul>")
		for _, c := range n.Content {
			writeBlockHTML(b, c)
		}
		b.WriteString("</ul>")
	case "orderedList":
		startAttr := ""
		if v, ok := n.Attrs["start"]; ok {
			startAttr = fmt.Sprintf(` start="%v"`, v)
		}
		fmt.Fprintf(b, "<ol%s>", startAttr)
		for _, c := range n.Content {
			writeBlockHTML(b, c)
		}
		b.WriteString("</ol>")
	case "listItem":
		b.WriteString("<li>")
		for _, c := range n.Content {
			writeBlockHTML(b, c)
		}
		b.WriteString("</li>")
	case "blockquote":
		b.WriteString("<blockquote>")
		for _, c := range n.Content {
			writeBlockHTML(b, c)
		}
		b.WriteString("</blockquote>")
	case "codeBlock":
		lang := ""
		if v, ok := n.Attrs["language"].(string); ok && v != "" {
			lang = fmt.Sprintf(` class="language-%s"`, html.EscapeString(v))
		}
		fmt.Fprintf(b, "<pre><code%s>", lang)
		for _, c := range n.Content {
			if c.Type == "text" {
				b.WriteString(html.EscapeString(c.Text))
			}
		}
		b.WriteString("</code></pre>")
	case "horizontalRule":
		b.WriteString("<hr>")
	case "hardBreak":
		b.WriteString("<br>")
	case "text":
		// Stray text at block level — wrap in a paragraph for safety.
		b.WriteString("<p>")
		writeInlineNodeHTML(b, n)
		b.WriteString("</p>")
	default:
		// Unknown node: render children if any, else skip.
		for _, c := range n.Content {
			writeBlockHTML(b, c)
		}
	}
}

func writeInlineHTML(b *strings.Builder, nodes []Node) {
	for _, n := range nodes {
		writeInlineNodeHTML(b, n)
	}
}

func writeInlineNodeHTML(b *strings.Builder, n Node) {
	switch n.Type {
	case "text":
		openMarks, closeMarks := renderMarkTags(n.Marks)
		b.WriteString(openMarks)
		b.WriteString(html.EscapeString(n.Text))
		b.WriteString(closeMarks)
	case "hardBreak":
		b.WriteString("<br>")
	case "wikilink":
		var dataID, refType, label string
		if v, ok := n.Attrs["id"]; ok {
			dataID = fmt.Sprintf("%v", v)
		}
		if v, ok := n.Attrs["refType"].(string); ok {
			refType = v
		}
		if v, ok := n.Attrs["label"].(string); ok {
			label = v
		}
		fmt.Fprintf(b, `<span data-type="wikilink" data-id="%s" data-ref-type="%s" data-label="%s">%s</span>`,
			html.EscapeString(dataID), html.EscapeString(refType),
			html.EscapeString(label), html.EscapeString(label))
	default:
		// Unknown inline: emit children.
		for _, c := range n.Content {
			writeInlineNodeHTML(b, c)
		}
	}
}

// renderMarkTags returns the opening and closing tag strings for a mark stack.
// Marks open in slice order and close in reverse — matches PM/HTML nesting.
func renderMarkTags(marks []Mark) (string, string) {
	if len(marks) == 0 {
		return "", ""
	}
	var open, close strings.Builder
	closeStack := make([]string, 0, len(marks))
	for _, m := range marks {
		switch m.Type {
		case "bold":
			open.WriteString("<strong>")
			closeStack = append(closeStack, "</strong>")
		case "italic":
			open.WriteString("<em>")
			closeStack = append(closeStack, "</em>")
		case "code":
			open.WriteString("<code>")
			closeStack = append(closeStack, "</code>")
		case "strike":
			open.WriteString("<s>")
			closeStack = append(closeStack, "</s>")
		case "link":
			href := ""
			if m.Attrs != nil {
				if v, ok := m.Attrs["href"].(string); ok {
					href = v
				}
			}
			fmt.Fprintf(&open, `<a href="%s">`, html.EscapeString(href))
			closeStack = append(closeStack, "</a>")
		}
	}
	for i := len(closeStack) - 1; i >= 0; i-- {
		close.WriteString(closeStack[i])
	}
	return open.String(), close.String()
}

// ExtractRefIDs walks a doc tree and returns all wikilink target IDs (int64).
// Replaces the HTML-regex approach in store.go once notes_doc is the source.
func ExtractRefIDs(jsonStr string) []int64 {
	n, err := UnmarshalDoc(jsonStr)
	if err != nil {
		return nil
	}
	var ids []int64
	seen := map[int64]bool{}
	var walk func(Node)
	walk = func(n Node) {
		if n.Type == "wikilink" {
			if v, ok := n.Attrs["id"]; ok {
				var id int64
				switch x := v.(type) {
				case float64:
					id = int64(x)
				case int64:
					id = x
				case int:
					id = int64(x)
				}
				if id > 0 && !seen[id] {
					ids = append(ids, id)
					seen[id] = true
				}
			}
		}
		for _, c := range n.Content {
			walk(c)
		}
	}
	walk(n)
	return ids
}
