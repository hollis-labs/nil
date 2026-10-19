package capture

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/hollis-labs/nil/parse"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type QuickFields struct {
	Title         string   `json:"title"`
	Projects      []string `json:"projects"`
	Contexts      []string `json:"contexts"`
	Tags          []string `json:"tags"`
	Priority      *string  `json:"priority"`
	DueDate       *string  `json:"dueDate"`
	ThresholdDate *string  `json:"thresholdDate"`
}

// ParseQuickAdd applies legacy token semantics only to the dedicated input.
// Bare @desk stays a context there; the body is always literal Markdown.
func ParseQuickAdd(input string) QuickFields {
	p := parse.ParseLine(input)
	return QuickFields{p.Title, p.Projects, p.Contexts, p.Tags, p.Priority, p.Due, p.Thresh}
}

type Reference struct {
	Relation string `json:"relation"`
	Provider string `json:"provider"`
	Instance string `json:"instance"`
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Revision string `json:"revision"`
	Label    string `json:"label"`
}

type Suggestion struct {
	Reference Reference `json:"reference"`
	URL       string    `json:"url"`
}

// ItemSuggestion is the local Nil adapter's versioned Markdown link encoding:
// nil://vault/<escaped vault ID>/item/<positive item ID>?revision=<updated_at>.
// It denotes a local item reference, never permission, remote fetch, or an
// assertion that a requested historical revision is still locally available.
func ItemSuggestion(vaultID string, id int64, revision, label string) Suggestion {
	u := "nil://vault/" + url.PathEscape(vaultID) + "/item/" + strconv.FormatInt(id, 10) + "?revision=" + url.QueryEscape(revision)
	return Suggestion{Reference{"mention", "nil", vaultID, "item", strconv.FormatInt(id, 10), revision, label}, u}
}

// References derives identities from actual Markdown links on every read. Code,
// raw @tokens, image URLs, labels and arbitrary web links cannot create refs.
// Editing/deleting a link updates this projection automatically.
func References(markdown string) []Reference {
	source := []byte(markdown)
	root := goldmark.DefaultParser().Parse(text.NewReader(source))
	refs := []Reference{}
	seen := map[string]bool{}
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		link, ok := n.(*ast.Link)
		if !entering || !ok {
			return ast.WalkContinue, nil
		}
		ref, valid := localReference(link, source)
		if valid {
			key := ref.Instance + "/" + ref.ID + "/" + ref.Revision
			if !seen[key] {
				refs = append(refs, ref)
				seen[key] = true
			}
		}
		return ast.WalkContinue, nil
	})
	return refs
}

func localReference(link *ast.Link, source []byte) (Reference, bool) {
	u, err := url.Parse(string(link.Destination))
	if err != nil || u.Scheme != "nil" || u.Host != "vault" || u.User != nil || u.Fragment != "" {
		return Reference{}, false
	}
	parts := strings.Split(strings.TrimPrefix(u.EscapedPath(), "/"), "/")
	if len(parts) != 3 || parts[1] != "item" {
		return Reference{}, false
	}
	instance, err := url.PathUnescape(parts[0])
	if err != nil || instance == "" || len(instance) > 200 || strings.ContainsAny(instance, "\x00\r\n") {
		return Reference{}, false
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || id <= 0 {
		return Reference{}, false
	}
	revision := u.Query().Get("revision")
	if len(revision) > 200 || strings.ContainsAny(revision, "\x00\r\n") {
		return Reference{}, false
	}
	var label strings.Builder
	_ = ast.Walk(link, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch v := n.(type) {
			case *ast.Text:
				label.Write(v.Value(source))
			case *ast.String:
				label.Write(v.Value)
			}
		}
		return ast.WalkContinue, nil
	})
	return Reference{"mention", "nil", instance, "item", strconv.FormatInt(id, 10), revision, label.String()}, true
}
