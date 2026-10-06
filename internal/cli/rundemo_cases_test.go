package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func baseEnv(apiURL, lookupURL string) map[string]string {
	return map[string]string{
		"RESPONDER_API_URL":       apiURL,
		"RESPONDER_PORTAL_URL":    "http://portal.example:3002",
		"RESPONDER_TOKEN":         "test-token",
		"RESPONDER_IP_LOOKUP_URL": lookupURL,
		"RESPONDER_LAT":           "29.0",
		"RESPONDER_LNG":           "-95.0",
	}
}

func graphqlVars(t *testing.T, api *apiFake) map[string]any {
	t.Helper()
	var payload struct {
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal([]byte(api.lastBody), &payload); err != nil {
		t.Fatalf("graphql request body is not JSON: %v", err)
	}
	return payload.Variables
}

func TestRunDemoFlagOverridesEnv(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	env := baseEnv(apiURL, lookupURL)
	env["RESPONDER_CALLER_PHONE"] = "+15551234567"
	env["RESPONDER_CALL_TYPE"] = "House Fire - English"
	res := runCommand(t, []string{
		"run-demo",
		"--phone", "(817) 973-1331",
		"--call-type", "Axon Arena",
	}, env)
	requireSuccess(t, res)

	vars := graphqlVars(t, api)
	if vars["phoneNumber"] != "+18179731331" {
		t.Fatalf("expected flag phone to win, got %v", vars["phoneNumber"])
	}
	if vars["chatroomName"] != "Axon Arena" {
		t.Fatalf("expected flag call type to win, got %v", vars["chatroomName"])
	}
	if vars["callerAudioUrl"] != "https://static.cdn.prepared911.dev/audio-demos/axon-arena-caller.raw" {
		t.Fatalf("expected axon arena audio URL, got %v", vars["callerAudioUrl"])
	}
}

func TestRunDemoEnvOverridesDefault(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	env := baseEnv(apiURL, lookupURL)
	env["RESPONDER_CALLER_PHONE"] = "+15551234567"
	env["RESPONDER_CALL_TYPE"] = "House Fire - English"
	res := runCommand(t, []string{"run-demo"}, env)
	requireSuccess(t, res)

	vars := graphqlVars(t, api)
	if vars["phoneNumber"] != "+15551234567" {
		t.Fatalf("expected env phone to win, got %v", vars["phoneNumber"])
	}
	if vars["chatroomName"] != "House Fire - English" {
		t.Fatalf("expected env call type to win, got %v", vars["chatroomName"])
	}
}

func TestRunDemoCountryFlagOverridesEnv(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	env := baseEnv(apiURL, lookupURL)
	env["RESPONDER_COUNTRY"] = "GB"
	res := runCommand(t, []string{
		"run-demo",
		"--country", "US",
		"--phone", "8179731331",
	}, env)
	requireSuccess(t, res)

	if got := graphqlVars(t, api)["phoneNumber"]; got != "+18179731331" {
		t.Fatalf("expected flag country to win, got %v", got)
	}
}

func TestRunDemoFlagsOverrideBaseURLEnvironment(t *testing.T) {
	envAPI := newAPIFake(t)
	envAPIURL := envAPI.start(t)
	flagAPI := newAPIFake(t)
	flagAPIURL := flagAPI.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	env := baseEnv(envAPIURL, lookupURL)
	env["RESPONDER_PORTAL_URL"] = "http://env.portal:3002"
	res := runCommand(t, []string{
		"run-demo",
		"--api-url", flagAPIURL,
		"--portal-url", "http://flag.portal:3002",
	}, env)
	requireSuccess(t, res)

	if got := flagAPI.calls(); got != 1 {
		t.Fatalf("expected the flag API URL to receive one GraphQL call, got %d", got)
	}
	if got := envAPI.calls(); got != 0 {
		t.Fatalf("expected the environment API URL to receive no GraphQL calls, got %d", got)
	}
	if !strings.Contains(res.stdout, "http://flag.portal:3002/chatroom/chatroom-123") {
		t.Fatalf("expected the flag portal URL in stdout, got:\n%s", res.stdout)
	}
}

func TestRunDemoPhoneNormalization(t *testing.T) {
	cases := []struct{ in, want string }{
		{"(817) 973-1331", "+18179731331"},
		{"8179731331", "+18179731331"},
		{"817-973-1331", "+18179731331"},
		{"+1 (817) 973-1331", "+18179731331"},
		{"18179731331", "+18179731331"},
		{"+44 20 7946 0958", "+442079460958"},
	}
	for _, tc := range cases {
		api := newAPIFake(t)
		apiURL := api.start(t)
		lookupURL := startLocationFake(t, 29.0, -95.0)
		env := baseEnv(apiURL, lookupURL)
		res := runCommand(t, []string{"run-demo", "--phone", tc.in}, env)
		requireSuccess(t, res)
		if vars := graphqlVars(t, api); vars["phoneNumber"] != tc.want {
			t.Fatalf("phone %q: expected %s, got %v", tc.in, tc.want, vars["phoneNumber"])
		}
	}
}

func TestRunDemoUnknownCallType(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	res := runCommand(t, []string{"run-demo", "--call-type", "Nope"}, baseEnv(apiURL, lookupURL))
	requireFailure(t, res)
	if !strings.Contains(res.stderr, `unknown call type "Nope"`) {
		t.Fatalf("expected unknown call type error, got:\n%s", res.stderr)
	}
	if !strings.Contains(res.stderr, "Shooting Incident - English") {
		t.Fatalf("expected valid call types listed, got:\n%s", res.stderr)
	}
	if got := api.calls(); got != 0 {
		t.Fatalf("expected no GraphQL call, got %d", got)
	}
}

func TestRunDemoTokenFromEnvLocal(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)
	checkout := checkoutWithToken(t, `GRAPHIQL_DEV_TOKEN="dev-token-abc"`)

	env := map[string]string{
		"RESPONDER_API_URL":       apiURL,
		"RESPONDER_PORTAL_URL":    "http://portal.example:3002",
		"RESPONDER_IP_LOOKUP_URL": lookupURL,
		"RESPONDER_LAT":           "29.0",
		"RESPONDER_LNG":           "-95.0",
		"PREPARED911_DIR":         checkout,
	}
	res := runCommand(t, []string{"run-demo"}, env)
	requireSuccess(t, res)
	if api.lastAuth != "Bearer dev-token-abc" {
		t.Fatalf("expected dev token auth, got %q", api.lastAuth)
	}
}

