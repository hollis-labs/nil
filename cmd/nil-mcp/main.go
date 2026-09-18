package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	gomcp "github.com/hollis-labs/go-mcp/server"

	"github.com/hollis-labs/nil/config"
)

const serverVersion = "1.0.0"

func main() {
	cfg := config.LoadOrDefault()
	cfg.EnsureDefaults()

	port := cfg.APIPort
	apiKey := cfg.APIKey

	fmt.Fprintf(os.Stderr, "nil-mcp: connecting to NIL API on port %d\n", port)

	// Optional health check — warn but don't abort.
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	hc := &http.Client{Timeout: 2 * time.Second}
	resp, err := hc.Get(base + "/api/v1/vaults")
	if err != nil {
		fmt.Fprintf(os.Stderr, "nil-mcp: warning: NIL API not reachable (%v) — make sure NIL is running with API enabled\n", err)
	} else {
		_ = resp.Body.Close()
	}

	client := newAPIClient(base, apiKey)

	srv := gomcp.NewServer("nil-mcp", serverVersion, gomcp.WithInstructions(
		"NIL personal task and note manager. Use nil_list_vaults to discover available vaults. "+
			"Pass vault_id in tool calls to target a specific vault; omit to use the active vault. "+
			"The inbox is a fast-capture area — use nil_create_inbox to add items without taxonomy friction, "+
			"then nil_process_inbox when ready to promote them.",
	))
	registerTools(srv, client)

	if err := srv.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "nil-mcp: server error: %v\n", err)
		os.Exit(1)
	}
}
