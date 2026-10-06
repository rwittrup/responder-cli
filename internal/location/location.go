// Package location resolves the demo's coordinates: explicit flags
// win, otherwise the machine's approximate location is looked up.
package location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultLookupURL is the free IP geolocation endpoint used to mirror
// the portal's "Use My Location" button. prepared911 has no existing
// IP geolocation service to reuse.
const DefaultLookupURL = "http://ip-api.com/json/"

// Coordinates is a latitude/longitude pair.
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// Resolve returns explicit coordinates when both lat and lng are set,
// otherwise it looks up the machine's approximate location. Lookup
// failures are errors: a wrong location must never be used silently.
func Resolve(ctx context.Context, client *http.Client, lookupURL string, lat, lng *float64) (Coordinates, error) {
	if lat != nil && lng != nil {
		return Coordinates{Latitude: *lat, Longitude: *lng}, nil
	}
	if lat != nil || lng != nil {
		return Coordinates{}, fmt.Errorf("pass both --lat and --lng to pick a location")
	}
	if lookupURL == "" {
		lookupURL = DefaultLookupURL
	}
	coords, err := lookup(ctx, client, lookupURL)
	if err != nil {
		return Coordinates{}, fmt.Errorf("could not determine your location (%v); pass --lat and --lng to pick a location explicitly", err)
	}
	return coords, nil
}

func lookup(ctx context.Context, client *http.Client, url string) (Coordinates, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Coordinates{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Coordinates{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Coordinates{}, fmt.Errorf("location service returned %s", resp.Status)
	}
	var body struct {
		Status    string   `json:"status"`
		Message   string   `json:"message"`
		Lat       *float64 `json:"lat"`
		Lon       *float64 `json:"lon"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
		Lng       *float64 `json:"lng"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Coordinates{}, err
	}
	if body.Status != "" && !strings.EqualFold(body.Status, "success") {
		if body.Message != "" {
			return Coordinates{}, fmt.Errorf("location service failed: %s", body.Message)
		}
		return Coordinates{}, fmt.Errorf("location service failed with status %q", body.Status)
	}
	lat := firstNonNil(body.Lat, body.Latitude)
	lon := firstNonNil(body.Lon, body.Lng, body.Longitude)
	if lat == nil || lon == nil {
		return Coordinates{}, fmt.Errorf("location service returned no coordinates")
	}
	return Coordinates{Latitude: *lat, Longitude: *lon}, nil
}

func firstNonNil(vals ...*float64) *float64 {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}
