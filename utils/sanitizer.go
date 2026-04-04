package utils

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// SanitizeString strips all HTML/XSS using bluemonday StrictPolicy and trims whitespace.
func SanitizeString(input string) string {
	p := bluemonday.StrictPolicy()
	return strings.TrimSpace(p.Sanitize(input))
}
