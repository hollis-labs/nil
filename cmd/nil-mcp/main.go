package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hollis-labs/nil/config"
)

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

	srv := NewServer(base, apiKey)
	srv.Run(os.Stdin, os.Stdout)
}
