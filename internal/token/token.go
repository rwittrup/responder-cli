// Package token resolves the API auth token: explicit setting first,
// then the dev token in the prepared911 checkout's .env.local.
package token

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DevTokenKey is the variable holding the dev token in .env.local,
// written by prepared911's dev-token recipe.
const DevTokenKey = "GRAPHIQL_DEV_TOKEN"

// Resolve returns the auth token. Order: explicit setting, then the
// dev token in the checkout's .env.local, then an error telling the
// user how to mint one.
func Resolve(explicit, checkoutDir, userEmail string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if tok := readDevToken(checkoutDir); tok != "" {
		return tok, nil
	}
	email := strings.TrimSpace(userEmail)
	if email == "" {
		email = gitEmail()
	}
	var hint strings.Builder
	fmt.Fprintf(&hint, "no auth token found (checked --token/RESPONDER_TOKEN and %s in %s/.env.local)", DevTokenKey, checkoutDir)
	if email != "" {
		fmt.Fprintf(&hint, " for user %s", email)
	}
	fmt.Fprintf(&hint, "\nTo mint a dev token, run:\n\n    just api graphiql-token\n\nin your prepared911 checkout (%s)", checkoutDir)
	if email != "" {
		fmt.Fprintf(&hint, " as %s", email)
	}
	fmt.Fprintf(&hint, "\n\nThe user must belong to a dispatch center with the audio demo setting enabled (for example Engineering PD).")
	return "", fmt.Errorf("%s", hint.String())
}

func readDevToken(checkoutDir string) string {
	data, err := os.ReadFile(filepath.Join(checkoutDir, ".env.local"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, DevTokenKey+"=") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(line, DevTokenKey+"="))
		val = strings.TrimPrefix(val, "export ")
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if val != "" {
			return val
		}
	}
	return ""
}

// DefaultUserEmail returns the git user.email, or "" when unavailable.
func DefaultUserEmail() string {
	return gitEmail()
}

func gitEmail() string {
	out, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
