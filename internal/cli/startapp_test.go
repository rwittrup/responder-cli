package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkoutWithJustfile creates a temporary prepared911 checkout holding
// a justfile so start-app finds its recipes.
func checkoutWithJustfile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "justfile"), []byte("# fake justfile\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// stubJust puts a fake `just` executable first on PATH. It records its
// arguments in markerFile and exits with the given code.
func stubJust(t *testing.T, exitCode int, markerFile string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\necho \"$@\" > " + markerFile + "\nexit " + itoa(exitCode) + "\n"
	path := filepath.Join(dir, "just")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return "1"
}

func TestStartAppSuccess(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	checkout := checkoutWithJustfile(t)
	marker := filepath.Join(t.TempDir(), "just-args")

	stubJust(t, 0, marker)
	res := runCommand(t, []string{"start-app"}, map[string]string{
		"RESPONDER_API_URL": apiURL,
		"PREPARED911_DIR":   checkout,
	})
	requireSuccess(t, res)

	raw, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("stub just was not invoked: %v\nstdout:\n%s\nstderr:\n%s", err, res.stdout, res.stderr)
	}
	if strings.TrimSpace(string(raw)) != "api run" {
		t.Fatalf("expected `just api run`, got %q", strings.TrimSpace(string(raw)))
	}
	if !strings.Contains(strings.ToLower(res.stdout), "healthy") {
		t.Fatalf("expected healthy message, got:\n%s", res.stdout)
	}
}

func TestStartAppMissingCheckout(t *testing.T) {
	stubJust(t, 0, filepath.Join(t.TempDir(), "just-args"))
	res := runCommand(t, []string{"start-app"}, map[string]string{
		"PREPARED911_DIR":   filepath.Join(t.TempDir(), "does-not-exist"),
		"RESPONDER_API_URL": "http://127.0.0.1:3000",
	})
	requireFailure(t, res)
	if !strings.Contains(res.stderr, "checkout not found") {
		t.Fatalf("expected missing-checkout error, got:\n%s", res.stderr)
	}
}

func TestStartAppMissingJustfile(t *testing.T) {
	stubJust(t, 0, filepath.Join(t.TempDir(), "just-args"))
	res := runCommand(t, []string{"start-app"}, map[string]string{
		"PREPARED911_DIR":   t.TempDir(),
		"RESPONDER_API_URL": "http://127.0.0.1:3000",
	})
	requireFailure(t, res)
	if !strings.Contains(res.stderr, "just") {
		t.Fatalf("expected missing-recipes error, got:\n%s", res.stderr)
	}
}

func TestStartAppHealthTimeout(t *testing.T) {
	api := newAPIFake(t)
	api.healthy = false
	apiURL := api.start(t)
	checkout := checkoutWithJustfile(t)

	stubJust(t, 0, filepath.Join(t.TempDir(), "just-args"))
	res := runCommand(t, []string{"start-app"}, map[string]string{
		"RESPONDER_API_URL":           apiURL,
		"PREPARED911_DIR":             checkout,
		"RESPONDER_START_APP_TIMEOUT": "2s",
	})
	requireFailure(t, res)
	if !strings.Contains(strings.ToLower(res.stderr), "healthy") {
		t.Fatalf("expected health-timeout error, got:\n%s", res.stderr)
	}
}

func TestStartAppJustFailure(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	checkout := checkoutWithJustfile(t)

	stubJust(t, 1, filepath.Join(t.TempDir(), "just-args"))
	res := runCommand(t, []string{"start-app"}, map[string]string{
		"RESPONDER_API_URL": apiURL,
		"PREPARED911_DIR":   checkout,
	})
	requireFailure(t, res)
	if !strings.Contains(res.stderr, "just api run") {
		t.Fatalf("expected just-failure error, got:\n%s", res.stderr)
	}
}

func TestStartAppHelp(t *testing.T) {
	res := runCommand(t, []string{"start-app", "--help"}, nil)
	requireSuccess(t, res)
	for _, want := range []string{"--prepared911-dir", "PREPARED911_DIR", "--api-url", "--timeout"} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("expected help to mention %q, got:\n%s", want, res.stdout)
		}
	}
}
