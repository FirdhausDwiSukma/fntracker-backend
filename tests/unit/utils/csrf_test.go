// Package utils_test contains unit tests for CSRF token utilities.
// Validates: Requirements 3.1, 3.4
package utils_test

import (
	"encoding/hex"
	"testing"

	"finance-tracker/utils"
)

func TestGenerateCsrfToken_Length64(t *testing.T) {
	token, err := utils.GenerateCsrfToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("expected length 64, got %d: %q", len(token), token)
	}
}

func TestGenerateCsrfToken_IsValidHex(t *testing.T) {
	token, _ := utils.GenerateCsrfToken()
	decoded, err := hex.DecodeString(token)
	if err != nil {
		t.Fatalf("token is not valid hex: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32 decoded bytes, got %d", len(decoded))
	}
}

func TestGenerateCsrfToken_Uniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := range 100 {
		token, err := utils.GenerateCsrfToken()
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if _, dup := seen[token]; dup {
			t.Fatalf("duplicate CSRF token generated at iteration %d: %q", i, token)
		}
		seen[token] = struct{}{}
	}
}

func TestGenerateCsrfToken_NoError(t *testing.T) {
	for range 10 {
		_, err := utils.GenerateCsrfToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}
