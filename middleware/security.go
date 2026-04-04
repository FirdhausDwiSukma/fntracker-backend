package middleware

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware sets X-Content-Type-Options, X-Frame-Options, X-XSS-Protection.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	}
}
