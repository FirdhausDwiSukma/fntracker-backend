package middleware

import (
	"net/http"

	"finance-tracker/utils"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware validates the JWT from the "jwt" cookie and sets userID in context.
func JWTAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("jwt")
		if err != nil || tokenStr == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, utils.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(tokenStr, jwtSecret)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, utils.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Next()
	}
}
