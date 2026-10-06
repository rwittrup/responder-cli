// Package phone normalizes caller phone numbers to E.164.
package phone

import (
	"fmt"
	"strings"
)

// Normalize returns the E.164 form of raw. A leading "+" marks the
// number as already international and is preserved; otherwise the
// country supplies the default region, which in this version means a
// "+1" prefix for US numbers.
func Normalize(raw, country string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("phone number is empty")
	}
	if strings.HasPrefix(s, "+") {
		digits := digitsOnly(s[1:])
		if digits == "" {
			return "", fmt.Errorf("phone number %q has no digits", raw)
		}
		return "+" + digits, nil
	}
	digits := digitsOnly(s)
	if digits == "" {
		return "", fmt.Errorf("phone number %q has no digits", raw)
	}
	if strings.EqualFold(strings.TrimSpace(country), "US") || strings.TrimSpace(country) == "" {
		if strings.HasPrefix(digits, "1") {
			return "+" + digits, nil
		}
		return "+1" + digits, nil
	}
	return "+" + digits, nil
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
