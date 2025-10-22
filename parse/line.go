package parse

import (
	"regexp"
	"strings"
)

var (
	rePriority = regexp.MustCompile(`^\(([A-Z])\)\s+`)
	reProject  = regexp.MustCompile(`\+\w[\w\-_.]*`)
	reContext  = regexp.MustCompile(`@\w[\w\-_.]*`)
	reTag      = regexp.MustCompile(`#\w[\w\-_.]*`)
	reKVDate   = regexp.MustCompile(`\b(due|t):(\d{4}-\d{2}-\d{2})\b`)
)

type Parsed struct {
	Title    string
	Priority *string
	Projects []string
	Contexts []string
	Tags     []string
	Due      *string
	Thresh   *string
}

func unique(in []string) []string {
	m := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !m[s] {
			m[s] = true
			out = append(out, s)
		}
	}
	return out
}

func stripPrefixAll(in []string, prefix string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, strings.TrimPrefix(s, prefix))
	}
	return out
}

func ParseLine(line string) Parsed {
	src := strings.TrimSpace(line)
	var prio *string
	if m := rePriority.FindStringSubmatch(src); len(m) == 2 {
		p := m[1]
		prio = &p
		src = rePriority.ReplaceAllString(src, "")
	}

	projects := unique(stripPrefixAll(reProject.FindAllString(src, -1), "+"))
	contexts := unique(stripPrefixAll(reContext.FindAllString(src, -1), "@"))
	tags := unique(stripPrefixAll(reTag.FindAllString(src, -1), "#"))

	var due, thr *string
	for _, m := range reKVDate.FindAllStringSubmatch(src, -1) {
		k, v := m[1], m[2]
		if k == "due" {
			d := v; due = &d
		} else {
			t := v; thr = &t
		}
		src = strings.Replace(src, m[0], "", 1)
	}
	// remove tokens from title
	src = reProject.ReplaceAllString(src, "")
	src = reContext.ReplaceAllString(src, "")
	src = reTag.ReplaceAllString(src, "")
	title := strings.TrimSpace(src)

	return Parsed{Title: title, Priority: prio, Projects: projects, Contexts: contexts, Tags: tags, Due: due, Thresh: thr}
}
