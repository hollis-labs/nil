package ingest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarkdownToDocBasicShapes(t *testing.T) {
	cases := []struct {
		name           string
		md             string
		wantBlockTypes []string
	}{
		{"paragraph", "hello world", []string{"paragraph"}},
		{"heading", "# title", []string{"heading"}},
		{"list", "- one\n- two", []string{"bulletList"}},
		{"ordered", "1. one\n2. two", []string{"orderedList"}},
		{"code", "```go\nfmt.Println(\"x\")\n```", []string{"codeBlock"}},
		{"hr", "---", []string{"horizontalRule"}},
		{"quote", "> quoted", []string{"blockquote"}},
		{"mixed", "# t\n\npara\n\n- item", []string{"heading", "paragraph", "bulletList"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			docJSON, err := MarkdownToDoc(c.md)
			if err != nil {
				t.Fatalf("MarkdownToDoc(%q): %v", c.md, err)
			}
			var d Node
			if err := json.Unmarshal([]byte(docJSON), &d); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if d.Type != "doc" {
				t.Errorf("root type = %q, want doc", d.Type)
			}
			if len(d.Content) != len(c.wantBlockTypes) {
				t.Fatalf("block count = %d, want %d (got %s)", len(d.Content), len(c.wantBlockTypes), docJSON)
			}
			for i, want := range c.wantBlockTypes {
				if d.Content[i].Type != want {
					t.Errorf("block[%d] = %q, want %q", i, d.Content[i].Type, want)
				}
			}
		})
	}
}

func TestMarkdownInlineMarks(t *testing.T) {
	docJSON, err := MarkdownToDoc("a **bold** and *italic* and `code` and [link](http://x)")
	if err != nil {
		t.Fatal(err)
	}
	var d Node
	if err := json.Unmarshal([]byte(docJSON), &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Content) != 1 || d.Content[0].Type != "paragraph" {
		t.Fatalf("expected single paragraph, got %s", docJSON)
	}
	wantMarks := map[string]string{
		"bold":   "bold",
		"italic": "italic",
		"code":   "code",
		"link":   "link",
	}
	found := map[string]bool{}
	for _, n := range d.Content[0].Content {
		for _, m := range n.Marks {
			if want, ok := wantMarks[n.Text]; ok && m.Type == want {
				found[n.Text] = true
			}
		}
	}
	for k := range wantMarks {
		if !found[k] {
			t.Errorf("missing mark for %q (got %s)", k, docJSON)
		}
	}
}

func TestHTMLToDocBasicShapes(t *testing.T) {
	cases := []struct {
		name string
		html string
		want []string
	}{
		{"para", "<p>hello</p>", []string{"paragraph"}},
		{"h2", "<h2>title</h2>", []string{"heading"}},
		{"ul", "<ul><li>one</li><li>two</li></ul>", []string{"bulletList"}},
		{"ol", "<ol><li>one</li></ol>", []string{"orderedList"}},
		{"code", `<pre><code class="language-go">x</code></pre>`, []string{"codeBlock"}},
		{"hr", "<hr>", []string{"horizontalRule"}},
		{"quote", "<blockquote><p>q</p></blockquote>", []string{"blockquote"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			docJSON, err := HTMLToDoc(c.html)
			if err != nil {
				t.Fatalf("HTMLToDoc: %v", err)
			}
			var d Node
			if err := json.Unmarshal([]byte(docJSON), &d); err != nil {
				t.Fatal(err)
			}
			if len(d.Content) != len(c.want) {
				t.Fatalf("got %d blocks (%s), want %d", len(d.Content), docJSON, len(c.want))
			}
			for i, w := range c.want {
				if d.Content[i].Type != w {
					t.Errorf("block[%d] = %q, want %q", i, d.Content[i].Type, w)
				}
			}
		})
	}
}

func TestWikilinkRoundTrip(t *testing.T) {
	in := `<p>hello <span data-type="wikilink" data-id="42" data-ref-type="note" data-label="Friend">Friend</span> there</p>`
	docJSON, err := HTMLToDoc(in)
	if err != nil {
		t.Fatal(err)
	}
	ids := ExtractRefIDs(docJSON)
	if len(ids) != 1 || ids[0] != 42 {
		t.Errorf("ExtractRefIDs = %v, want [42]", ids)
	}
	rendered, err := DocToHTML(docJSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-type="wikilink"`,
		`data-id="42"`,
		`data-ref-type="note"`,
		`Friend`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered HTML missing %q: %s", want, rendered)
		}
	}
}

func TestDocToPlainText(t *testing.T) {
	md := "# Title\n\nA paragraph with **bold** text.\n\n- list item 1\n- list item 2"
	docJSON, err := MarkdownToDoc(md)
	if err != nil {
		t.Fatal(err)
	}
	text, err := DocToPlainText(docJSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Title", "paragraph with bold text", "list item 1", "list item 2"} {
		if !strings.Contains(text, want) {
			t.Errorf("plain text missing %q: %q", want, text)
		}
	}
}

func TestEmptyInputProducesEmptyDoc(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\n"} {
		md, err := MarkdownToDoc(in)
		if err != nil {
			t.Fatalf("MarkdownToDoc(%q): %v", in, err)
		}
		ht, err := HTMLToDoc(in)
		if err != nil {
			t.Fatalf("HTMLToDoc(%q): %v", in, err)
		}
		for _, j := range []string{md, ht} {
			var d Node
			if err := json.Unmarshal([]byte(j), &d); err != nil {
				t.Fatal(err)
			}
			if d.Type != "doc" || len(d.Content) != 1 || d.Content[0].Type != "paragraph" {
				t.Errorf("empty input %q produced %s, want canonical empty doc", in, j)
			}
		}
	}
}
