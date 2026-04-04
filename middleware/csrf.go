package middleware

import (
	"net/http"

	"finance-tracker/utils"

	"github.com/gin-gonic/gin"
)

// CSRFMiddleware validates the X-CSRF-Token header against the csrf_token cookie.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		headerToken := c.GetHeader("X-CSRF-Token")
		cookieToken, err := c.Cookie("csrf_token")

		if err != nil || headerToken == "" || cookieToken == "" || headerToken != cookieToken {
			utils.ErrorResponse(c, http.StatusForbidden, utils.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