func TestRunDemoExplicitTokenBeatsEnvLocal(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)
	checkout := checkoutWithToken(t, `GRAPHIQL_DEV_TOKEN="dev-token-abc"`)

	env := baseEnv(apiURL, lookupURL)
	env["PREPARED911_DIR"] = checkout
	res := runCommand(t, []string{"run-demo"}, env)
	requireSuccess(t, res)
	if api.lastAuth != "Bearer test-token" {
		t.Fatalf("expected explicit token to win, got %q", api.lastAuth)
	}
}

func TestRunDemoNoTokenMessage(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)
	checkout := checkoutWithToken(t, "")

	env := map[string]string{
		"RESPONDER_API_URL":       apiURL,
		"RESPONDER_PORTAL_URL":    "http://portal.example:3002",
		"RESPONDER_IP_LOOKUP_URL": lookupURL,
		"RESPONDER_LAT":           "29.0",
		"RESPONDER_LNG":           "-95.0",
		"PREPARED911_DIR":         checkout,
		"RESPONDER_USER_EMAIL":    "dev@example.com",
	}
	res := runCommand(t, []string{"run-demo"}, env)
	requireFailure(t, res)
	for _, want := range []string{"no auth token", "graphiql-token", "dev@example.com"} {
		if !strings.Contains(strings.ToLower(res.stderr), strings.ToLower(want)) {
			t.Fatalf("expected stderr to mention %q, got:\n%s", want, res.stderr)
		}
	}
	if got := api.calls(); got != 0 {
		t.Fatalf("expected no GraphQL call without a token, got %d", got)
	}
}

func TestRunDemoExplicitLocationSkipsLookup(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)

	env := baseEnv(apiURL, "http://127.0.0.1:1/unreachable")
	res := runCommand(t, []string{"run-demo", "--lat", "40.1", "--lng", "-74.2"}, env)
	requireSuccess(t, res)
	vars := graphqlVars(t, api)
	if vars["latitude"] != 40.1 || vars["longitude"] != -74.2 {
		t.Fatalf("expected explicit coordinates, got %v / %v", vars["latitude"], vars["longitude"])
	}
}

