// Package utils_test contains unit tests for JWT utilities.
// Validates: Requirements 2.2, 2.6, 2.7
package utils_test

import (
	"testing"
	"time"

	"finance-tracker/utils"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-at-least-32-chars-long"

func TestGenerateToken_ReturnsValidJWT(t *testing.T) {
	token, err := utils.GenerateToken(42, testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestGenerateToken_ClaimsContainUserID(t *testing.T) {
	var userID uint = 99
	token, _ := utils.GenerateToken(userID, testSecret)

	claims, err := utils.ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("expected userID %d, got %d", userID, claims.UserID)
	}
}

func TestGenerateToken_ExpiresIn24Hours(t *testing.T) {
	token, _ := utils.GenerateToken(1, testSecret)
	claims, _ := utils.ValidateToken(token, testSecret)

	expiry := claims.ExpiresAt.Time
	expected := time.Now().Add(24 * time.Hour)

	diff := expiry.Sub(expected)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Fatalf("expiry not ~24h from now: diff=%v", diff)
	}
}

func TestValidateToken_RejectsWrongSecret(t *testing.T) {
	token, _ := utils.GenerateToken(1, testSecret)
	_, err := utils.ValidateToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestValidateToken_RejectsTamperedToken(t *testing.T) {
	token, _ := utils.GenerateToken(1, testSecret)
	tampered := token + "x"
	_, err := utils.ValidateToken(tampered, testSecret)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestValidateToken_RejectsExpiredToken(t *testing.T) {
	claims := utils.JWTClaims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))

	_, err := utils.ValidateToken(token, testSecret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateToken_RejectsGarbageString(t *testing.T) {
	_, err := utils.ValidateToken("not.a.jwt", testSecret)
	if err == nil {
		t.Fatal("expected error for garbage token, got nil")
	}
}

func TestValidateToken_RejectsEmptyString(t *testing.T) {
	_, err := utils.ValidateToken("", testSecret)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}
