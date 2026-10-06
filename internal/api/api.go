// Package api talks to the prepared911 API: health checks and the
// startAudioDemo GraphQL mutation the portal uses.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rwittrup/responder-cli/internal/calltypes"
	"github.com/rwittrup/responder-cli/internal/location"
)

// HealthPath is the API's health endpoint (OkComputer mount).
const HealthPath = "/healthz"

// GraphQLPath is the API's GraphQL endpoint.
const GraphQLPath = "/graphql"

// Client talks to one API base URL.
type Client struct {
	BaseURL    string
	Token      string
	HTTP       *http.Client
	APIBaseURL string
}

// NewClient builds a client with the given per-request timeout.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTP:    &http.Client{Timeout: timeout},
	}
}

// CheckHealth GETs the health endpoint. A healthy stack returns 2xx.
func (c *Client) CheckHealth(ctx context.Context) error {
	url := c.BaseURL + HealthPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("API at %s is not healthy (%s); run `responder start-app` to bring up the local stack", c.BaseURL, resp.Status)
	}
	return nil
}

// startAudioDemo mutation mirroring the portal's StartAudioDemo.graphql.
const startAudioDemoMutation = `mutation StartAudioDemo(
  $chatroomName: String!
  $phoneNumber: String!
  $languageCode: String!
  $callerLanguageCode: String
  $callerAudioUrl: String
  $dispatcherAudioUrl: String
  $dispatcherLanguageCode: String
  $forceAbandoned: Boolean
  $produceLocation: Boolean
  $latitude: Float
  $longitude: Float
  $locationMode: DemoLocationMode
  $produceCadIncident: Boolean
) {
  startAudioDemo(
    input: {
      chatroomName: $chatroomName
      phoneNumber: $phoneNumber
      languageCode: $languageCode
      callerLanguageCode: $callerLanguageCode
      callerAudioUrl: $callerAudioUrl
      dispatcherAudioUrl: $dispatcherAudioUrl
      dispatcherLanguageCode: $dispatcherLanguageCode
      forceAbandoned: $forceAbandoned
      produceLocation: $produceLocation
      latitude: $latitude
      longitude: $longitude
      locationMode: $locationMode
      produceCadIncident: $produceCadIncident
    }
  ) {
    chatroom { id }
    audioServerResponseCode
  }
}`

// StartAudioDemo sends one startAudioDemo mutation and returns the new
// incident (chatroom) ID. It requires an ID in the response: a missing
// ID, a GraphQL error, or an HTTP error is a failure.
func (c *Client) StartAudioDemo(ctx context.Context, ct calltypes.CallType, phoneNumber string, coords location.Coordinates) (string, error) {
	variables := map[string]any{
		"chatroomName":       ct.Name,
		"phoneNumber":        phoneNumber,
		"languageCode":       ct.LanguageCode,
		"callerAudioUrl":     ct.CallerAudioURL,
		"dispatcherAudioUrl": ct.DispatcherAudioURL,
		"produceLocation":    true,
		"produceCadIncident": true,
		"forceAbandoned":     false,
		"locationMode":       "custom",
		"latitude":           coords.Latitude,
		"longitude":          coords.Longitude,
	}
	if ct.CallerLanguageCode != "" {
		variables["callerLanguageCode"] = ct.CallerLanguageCode
	} else {
		variables["callerLanguageCode"] = nil
	}
	if ct.DispatcherLanguageCode != "" {
		variables["dispatcherLanguageCode"] = ct.DispatcherLanguageCode
	} else {
		variables["dispatcherLanguageCode"] = ct.LanguageCode
	}

	body, err := json.Marshal(map[string]any{"query": startAudioDemoMutation, "variables": variables})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+GraphQLPath, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("API request failed (%s): %s", resp.Status, truncate(string(raw), 300))
	}
	var payload struct {
		Data *struct {
			StartAudioDemo *struct {
				Chatroom *struct {
					ID string `json:"id"`
				} `json:"chatroom"`
			} `json:"startAudioDemo"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("could not parse API response: %v", err)
	}
	if len(payload.Errors) > 0 {
		msgs := make([]string, 0, len(payload.Errors))
		for _, e := range payload.Errors {
			msgs = append(msgs, e.Message)
		}
		return "", fmt.Errorf("API error: %s", strings.Join(msgs, "; "))
	}
	if payload.Data == nil || payload.Data.StartAudioDemo == nil ||
		payload.Data.StartAudioDemo.Chatroom == nil ||
		strings.TrimSpace(payload.Data.StartAudioDemo.Chatroom.ID) == "" {
		return "", fmt.Errorf("API did not return an incident ID; the demo was not created")
	}
	return payload.Data.StartAudioDemo.Chatroom.ID, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
