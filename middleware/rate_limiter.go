package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter holds per-IP rate limiters using a sync.Map.
type IPRateLimiter struct {
	limiters sync.Map
	r        rate.Limit
	b        int
}

// NewIPRateLimiter creates a new IPRateLimiter with the given rate and burst.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{r: r, b: b}
}

// GetLimiter returns the rate.Limiter for the given IP, creating one if it doesn't exist.
func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	v, _ := l.limiters.LoadOrStore(ip, rate.NewLimiter(l.r, l.b))
	return v.(*rate.Limiter)
}

// LoginRateLimiter returns a middleware that limits to 5 req/min per IP.
func LoginRateLimiter() gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate.Every(time.Minute/5), 5)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.GetLimiter(ip).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			c.Abort()
			return
		}
		c.Next()
	}
}
