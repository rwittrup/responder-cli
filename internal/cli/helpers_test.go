package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// cmdResult captures an in-process command invocation.
type cmdResult struct {
	stdout string
	stderr string
	code   int
}

// runCommand executes the responder command in-process with the given
// args and environment, returning captured output and exit code.
func runCommand(t *testing.T, args []string, env map[string]string) cmdResult {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
	var stdout, stderr bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	code := Execute(root)
	return cmdResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// apiFake stands in for the prepared911 API (health and GraphQL),
// recording the requests it receives and returning canned responses.
type apiFake struct {
	t              *testing.T
	mu             sync.Mutex
	healthy        bool
	graphqlHandler func(w http.ResponseWriter, r *http.Request)
	graphqlCalls   int
	lastMethod     string
	lastPath       string
	lastAuth       string
	lastBody       string
}

func newAPIFake(t *testing.T) *apiFake {
	t.Helper()
	return &apiFake{t: t, healthy: true}
}

func (f *apiFake) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if f.healthy {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "ok")
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "not healthy")
	})
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.graphqlCalls++
		f.lastMethod = r.Method
		f.lastPath = r.URL.Path
		f.lastAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		f.lastBody = string(body)
		f.mu.Unlock()
		if f.graphqlHandler != nil {
			f.graphqlHandler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"startAudioDemo":{"chatroom":{"id":"chatroom-123"},"audioServerResponseCode":200}}}`)
	})
	return mux
}

func (f *apiFake) start(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return srv.URL
}

func (f *apiFake) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.graphqlCalls
}

// locationFake stands in for the IP geolocation lookup.
func startLocationFake(t *testing.T, lat, lon float64) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","lat":%v,"lon":%v}`, lat, lon)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// checkoutWithToken creates a temporary prepared911 checkout holding a
// fake .env.local with the given token line.
func checkoutWithToken(t *testing.T, tokenLine string) string {
	t.Helper()
	dir := t.TempDir()
	content := "DEVELOPER_EMAIL=dev@example.com\n"
	if tokenLine != "" {
		content += tokenLine + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.local"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func requireSuccess(t *testing.T, res cmdResult) {
	t.Helper()
	if res.code != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout:\n%s\nstderr:\n%s", res.code, res.stdout, res.stderr)
	}
}

func requireFailure(t *testing.T, res cmdResult) {
	t.Helper()
	if res.code == 0 {
		t.Fatalf("expected non-zero exit, got 0\nstdout:\n%s", res.stdout)
	}
	if strings.TrimSpace(res.stderr) == "" {
		t.Fatalf("expected an error on stderr, got empty stderr\nstdout:\n%s", res.stdout)
	}
}
