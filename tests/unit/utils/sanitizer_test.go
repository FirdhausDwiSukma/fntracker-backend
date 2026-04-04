// Package utils_test contains unit tests for the XSS sanitizer.
// Validates: Requirements 8.7
package utils_test

import (
	"testing"

	"finance-tracker/utils"
)

func TestSanitizeString_StripsTags(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"<script>alert('xss')</script>", ""},
		{"<b>bold</b>", "bold"},
		{"<img src=x onerror=alert(1)>", ""},
		{"<a href='javascript:void(0)'>click</a>", "click"},
		{"Hello <em>world</em>!", "Hello world!"},
		{"<div><p>nested</p></div>", "nested"},
	}

	for _, tc := range cases {
		got := utils.SanitizeString(tc.input)
		if got != tc.expected {
			t.Errorf("SanitizeString(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestSanitizeString_TrimsWhitespace(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"\t\nworld\n\t", "world"},
		{"  ", ""},
	}

	for _, tc := range cases {
		got := utils.SanitizeString(tc.input)
		if got != tc.expected {
			t.Errorf("SanitizeString(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestSanitizeString_PreservesPlainText(t *testing.T) {
	// bluemonday HTML-encodes special chars like & → &amp; as part of strict sanitization.
	// We verify that plain alphanumeric text and common punctuation pass through unchanged.
	inputs := []string{
		"Hello, World!",
		"Gaji Bulanan",
		"100.000",
		"user@example.com",
		"Makanan Sehari-hari",
	}
	for _, input := range inputs {
		got := utils.SanitizeString(input)
		if got != input {
			t.Errorf("SanitizeString(%q) = %q, want unchanged", input, got)
		}
	}
}

func TestSanitizeString_EmptyInput(t *testing.T) {
	got := utils.SanitizeString("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestSanitizeString_XSSPayloads(t *testing.T) {
	payloads := []string{
		`<script>document.cookie</script>`,
		`"><script>alert(1)</script>`,
		`<svg onload=alert(1)>`,
		`<iframe src="javascript:alert(1)">`,
		`<body onload=alert(1)>`,
	}
	for _, payload := range payloads {
		got := utils.SanitizeString(payload)
		if got == payload {
			t.Errorf("XSS payload not sanitized: %q", payload)
		}
	}
}
