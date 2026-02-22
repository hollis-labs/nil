package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"nanite/store"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const claudeAPIURL = "https://api.anthropic.com/v1/messages"
const claudeAPIVersion = "2023-06-01"
const defaultModel = "claude-haiku-4-5-20251001"
const defaultMaxTokens = 1024

// systemPromptTemplate is injected per-request with vault/session context substituted.
const systemPromptTemplate = `You are a NANITE vault assistant. NANITE is a personal task and note management app.
You have access to the user's vault via search results that may be included in messages.
Today's date: {date}
Vault: {vault_name} | Mode: {mode}
Capabilities — Read: true | Write: {write_cap} | Delete: {delete_cap}

## Your responsibilities

For SEARCH or READ queries:
- Summarise the relevant items found in the context.
- If no search results are provided, say so and suggest the user try a more specific query.

For MUTATING requests (create / update / delete):
- Never execute mutations directly.
- Always emit a structured JSON action block so the user can review and approve it first.
- Emit the action as a fenced JSON block at the END of your response, like this:

` + "```" + `json
{
  "action": {
    "type": "create",
    "item_type": "todo",
    "vault_id": "{vault_id}",
    "payload": {
      "title": "...",
      "priority": "A",
      "section": "now",
      "projects": [],
      "contexts": [],
      "tags": []
    }
  }
}
` + "```" + `

For updates, include a "diff" field: {"field_name": ["old_value", "new_value"]}.
For deletes, include "payload": {"id": <item_id>, "title": "<title>"}.
For updates, also include the full updated item in "payload".

Always address the user's intent first in plain text, then emit the action block.
If the vault does not have write or delete capability, explain that and do not emit an action.`

// VaultCaps describes what chat operations are permitted on a vault.
type VaultCaps struct {
	Read   bool
	Write  bool
	Delete bool
}

// BridgeRequest carries everything needed for one LLM call.
type BridgeRequest struct {
	APIKey      string
	Model       string // optional; falls back to defaultModel
	VaultID     string
	VaultName   string
	Caps        VaultCaps
	DryRun      bool
	History     []ChatMessage // prior turns (for context window)
	UserMessage string
	SearchCtx   string // optional pre-serialised search results
}

// Bridge calls the Claude API and parses the response.
// It carries no per-session state; callers manage persistence via ChatStore.
type Bridge struct{}

// Send executes one round-trip to the Claude API.
// If the assistant emits a JSON action block, ChatResponse.Proposal is populated
// (not yet persisted — the caller must call ChatStore.CreateProposal).
func (b *Bridge) Send(ctx context.Context, req BridgeRequest) (*ChatResponse, error) {
	if req.APIKey == "" {
		return &ChatResponse{
			Message: ChatMessage{Role: "assistant", Content: ""},
			Error:   "No API key configured — add one in Settings → Chat",
		}, nil
	}

	model := req.Model
	if model == "" {
		model = defaultModel
	}

	systemPrompt := buildSystemPrompt(req)

	type apiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var msgs []apiMsg

	// Include up to the last 20 non-system turns for context.
	start := 0
	if len(req.History) > 20 {
		start = len(req.History) - 20
	}
	for _, m := range req.History[start:] {
		if m.Role == "system" {
			continue
		}
		msgs = append(msgs, apiMsg{Role: m.Role, Content: m.Content})
	}

	// Append the current user message, injecting search results if present.
	userContent := req.UserMessage
	if req.SearchCtx != "" {
		userContent = req.UserMessage + "\n\n--- Vault search results ---\n" + req.SearchCtx
	}
	msgs = append(msgs, apiMsg{Role: "user", Content: userContent})

	body, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": defaultMaxTokens,
		"system":     systemPrompt,
		"messages":   msgs,
	})
	if err != nil {
		return nil, fmt.Errorf("chat/bridge: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("chat/bridge: build http request: %w", err)
	}
	httpReq.Header.Set("x-api-key", req.APIKey)
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)
	httpReq.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("chat/bridge: api call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("chat/bridge: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			return &ChatResponse{
				Message: ChatMessage{Role: "assistant", Content: ""},
				Error:   fmt.Sprintf("API error (%d): %s", resp.StatusCode, apiErr.Error.Message),
			}, nil
		}
		return &ChatResponse{
			Message: ChatMessage{Role: "assistant", Content: ""},
			Error:   fmt.Sprintf("API returned status %d", resp.StatusCode),
		}, nil
	}

	// Parse Claude response envelope.
	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("chat/bridge: parse api response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("chat/bridge: empty response from API")
	}

	rawText := apiResp.Content[0].Text
	proposal, cleanText := extractAction(rawText, req.VaultID)

	return &ChatResponse{
		Message:  ChatMessage{Role: "assistant", Content: cleanText},
		Proposal: proposal,
	}, nil
}

// buildSystemPrompt substitutes placeholders into the system prompt template.
func buildSystemPrompt(req BridgeRequest) string {
	mode := "live"
	if req.DryRun {
		mode = "DRY RUN — proposals only, no execution"
	}
	writeCap, deleteCap := "false", "false"
	if req.Caps.Write {
		writeCap = "true"
	}
	if req.Caps.Delete {
		deleteCap = "true"
	}
	p := systemPromptTemplate
	p = strings.ReplaceAll(p, "{vault_name}", req.VaultName)
	p = strings.ReplaceAll(p, "{vault_id}", req.VaultID)
	p = strings.ReplaceAll(p, "{mode}", mode)
	p = strings.ReplaceAll(p, "{write_cap}", writeCap)
	p = strings.ReplaceAll(p, "{delete_cap}", deleteCap)
	p = strings.ReplaceAll(p, "{date}", time.Now().Format("2006-01-02"))
	return p
}

// jsonBlockRe matches a fenced ```json ... ``` block (non-greedy, dot-matches-newline).
var jsonBlockRe = regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")

// actionEnvelope mirrors the JSON structure the LLM emits for mutating actions.
type actionEnvelope struct {
	Action struct {
		Type     string         `json:"type"`
		ItemType string         `json:"item_type"`
		VaultID  string         `json:"vault_id"`
		Payload  store.Item     `json:"payload"`
		Diff     map[string]any `json:"diff"`
	} `json:"action"`
}

// extractAction scans rawText for an embedded JSON action block.
// Returns the parsed ActionProposal stub (not yet persisted) and the cleaned text.
func extractAction(rawText, vaultID string) (*ActionProposal, string) {
	matches := jsonBlockRe.FindStringSubmatch(rawText)
	if matches == nil {
		return nil, rawText
	}
	jsonStr := matches[1]
	cleanText := strings.TrimSpace(jsonBlockRe.ReplaceAllString(rawText, ""))

	var env actionEnvelope
	if err := json.Unmarshal([]byte(jsonStr), &env); err != nil || env.Action.Type == "" {
		// Malformed or missing action — treat the whole response as plain text.
		return nil, rawText
	}

	resolvedVaultID := env.Action.VaultID
	if resolvedVaultID == "" {
		resolvedVaultID = vaultID
	}

	proposal := &ActionProposal{
		ActionType: env.Action.Type,
		ItemType:   env.Action.ItemType,
		VaultID:    resolvedVaultID,
		Payload:    env.Action.Payload,
		Diff:       env.Action.Diff,
		Status:     "pending",
	}
	return proposal, cleanText
}
