package ingest

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkdownToDoc parses CommonMark markdown into PM JSON. The shape targets
// what TipTap's StarterKit + tiptap-markdown extension produces, so the
// frontend editor can load the result without normalization on first save.
func MarkdownToDoc(md string) (string, error) {
	if md == "" {
		return MarshalDoc(EmptyDoc())
	}
	source := []byte(md)
	reader := text.NewReader(source)
	root := goldmark.DefaultParser().Parse(reader)

	doc := Node{Type: "doc"}
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		n, ok := mdBlockToNode(child, source)
		if ok {
			doc.Content = append(doc.Content, n)
		}
	}
	if len(doc.Content) == 0 {
		doc.Content = []Node{{Type: "paragraph"}}
	}
	return MarshalDoc(doc)
}

func mdBlockToNode(n ast.Node, src []byte) (Node, bool) {
	switch v := n.(type) {
	case *ast.Heading:
		return Node{
			Type:    "heading",
			Attrs:   map[string]any{"level": v.Level},
			Content: mdInlineChildren(v, src),
		}, true
	case *ast.Paragraph:
		return Node{
			Type:    "paragraph",
			Content: mdInlineChildren(v, src),
		}, true
	case *ast.TextBlock:
		// Goldmark emits TextBlock (instead of Paragraph) for inline content
		// inside tight list items; TipTap-side it's still a paragraph.
		return Node{
			Type:    "paragraph",
			Content: mdInlineChildren(v, src),
		}, true
	case *ast.ThematicBreak:
		return Node{Type: "horizontalRule"}, true
	case *ast.Blockquote:
		var content []Node
		for c := v.FirstChild(); c != nil; c = c.NextSibling() {
			if n, ok := mdBlockToNode(c, src); ok {
				content = append(content, n)
			}
		}
		return Node{Type: "blockquote", Content: content}, true
	case *ast.List:
		listType := "bulletList"
		attrs := map[string]any{}
		if v.IsOrdered() {
			listType = "orderedList"
			if v.Start > 1 {
				attrs["start"] = v.Start
			}
		}
		var items []Node
		for c := v.FirstChild(); c != nil; c = c.NextSibling() {
			li, ok := c.(*ast.ListItem)
			if !ok {
				continue
			}
			items = append(items, mdListItemToNode(li, src))
		}
		out := Node{Type: listType, Content: items}
		if len(attrs) > 0 {
			out.Attrs = attrs
		}
		return out, true
	case *ast.FencedCodeBlock:
		var buf bytes.Buffer
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			buf.Write(seg.Value(src))
		}
		attrs := map[string]any{}
		if lang := v.Language(src); len(lang) > 0 {
			attrs["language"] = string(lang)
		} else {
			attrs["language"] = nil
		}
		text := strings.TrimRight(buf.String(), "\n")
		var content []Node
		if text != "" {
			content = []Node{{Type: "text", Text: text}}
		}
		return Node{Type: "codeBlock", Attrs: attrs, Content: content}, true
	case *ast.CodeBlock:
		var buf bytes.Buffer
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			buf.Write(seg.Value(src))
		}
		text := strings.TrimRight(buf.String(), "\n")
		var content []Node
		if text != "" {
			content = []Node{{Type: "text", Text: text}}
		}
		return Node{
			Type:    "codeBlock",
			Attrs:   map[string]any{"language": nil},
			Content: content,
		}, true
	case *ast.HTMLBlock:
		// Pass through as a paragraph containing the raw HTML — TipTap will
		// re-parse it when html: true is enabled. Edge case; rare in practice.
		var buf bytes.Buffer
		lines := v.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			buf.Write(seg.Value(src))
		}
		text := strings.TrimSpace(buf.String())
		if text == "" {
			return Node{}, false
		}
		return Node{Type: "paragraph", Content: []Node{{Type: "text", Text: text}}}, true
	}
	return Node{}, false
}

func mdListItemToNode(li *ast.ListItem, src []byte) Node {
	var content []Node
	for c := li.FirstChild(); c != nil; c = c.NextSibling() {
		if n, ok := mdBlockToNode(c, src); ok {
			content = append(content, n)
		}
	}
	// TipTap requires listItem.content to be block nodes (typically paragraph).
	// Goldmark sometimes emits a TextBlock (which we treat as paragraph) directly.
	if len(content) == 0 {
		content = []Node{{Type: "paragraph"}}
	}
	return Node{Type: "listItem", Content: content}
}

// mdInlineChildren walks the inline children of a block node, producing a
// flat slice of inline nodes (text + hardBreak), each with the appropriate
// marks applied.
func mdInlineChildren(parent ast.Node, src []byte) []Node {
	var out []Node
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		out = appendInline(out, c, src, nil)
	}
	return out
}

func appendInline(out []Node, n ast.Node, src []byte, marks []Mark) []Node {
	switch v := n.(type) {
	case *ast.Text:
		seg := v.Segment
		txt := string(seg.Value(src))
		if txt == "" {
			return out
		}
		node := Node{Type: "text", Text: txt}
		if len(marks) > 0 {
			node.Marks = append([]Mark(nil), marks...)
		}
		out = append(out, node)
		if v.HardLineBreak() {
			out = append(out, Node{Type: "hardBreak"})
		}
		return out
	case *ast.String:
		txt := string(v.Value)
		if txt == "" {
			return out
		}
		node := Node{Type: "text", Text: txt}
		if len(marks) > 0 {
			node.Marks = append([]Mark(nil), marks...)
		}
		return append(out, node)
	case *ast.CodeSpan:
		var buf bytes.Buffer
		for c := v.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Segment.Value(src))
			}
		}
		txt := buf.String()
		if txt == "" {
			return out
		}
		newMarks := append([]Mark(nil), marks...)
		newMarks = append(newMarks, Mark{Type: "code"})
		return append(out, Node{Type: "text", Text: txt, Marks: newMarks})
	case *ast.Emphasis:
		markType := "italic"
		if v.Level >= 2 {
			markType = "bold"
		}
		newMarks := append([]Mark(nil), marks...)
		newMarks = append(newMarks, Mark{Type: markType})
		for c := v.FirstChild(); c != nil; c = c.NextSibling() {
			out = appendInline(out, c, src, newMarks)
		}
		return out
	case *ast.Link:
		linkMark := Mark{
			Type:  "link",
			Attrs: map[string]any{"href": string(v.Destination)},
		}
		if len(v.Title) > 0 {
			linkMark.Attrs["title"] = string(v.Title)
		}
		newMarks := append([]Mark(nil), marks...)
		newMarks = append(newMarks, linkMark)
		for c := v.FirstChild(); c != nil; c = c.NextSibling() {
			out = appendInline(out, c, src, newMarks)
		}
		return out
	case *ast.AutoLink:
		txt := string(v.URL(src))
		if txt == "" {
			return out
		}
		newMarks := append([]Mark(nil), marks...)
		newMarks = append(newMarks, Mark{Type: "link", Attrs: map[string]any{"href": txt}})
		return append(out, Node{Type: "text", Text: txt, Marks: newMarks})
	case *ast.RawHTML:
		// Skip raw inline HTML in markdown — TipTap-managed content shouldn't carry it.
		return out
	}
	// Unknown inline: walk children with current marks.
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		out = appendInline(out, c, src, marks)
	}
	return out
}
