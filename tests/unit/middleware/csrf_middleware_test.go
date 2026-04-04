// Package middleware_test contains unit tests for CSRFMiddleware.
// Validates: Requirements 3.2, 3.3
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"finance-tracker/middleware"

	"github.com/gin-gonic/gin"
)

func newCSRFRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.CSRFMiddleware())
	r.POST("/action", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

const validCSRF = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 64 chars

func TestCSRF_NoHeaderNoCookie_Returns403(t *testing.T) {
	r := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCSRF_HeaderWithoutCookie_Returns403(t *testing.T) {
	r := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.Header.Set("X-CSRF-Token", validCSRF)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCSRF_CookieWithoutHeader_Returns403(t *testing.T) {
	r := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: validCSRF})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCSRF_MismatchedTokens_Returns403(t *testing.T) {
	r := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: validCSRF})
	req.Header.Set("X-CSRF-Token", "different-token-value-that-does-not-match-cookie")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCSRF_MatchingTokens_Returns200(t *testing.T) {
	r := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: validCSRF})
	req.Header.Set("X-CSRF-Token", validCSRF)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCSRF_EmptyHeaderValue_Returns403(t *testing.T) {
	r := newCSRFRouter()
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: validCSRF})
	req.Header.Set("X-CSRF-Token", "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
