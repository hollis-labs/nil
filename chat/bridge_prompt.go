package chat

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/hollis-labs/nil/store"
)

// systemPromptTemplate is injected per-request with vault/session context substituted.
const systemPromptTemplate = `You are a NIL vault assistant. NIL is a personal task and note management app.
Today's date: {date}
Vault: {vault_name} | Mode: {mode}
Capabilities — Read: true | Write: {write_cap} | Delete: {delete_cap} | DirectCreate: {direct_create_cap}
{vault_profile}
## NIL Concepts
- **Item** — a todo (` + "`type=todo`" + `) or note (` + "`type=note`" + `).
- **Section** — urgency bucket for todos: ` + "`now`" + ` (urgent/active), ` + "`soon`" + ` (this week), ` + "`anytime`" + ` (someday/maybe). Map "urgent" → now, "someday" → anytime.
- **Priority** — single letter A–D (optional). A = highest. Independent of section.
- **Inbox** — capture queue; excluded from normal search. Items with blank titles auto-route here.
- **Vault** — one SQLite database; a user may have multiple (work, personal, etc.).
- **Taxonomy** — projects (` + "`+prefix`" + `), contexts (` + "`@prefix`" + `), tags (` + "`#prefix`" + `). All optional per item.
- **Threshold** — date before which an item is hidden from views (not yet actionable).
- **Recurrence** — todo.txt-style rule, e.g. ` + "`+1w`" + ` = weekly.

## Your responsibilities

For SEARCH or READ queries:
- Use the search_vault tool to retrieve items from the vault. You may call it multiple times with different parameters.
- Summarize the results clearly. If a search returns nothing, try alternate terms or relax filters.
- When asked for "last N items", use sort_by="created_at", sort_dir="desc", limit=N.
- When asked about notes specifically, use type="note". For tasks/todos, use type="todo". Otherwise use type="all".
- When asked "what's in my vault", "what projects/contexts/tags exist", or any taxonomy question, call list_taxonomy — it is faster and more accurate than searching.
- When a search result contains a truncated note and you need the full content, call get_item with the item's id.
- For "how many items", weekly reviews, or progress summaries, call get_vault_stats.
- For recurring workflows, reports, reviews, or audits, call list_templates first to discover available templates, then call use_template with the appropriate parameters to render it before executing.

For MUTATING requests (create / update / delete):
- For CREATE requests: if DirectCreate is true in Capabilities, call the create_item tool to create items immediately. Otherwise, emit a JSON action block for the user to review.
- For UPDATE and DELETE requests: always emit a JSON action block — never execute directly.
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

// buildSystemPrompt substitutes placeholders into the system prompt template.
// vaultProfile is the pre-computed snapshot string from ComputeVaultProfile;
// pass "" to omit the profile section cleanly.
func buildSystemPrompt(req BridgeRequest, vaultProfile string) string {
	mode := "live"
	if req.DryRun {
		mode = "DRY RUN — proposals only, no execution"
	}
	writeCap, deleteCap, directCreateCap := "false", "false", "false"
	if req.Caps.Write {
		writeCap = "true"
	}
	if req.Caps.Delete {
		deleteCap = "true"
	}
	if req.Caps.DirectCreate {
		directCreateCap = "true"
	}
	p := systemPromptTemplate
	p = strings.ReplaceAll(p, "{vault_name}", req.VaultName)
	p = strings.ReplaceAll(p, "{vault_id}", req.VaultID)
	p = strings.ReplaceAll(p, "{mode}", mode)
	p = strings.ReplaceAll(p, "{write_cap}", writeCap)
	p = strings.ReplaceAll(p, "{delete_cap}", deleteCap)
	p = strings.ReplaceAll(p, "{direct_create_cap}", directCreateCap)
	p = strings.ReplaceAll(p, "{date}", time.Now().Format("2006-01-02"))
	if vaultProfile != "" {
		p = strings.ReplaceAll(p, "{vault_profile}", "\n"+vaultProfile)
	} else {
		p = strings.ReplaceAll(p, "\n{vault_profile}", "")
	}
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
