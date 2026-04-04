// Package middleware_test contains unit tests for the IP rate limiter.
// Validates: Requirements 8.2
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"finance-tracker/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func newRateLimitRouter() *gin.Engine {
	r := gin.New()
	r.POST("/login", middleware.LoginRateLimiter(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func doLoginRequest(r *gin.Engine, ip string) int {
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRateLimiter_AllowsFirst5Requests(t *testing.T) {
	r := newRateLimitRouter()
	ip := "10.0.0.1"
	for i := range 5 {
		code := doLoginRequest(r, ip)
		if code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, code)
		}
	}
}

func TestRateLimiter_Blocks6thRequest(t *testing.T) {
	r := newRateLimitRouter()
	ip := "10.0.0.2"
	for range 5 {
		doLoginRequest(r, ip)
	}
	code := doLoginRequest(r, ip)
	if code != http.StatusTooManyRequests {
		t.Fatalf("6th request: expected 429, got %d", code)
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	r := newRateLimitRouter()

	// Exhaust IP A.
	for range 5 {
		doLoginRequest(r, "10.1.0.1")
	}
	if code := doLoginRequest(r, "10.1.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("IP A 6th: expected 429, got %d", code)
	}

	// IP B should still be allowed.
	if code := doLoginRequest(r, "10.1.0.2"); code != http.StatusOK {
		t.Fatalf("IP B first request: expected 200, got %d", code)
	}
}

func TestIPRateLimiter_GetLimiter_ReturnsSameInstance(t *testing.T) {
	l := middleware.NewIPRateLimiter(rate.Every(time.Minute), 5)
	l1 := l.GetLimiter("192.168.1.1")
	l2 := l.GetLimiter("192.168.1.1")
	if l1 != l2 {
		t.Error("expected same limiter instance for same IP")
	}
}

func TestIPRateLimiter_GetLimiter_DifferentIPsDifferentInstances(t *testing.T) {
	l := middleware.NewIPRateLimiter(rate.Every(time.Minute), 5)
	l1 := l.GetLimiter("192.168.1.1")
	l2 := l.GetLimiter("192.168.1.2")
	if l1 == l2 {
		t.Error("expected different limiter instances for different IPs")
	}
}
