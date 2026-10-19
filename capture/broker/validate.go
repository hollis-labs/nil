package broker

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func textOK(s string, limit int, required bool) bool {
	return utf8.ValidString(s) && len(s) <= limit && !strings.ContainsFunc(s, unicode.IsControl) && (!required || strings.TrimSpace(s) != "")
}

// ValidateRequest mirrors the published v1 limits without silently dropping
// unsupported user metadata. Final encoded bytes are persisted before sending.
func ValidateRequest(r Request, c Capabilities) ([]byte, error) {
	if !slices.Contains(c.MetadataVersions, 1) {
		return nil, errors.New("basalisk server update required: capture metadata v1 is unsupported; draft retained")
	}
	if !ValidID(r.CaptureID) || (r.ContextBindingID != "" && !ValidID(r.ContextBindingID)) || !textOK(r.IntegrationID, 100, true) || !utf8.ValidString(r.Content) || strings.TrimSpace(r.Content) == "" || strings.ContainsRune(r.Content, 0) || len(r.Content) > 65536 || len(r.Tags) > 32 {
		return nil, errors.New("capture exceeds v1 content, identity or tag limits; draft retained")
	}
	for _, tag := range r.Tags {
		if !textOK(tag, 100, true) {
			return nil, errors.New("capture tag exceeds v1 limits; draft retained")
		}
	}
	draft, e1 := time.Parse(time.RFC3339Nano, r.DraftAt)
	submitted, e2 := time.Parse(time.RFC3339Nano, r.SubmittedAt)
	if e1 != nil || e2 != nil || draft.After(submitted) || submitted.After(time.Now().Add(30*time.Second)) {
		return nil, errors.New("capture timestamps require clock repair; draft retained")
	}
	m := r.Metadata
	if m.SchemaVersion != 1 || !textOK(m.Client.Name, 64, true) || !textOK(m.Client.Version, 64, true) || !textOK(m.Title, 200, false) || !slices.Contains([]string{"", "note", "todo", "scratch"}, m.Kind) || !slices.Contains([]string{"", "A", "B", "C"}, m.Priority) {
		return nil, errors.New("capture metadata exceeds v1 limits; draft retained")
	}
	if m.DueDate != "" {
		if _, err := time.Parse("2006-01-02", m.DueDate); err != nil {
			return nil, errors.New("capture due date must be a valid YYYY-MM-DD date")
		}
	}
	for _, values := range [][]string{m.Projects, m.Contexts} {
		if len(values) > 32 {
			return nil, errors.New("capture has too many projects or contexts")
		}
		for _, value := range values {
			if !textOK(value, 100, true) {
				return nil, errors.New("capture project/context exceeds v1 limits")
			}
		}
	}
	if len(m.References) > 32 {
		return nil, errors.New("capture has too many references")
	}
	for _, ref := range m.References {
		if !slices.Contains([]string{"mention", "link"}, ref.Relation) || !textOK(ref.Provider, 64, true) || !textOK(ref.Instance, 200, true) || !textOK(ref.Kind, 64, true) || !textOK(ref.ID, 200, true) || !textOK(ref.Revision, 200, false) || !textOK(ref.Label, 200, false) {
			return nil, errors.New("capture reference exceeds v1 limits")
		}
	}
	if p := m.Preset; p != nil {
		if !textOK(p.ID, 100, true) || !textOK(p.Revision, 100, true) || !slices.Contains([]string{"manual", "context-suggestion", "context-rule"}, p.AppliedBy) {
			return nil, errors.New("capture preset exceeds v1 limits")
		}
	}
	metadata, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	if len(metadata) > 16384 {
		return nil, errors.New("capture metadata exceeds 16 KiB")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	if len(data) > 131072 {
		return nil, errors.New("capture request exceeds 128 KiB")
	}
	if c.Limits.ContentBytes <= 0 || c.Limits.RequestBytes <= 0 || c.Limits.MetadataBytes <= 0 || c.Limits.Tags <= 0 || c.Limits.References <= 0 {
		return nil, errors.New("basalisk server update required: capture limits unavailable")
	}
	if len(r.Content) > c.Limits.ContentBytes || len(data) > c.Limits.RequestBytes || len(metadata) > c.Limits.MetadataBytes || len(r.Tags) > c.Limits.Tags || len(m.References) > c.Limits.References {
		return nil, errors.New("capture exceeds advertised Basalisk limits; metadata has been retained")
	}
	return data, nil
}
