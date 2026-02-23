package chat

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ComputeVaultProfile builds a concise vault snapshot string for injection into the
// system prompt. Returns an empty string if the store is nil or any query fails.
func ComputeVaultProfile(ctx context.Context, st BridgeStore) string {
	if st == nil {
		return ""
	}
	stats, err := st.GetStats(ctx)
	if err != nil {
		return ""
	}
	projects, contexts, _, err := st.GetTaxonomyWithCounts(ctx)
	if err != nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Vault snapshot (%s):\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("- %d open todos · %d notes · %d overdue · %d inbox\n",
		stats.Open, stats.Notes, stats.Overdue, stats.Inbox))

	if len(projects) > 0 {
		top := projects
		if len(top) > 5 {
			top = top[:5]
		}
		names := make([]string, len(top))
		for i, p := range top {
			names[i] = "+" + p.Name
		}
		sb.WriteString("- Projects: " + strings.Join(names, ", ") + "\n")
	}

	if len(contexts) > 0 {
		top := contexts
		if len(top) > 5 {
			top = top[:5]
		}
		names := make([]string, len(top))
		for i, c := range top {
			names[i] = "@" + c.Name
		}
		sb.WriteString("- Contexts: " + strings.Join(names, ", "))
	}

	return sb.String()
}
