package cli

// commands_serve.go holds cmdServeAPI in isolation from the rest of the
// commandEnv-native commands: it is conceptually "run the HTTP API surface"
// rather than a CLI item operation, and it's the only command in this
// package that pulls in apiserver as a dependency.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hollis-labs/nil/apiserver"
	"github.com/hollis-labs/nil/config"
)

// cmdServeAPI starts the same HTTP API server the GUI embeds (see
// apiserver.New), but as a standalone, long-running process with no Wails
// dependency. It reads/writes the same config.json and vault registry as
// the GUI and CLI, so it works whether or not the desktop app has ever been
// launched on this machine — EnsureDefaults seeds an API port/key on first
// run just like app.go's startup() does for the GUI.
//
// Blocks until SIGINT/SIGTERM, then shuts the HTTP server down gracefully
// and closes all open vault stores before returning.
func cmdServeAPI(ctx context.Context, args []string, env *commandEnv) {
	fs := flag.NewFlagSet("serve-api", flag.ContinueOnError)
	port := fs.Int("port", 0, "override the configured API port for this run only (not persisted)")
	if _, err := parseInterspersed(args, fs); err != nil {
		die("serve-api: %v", err)
	}

	// Work on a copy so a --port override never mutates env.cfg or gets
	// persisted, but EnsureDefaults (port/key seeding) is saved for real —
	// otherwise a machine that has only ever used the CLI would have no
	// APIKey and every request would be rejected.
	cfg := *env.cfg
	if cfg.EnsureDefaults() {
		if err := config.Save(&cfg); err != nil {
			fmt.Fprintf(os.Stderr, "nil: warning: failed to persist API defaults: %v\n", err)
		}
	}
	if *port != 0 {
		cfg.APIPort = *port
	}

	srv := apiserver.New(&cfg, env.mgr)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	fmt.Fprintf(os.Stderr, "nil: API server listening on %s (Ctrl+C to stop)\n", srv.Addr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			die("serve-api: %v (is NIL already running with the API enabled on this port?)", err)
		}
	case <-sigCh:
		fmt.Fprintln(os.Stderr, "nil: shutting down API server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "nil: warning: API server shutdown error: %v\n", err)
		}
	}

	env.mgr.CloseAll()
}
