package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DBUrl          string
	JWTSecret      string
	AllowedOrigins []string
	Port           string
	Env            string
}

// Load reads environment variables (from .env file if present) and returns a Config.
func Load() *Config {
	// Load .env file if it exists; ignore error in production where env vars are set directly.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	originsRaw := os.Getenv("ALLOWED_ORIGINS")
	var origins []string
	if originsRaw != "" {
		for _, o := range strings.Split(originsRaw, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}

	return &Config{
		DBUrl:          os.Getenv("DB_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		AllowedOrigins: origins,
		Port:           port,
		Env:            env,
	}
}