func TestRunDemoEnvLocation(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	env := baseEnv(apiURL, lookupURL)
	env["RESPONDER_LAT"] = "40.7"
	env["RESPONDER_LNG"] = "-74.0"
	res := runCommand(t, []string{"run-demo"}, env)
	requireSuccess(t, res)
	vars := graphqlVars(t, api)
	if vars["latitude"] != 40.7 || vars["longitude"] != -74.0 {
		t.Fatalf("expected env coordinates, got %v / %v", vars["latitude"], vars["longitude"])
	}
}

func TestRunDemoPartialLocationIsAnError(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	res := runCommand(t, []string{"run-demo", "--lat", "40.1"}, baseEnv(apiURL, lookupURL))
	requireFailure(t, res)
	if !strings.Contains(res.stderr, "--lat and --lng") {
		t.Fatalf("expected both-flags error, got:\n%s", res.stderr)
	}
}

func TestRunDemoLookupFailure(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(broken.Close)

	env := baseEnv(apiURL, broken.URL)
	delete(env, "RESPONDER_LAT")
	delete(env, "RESPONDER_LNG")
	res := runCommand(t, []string{"run-demo"}, env)
	requireFailure(t, res)
	if !strings.Contains(res.stderr, "--lat") || !strings.Contains(res.stderr, "--lng") {
		t.Fatalf("expected location error to mention --lat/--lng, got:\n%s", res.stderr)
	}
	if got := api.calls(); got != 0 {
		t.Fatalf("expected no GraphQL call after lookup failure, got %d", got)
	}
}

func TestRunDemoUnhealthyAPI(t *testing.T) {
	api := newAPIFake(t)
	api.healthy = false
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	res := runCommand(t, []string{"run-demo"}, baseEnv(apiURL, lookupURL))
	requireFailure(t, res)
	if !strings.Contains(res.stderr, "responder start-app") {
		t.Fatalf("expected start-app hint, got:\n%s", res.stderr)
	}
	if got := api.calls(); got != 0 {
		t.Fatalf("expected no GraphQL call against an unhealthy API, got %d", got)
	}
}

func TestRunDemoGraphQLError(t *testing.T) {
	api := newAPIFake(t)
	api.graphqlHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"user's dispatch center lacks the audio demo setting"}]}`))
	}
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	res := runCommand(t, []string{"run-demo"}, baseEnv(apiURL, lookupURL))
	requireFailure(t, res)
	if !strings.Contains(res.stderr, "lacks the audio demo setting") {
		t.Fatalf("expected GraphQL error surfaced, got:\n%s", res.stderr)
	}
}

func TestRunDemoMissingIncidentID(t *testing.T) {
	api := newAPIFake(t)
	api.graphqlHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"startAudioDemo":{"chatroom":null}}}`))
	}
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	res := runCommand(t, []string{"run-demo"}, baseEnv(apiURL, lookupURL))
	requireFailure(t, res)
	if !strings.Contains(res.stderr, "incident ID") {
		t.Fatalf("expected missing-ID error, got:\n%s", res.stderr)
	}
}

