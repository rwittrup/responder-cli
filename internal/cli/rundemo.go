package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rwittrup/responder-cli/internal/api"
	"github.com/rwittrup/responder-cli/internal/calltypes"
	"github.com/rwittrup/responder-cli/internal/location"
	"github.com/rwittrup/responder-cli/internal/phone"
	"github.com/rwittrup/responder-cli/internal/token"
)

const (
	defaultPhone         = "+1 (817) 973-1331"
	defaultCountry       = "US"
	defaultAPIURL        = "http://127.0.0.1:3000"
	defaultPortalURL     = "http://localhost:3002"
	defaultCheckout      = "~/workspace/prepared911"
	defaultCreateTimeout = 30 * time.Second
)

type runDemoOptions struct {
	phone       string
	country     string
	callType    string
	lat         float64
	lng         float64
	apiURL      string
	portalURL   string
	authToken   string
	checkoutDir string
	userEmail   string
}

func newRunDemoCmd() *cobra.Command {
	var opts runDemoOptions
	cmd := &cobra.Command{
		Use:   "run-demo",
		Short: "Create an incident from an audio demo",
		Long: `Create an Incident from an Audio Demo with sensible defaults,
using the same API the portal uses, then print the incident ID and
a clickable portal URL.

Configuration precedence is flag, then environment variable,
then default.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemo(cmd, opts)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.phone, "phone", defaultPhone, "caller phone number (env RESPONDER_CALLER_PHONE)")
	f.StringVar(&opts.country, "country", defaultCountry, "country for numbers entered without a + (env RESPONDER_COUNTRY)")
	f.StringVar(&opts.callType, "call-type", calltypes.DefaultName, "call type to run (env RESPONDER_CALL_TYPE)")
	f.Float64Var(&opts.lat, "lat", 0, "demo latitude; requires --lng (env RESPONDER_LAT)")
	f.Float64Var(&opts.lng, "lng", 0, "demo longitude; requires --lat (env RESPONDER_LNG)")
	f.StringVar(&opts.apiURL, "api-url", defaultAPIURL, "prepared911 API base URL (env RESPONDER_API_URL)")
	f.StringVar(&opts.portalURL, "portal-url", defaultPortalURL, "portal base URL for the incident link (env RESPONDER_PORTAL_URL)")
	f.StringVar(&opts.authToken, "token", "", "API auth token (env RESPONDER_TOKEN)")
	f.StringVar(&opts.checkoutDir, "prepared911-dir", defaultCheckout, "prepared911 checkout for the dev-token fallback (env PREPARED911_DIR)")
	f.StringVar(&opts.userEmail, "user-email", "", "user whose token is wanted; defaults to git email (env RESPONDER_USER_EMAIL)")
	return cmd
}

func runDemo(cmd *cobra.Command, opts runDemoOptions) error {
	changed := func(name string) bool { return cmd.Flags().Changed(name) }
	if v, ok := envOrUnset("RESPONDER_CALLER_PHONE", changed("phone")); ok {
		opts.phone = v
	}
	if v, ok := envOrUnset("RESPONDER_COUNTRY", changed("country")); ok {
		opts.country = v
	}
	if v, ok := envOrUnset("RESPONDER_CALL_TYPE", changed("call-type")); ok {
		opts.callType = v
	}
	if v, ok := envOrUnset("RESPONDER_API_URL", changed("api-url")); ok {
		opts.apiURL = v
	}
	if v, ok := envOrUnset("RESPONDER_PORTAL_URL", changed("portal-url")); ok {
		opts.portalURL = v
	}
	if v, ok := envOrUnset("RESPONDER_TOKEN", changed("token")); ok {
		opts.authToken = v
	}
	if v, ok := envOrUnset("PREPARED911_DIR", changed("prepared911-dir")); ok {
		opts.checkoutDir = v
	}
	if v, ok := envOrUnset("RESPONDER_USER_EMAIL", changed("user-email")); ok {
		opts.userEmail = v
	}
	if strings.TrimSpace(opts.userEmail) == "" {
		opts.userEmail = token.DefaultUserEmail()
	}

	ct, err := calltypes.Lookup(strings.TrimSpace(opts.callType))
	if err != nil {
		return err
	}
	e164, err := phone.Normalize(opts.phone, opts.country)
	if err != nil {
		return err
	}

	createTimeout := parseDurationEnv("RESPONDER_CREATE_TIMEOUT", defaultCreateTimeout)
	client := api.NewClient(opts.apiURL, "", createTimeout)

	healthCtx, cancelHealth := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelHealth()
	if err := client.CheckHealth(healthCtx); err != nil {
		return err
	}

	checkoutDir := expandHome(opts.checkoutDir)
	authToken, err := token.Resolve(opts.authToken, checkoutDir, opts.userEmail)
	if err != nil {
		return err
	}
	client.Token = authToken

	lat, lng, err := resolveLatLng(cmd, changed)
	if err != nil {
		return err
	}
	lookupURL := strings.TrimSpace(os.Getenv("RESPONDER_IP_LOOKUP_URL"))
	locCtx, cancelLoc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelLoc()
	coords, err := location.Resolve(locCtx, &http.Client{Timeout: 10 * time.Second}, lookupURL, lat, lng)
	if err != nil {
		return err
	}

	createCtx, cancelCreate := context.WithTimeout(context.Background(), createTimeout)
	defer cancelCreate()
	incidentID, err := client.StartAudioDemo(createCtx, ct, e164, coords)
	if err != nil {
		if createCtx.Err() == context.DeadlineExceeded || isTimeoutError(err) {
			return fmt.Errorf("creating the demo timed out after %s; the audio pipeline may be stalled", createTimeout)
		}
		return err
	}

	portal := strings.TrimRight(strings.TrimSpace(opts.portalURL), "/")
	cmd.Printf("Incident created: %s\n", incidentID)
	cmd.Printf("Portal URL: %s/chatroom/%s\n", portal, incidentID)
	return nil
}

// resolveLatLng returns explicit coordinates when both --lat and --lng
// (or both RESPONDER_LAT and RESPONDER_LNG) are set, nils for lookup
// otherwise, and an error when only one side is set.
func resolveLatLng(cmd *cobra.Command, changed func(string) bool) (*float64, *float64, error) {
	latChanged, lngChanged := changed("lat"), changed("lng")
	if latChanged || lngChanged {
		if !latChanged || !lngChanged {
			return nil, nil, fmt.Errorf("pass both --lat and --lng to pick a location")
		}
		lat, _ := cmd.Flags().GetFloat64("lat")
		lng, _ := cmd.Flags().GetFloat64("lng")
		return &lat, &lng, nil
	}
	latStr, latOK := os.LookupEnv("RESPONDER_LAT")
	lngStr, lngOK := os.LookupEnv("RESPONDER_LNG")
	if latOK || lngOK {
		if !latOK || strings.TrimSpace(latStr) == "" || !lngOK || strings.TrimSpace(lngStr) == "" {
			return nil, nil, fmt.Errorf("pass both RESPONDER_LAT and RESPONDER_LNG to pick a location")
		}
		lat, err := strconv.ParseFloat(strings.TrimSpace(latStr), 64)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid RESPONDER_LAT %q", latStr)
		}
		lng, err := strconv.ParseFloat(strings.TrimSpace(lngStr), 64)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid RESPONDER_LNG %q", lngStr)
		}
		return &lat, &lng, nil
	}
	return nil, nil, nil
}

func envOrUnset(key string, flagChanged bool) (string, bool) {
	if flagChanged {
		return "", false
	}
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

func parseDurationEnv(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil || d <= 0 {
		return def
	}
	return d
}

// isTimeoutError reports whether err is a network timeout.
func isTimeoutError(err error) bool {
	var te interface{ Timeout() bool }
	return errors.As(err, &te) && te.Timeout()
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
