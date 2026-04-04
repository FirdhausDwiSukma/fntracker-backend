// Package middleware_test contains unit tests for JWTAuthMiddleware.
// Validates: Requirements 2.6, 2.7
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"finance-tracker/middleware"
	"finance-tracker/utils"

	"github.com/gin-gonic/gin"
)

const jwtSecret = "unit-test-jwt-secret-32-chars-ok"

func init() {
	gin.SetMode(gin.TestMode)
}

func newAuthRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.JWTAuthMiddleware(jwtSecret))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		c.JSON(http.StatusOK, gin.H{"userID": userID})
	})
	return r
}

func TestJWTAuth_NoCooke_Returns401(t *testing.T) {
	r := newAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_InvalidToken_Returns401(t *testing.T) {
	r := newAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: "invalid.jwt.token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_WrongSecret_Returns401(t *testing.T) {
	token, _ := utils.GenerateToken(1, "different-secret-key-32-chars-ok!")
	r := newAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuth_ValidToken_Returns200AndSetsUserID(t *testing.T) {
	var userID uint = 42
	token, err := utils.GenerateToken(userID, jwtSecret)
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	r := newAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJWTAuth_EmptyCookieValue_Returns401(t *testing.T) {
	r := newAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: ""})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
