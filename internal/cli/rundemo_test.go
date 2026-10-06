package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunDemoSuccessWithDefaults(t *testing.T) {
	api := newAPIFake(t)
	apiURL := api.start(t)
	lookupURL := startLocationFake(t, 29.7604, -95.3698)

	res := runCommand(t, []string{"run-demo"}, map[string]string{
		"RESPONDER_API_URL":       apiURL,
		"RESPONDER_PORTAL_URL":    "http://portal.example:3002",
		"RESPONDER_TOKEN":         "test-token",
		"RESPONDER_IP_LOOKUP_URL": lookupURL,
	})
	requireSuccess(t, res)

	if !strings.Contains(res.stdout, "chatroom-123") {
		t.Fatalf("expected incident ID in stdout, got:\n%s", res.stdout)
	}
	if !strings.Contains(res.stdout, "http://portal.example:3002/chatroom/chatroom-123") {
		t.Fatalf("expected portal URL in stdout, got:\n%s", res.stdout)
	}

	if got := api.calls(); got != 1 {
		t.Fatalf("expected 1 GraphQL call, got %d", got)
	}
	if api.lastAuth != "Bearer test-token" {
		t.Fatalf("expected bearer auth header, got %q", api.lastAuth)
	}

	var payload struct {
		Variables map[string]any `json:"variables"`
	}
	if err := json.Unmarshal([]byte(api.lastBody), &payload); err != nil {
		t.Fatalf("graphql request body is not JSON: %v", err)
	}
	vars := payload.Variables
	if vars["chatroomName"] != "Shooting Incident - English" {
		t.Fatalf("expected default call type name, got %v", vars["chatroomName"])
	}
	if vars["phoneNumber"] != "+18179731331" {
		t.Fatalf("expected default E.164 phone, got %v", vars["phoneNumber"])
	}
	if vars["locationMode"] != "custom" {
		t.Fatalf("expected custom location mode, got %v", vars["locationMode"])
	}
	if vars["produceLocation"] != true || vars["produceCadIncident"] != true || vars["forceAbandoned"] != false {
		t.Fatalf("expected location/cad production true and forced-abandoned false, got %v", vars)
	}
	if vars["latitude"] != 29.7604 || vars["longitude"] != -95.3698 {
		t.Fatalf("expected looked-up coordinates, got %v / %v", vars["latitude"], vars["longitude"])
	}
}
