package utils

import (
	"regexp"
	"time"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// IsValidDate checks if a string is a valid ISO 8601 date (YYYY-MM-DD).
func IsValidDate(dateStr string) bool {
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

// IsValidEmail checks if a string is a valid email format.
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// IsPositiveAmount checks if amount is strictly greater than 0.
func IsPositiveAmount(amount float64) bool {
	return amount > 0
}
