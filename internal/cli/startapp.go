package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwittrup/responder-cli/internal/api"
	"github.com/spf13/cobra"
)

const defaultStartAppTimeout = 5 * time.Minute

type startAppOptions struct {
	checkoutDir string
	apiURL      string
	timeout     time.Duration
}

func newStartAppCmd() *cobra.Command {
	var opts startAppOptions
	cmd := &cobra.Command{
		Use:   "start-app",
		Short: "Start the local prepared911 API stack",
		Long: `Start the local prepared911 API stack using prepared911's own
recipes, then wait until the API health endpoint reports healthy.

Only the API stack is started; the portal UI is left alone.
Requires prepared911's own prerequisites (OrbStack,
1Password-provided secrets, AWS credentials).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return startApp(cmd, opts)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.checkoutDir, "prepared911-dir", defaultCheckout, "prepared911 checkout holding the just recipes (env PREPARED911_DIR)")
	f.StringVar(&opts.apiURL, "api-url", defaultAPIURL, "API base URL to poll for health (env RESPONDER_API_URL)")
	f.DurationVar(&opts.timeout, "timeout", 0, "how long to wait for a healthy API (env RESPONDER_START_APP_TIMEOUT, default 5m)")
	return cmd
}

func startApp(cmd *cobra.Command, opts startAppOptions) error {
	changed := func(name string) bool { return cmd.Flags().Changed(name) }
	if v, ok := envOrUnset("PREPARED911_DIR", changed("prepared911-dir")); ok {
		opts.checkoutDir = v
	}
	if v, ok := envOrUnset("RESPONDER_API_URL", changed("api-url")); ok {
		opts.apiURL = v
	}
	if !changed("timeout") {
		opts.timeout = parseDurationEnv("RESPONDER_START_APP_TIMEOUT", defaultStartAppTimeout)
	}
	if opts.timeout <= 0 {
		opts.timeout = defaultStartAppTimeout
	}

	checkoutDir := expandHome(opts.checkoutDir)
	info, err := os.Stat(checkoutDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("prepared911 checkout not found at %s (set PREPARED911_DIR or --prepared911-dir)", checkoutDir)
	}
	if _, err := os.Stat(filepath.Join(checkoutDir, "justfile")); err != nil {
		return fmt.Errorf("no just recipes found in %s; is that a prepared911 checkout?", checkoutDir)
	}
	if _, err := exec.LookPath("just"); err != nil {
		return fmt.Errorf("the `just` command was not found on PATH; install it to run the prepared911 recipes")
	}

	cmd.Printf("Starting API stack with `just api run` in %s...\n", checkoutDir)
	just := exec.Command("just", "api", "run")
	just.Dir = checkoutDir
	var out bytes.Buffer
	just.Stdout = &out
	just.Stderr = &out
	if err := just.Run(); err != nil {
		return fmt.Errorf("`just api run` failed in %s: %v\n%s", checkoutDir, err, strings.TrimSpace(out.String()))
	}

	cmd.Printf("Waiting for API health at %s...\n", opts.apiURL)
	client := api.NewClient(opts.apiURL, "", 10*time.Second)
	deadline := time.Now().Add(opts.timeout)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := client.CheckHealth(ctx)
		cancel()
		if err == nil {
			cmd.Printf("API is healthy at %s\n", opts.apiURL)
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("API at %s did not become healthy within %s", opts.apiURL, opts.timeout)
		}
		time.Sleep(time.Second)
	}
}
