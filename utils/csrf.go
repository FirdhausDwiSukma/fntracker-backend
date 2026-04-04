package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateCsrfToken generates a cryptographically random 32-byte token, hex-encoded to 64 chars.
func GenerateCsrfToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
