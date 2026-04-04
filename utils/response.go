package utils

import "github.com/gin-gonic/gin"

// Standard error message constants (safe for client exposure)
const (
	ErrInvalidInput       = "invalid input"
	ErrUnauthorized       = "unauthorized"
	ErrForbidden          = "forbidden"
	ErrNotFound           = "not found"
	ErrConflict           = "resource already exists"
	ErrInternalServer     = "internal server error"
	ErrInvalidCredentials = "invalid email or password"
)

// SuccessResponse writes a JSON success response: {"data": data, "message": message}
func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, gin.H{
		"data":    data,
		"message": message,
	})
}

// ErrorResponse writes a JSON error response: {"error": message}
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"error": message,
	})
}