func TestRunDemoCreateTimeout(t *testing.T) {
	api := newAPIFake(t)
	api.graphqlHandler = func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"startAudioDemo":{"chatroom":{"id":"chatroom-123"}}}}`))
	}
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	env := baseEnv(apiURL, lookupURL)
	env["RESPONDER_CREATE_TIMEOUT"] = "100ms"
	res := runCommand(t, []string{"run-demo"}, env)
	requireFailure(t, res)
	if !strings.Contains(strings.ToLower(res.stderr), "timed out") {
		t.Fatalf("expected timeout message, got:\n%s", res.stderr)
	}
}

func TestRunDemoSendsCallTypeAudioURLs(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.0, -95.0)

	env := baseEnv(apiURL, lookupURL)
	res := runCommand(t, []string{"run-demo", "--call-type", "Shooting Incident - Vietnamese"}, env)
	requireSuccess(t, res)
	vars := graphqlVars(t, api)
	if vars["languageCode"] != "vi" {
		t.Fatalf("expected vi language, got %v", vars["languageCode"])
	}
	if vars["callerLanguageCode"] != "vi" || vars["dispatcherLanguageCode"] != "en-US" {
		t.Fatalf("expected per-side languages, got %v", vars)
	}
	if !strings.Contains(vars["callerAudioUrl"].(string), "vietnamese-shooting-caller") {
		t.Fatalf("expected vietnamese caller audio, got %v", vars["callerAudioUrl"])
	}
}

func TestRunDemoSendsEverySupportedCallType(t *testing.T) {
	cases := []struct {
		name       string
		callerURL  string
		dispatcher string
		language   string
	}{
		{
			name:       "Shooting Incident - English",
			callerURL:  "https://static.cdn.prepared911.dev/audio-demos/shooting-incident-english-caller.raw",
			dispatcher: "https://static.cdn.prepared911.dev/audio-demos/shooting-incident-english-dispatcher.raw",
			language:   "en-US",
		},
		{
			name:       "Medical Emergency - School",
			callerURL:  "https://static.cdn.prepared911.dev/audio-demos/shooting-incident-english-caller.raw",
			dispatcher: "https://static.cdn.prepared911.dev/audio-demos/shooting-incident-english-dispatcher.raw",
			language:   "en-US",
		},
		{
			name:       "House Fire - English",
			callerURL:  "https://static.cdn.prepared911.dev/audio-demos/house-fire-english-caller.raw",
			dispatcher: "https://static.cdn.prepared911.dev/audio-demos/house-fire-english-dispatcher.raw",
			language:   "en-US",
		},
		{
			name:       "Home Invasion - English",
			callerURL:  "https://static.cdn.prepared911.dev/audio-demos/home-invasion-english-caller.raw",
			dispatcher: "https://static.cdn.prepared911.dev/audio-demos/home-invasion-english-dispatcher.raw",
			language:   "en-US",
		},
		{
			name:       "Shooting Incident - Vietnamese",
			callerURL:  "https://static.cdn.prepared911.dev/audio-demos/vietnamese-shooting-caller-take1-trimmed.raw",
			dispatcher: "https://static.cdn.prepared911.dev/audio-demos/vietnamese-shooting-dispatcher-take1.raw",
			language:   "vi",
		},
		{
			name:       "Prepared Translator - Spanish",
			callerURL:  "https://static.cdn.prepared911.dev/audio-demos/spanish-translator-caller.raw",
			dispatcher: "https://static.cdn.prepared911.dev/audio-demos/spanish-translator-dispatcher.raw",
			language:   "es",
		},
		{
			name:       "Crime In-Progress - Mayo Blvd",
			callerURL:  "https://static.cdn.prepared911.dev/audio-demos/axon-hq-gsoc-caller.raw",
			dispatcher: "https://static.cdn.prepared911.dev/audio-demos/axon-hq-gsoc-dispatcher.raw",
			language:   "en-US",
		},
		{
			name:       "Axon Arena",
			callerURL:  "https://static.cdn.prepared911.dev/audio-demos/axon-arena-caller.raw",
			dispatcher: "https://static.cdn.prepared911.dev/audio-demos/axon-arena-dispatcher.raw",
			language:   "en-US",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := newAPIFake(t)
			apiURL := api.start(t)
			lookupURL := startLocationFake(t, 29.0, -95.0)

			res := runCommand(t, []string{"run-demo", "--call-type", tc.name}, baseEnv(apiURL, lookupURL))
			requireSuccess(t, res)

			vars := graphqlVars(t, api)
			if vars["chatroomName"] != tc.name {
				t.Fatalf("expected call type name %q, got %v", tc.name, vars["chatroomName"])
			}
			if vars["callerAudioUrl"] != tc.callerURL {
				t.Fatalf("expected caller audio URL %q, got %v", tc.callerURL, vars["callerAudioUrl"])
			}
			if vars["dispatcherAudioUrl"] != tc.dispatcher {
				t.Fatalf("expected dispatcher audio URL %q, got %v", tc.dispatcher, vars["dispatcherAudioUrl"])
			}
			if vars["languageCode"] != tc.language {
				t.Fatalf("expected language code %q, got %v", tc.language, vars["languageCode"])
			}
		})
	}
}

func TestHelpDescribesFlagsAndEnv(t *testing.T) {
	res := runCommand(t, []string{"run-demo", "--help"}, nil)
	requireSuccess(t, res)
	for _, want := range []string{"--phone", "--call-type", "RESPONDER_CALLER_PHONE", "RESPONDER_TOKEN", "RESPONDER_LAT"} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("expected help to mention %q, got:\n%s", want, res.stdout)
		}
	}
	top := runCommand(t, []string{"--help"}, nil)
	requireSuccess(t, top)
	for _, want := range []string{"run-demo", "start-app"} {
		if !strings.Contains(top.stdout, want) {
			t.Fatalf("expected top-level help to mention %q, got:\n%s", want, top.stdout)
		}
	}
}
