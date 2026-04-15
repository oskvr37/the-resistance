package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	RedisAddr     string
	RedisUser     string
	RedisPassword string
	RedisDB       int
	JWTSecret     string
	Port          string
	Environment   string
}

func LoadConfig() *Config {
	// Load .env file. We don't panic if it fails, as
	// env vars might be set via Docker or the shell.
	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found; using system environment variables.")
	}

	return &Config{
		RedisAddr:     getEnv("REDIS_ADDRESS", "localhost:6379"),
		RedisUser:     getEnv("REDIS_USERNAME", ""),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		JWTSecret:     getEnv("JWT_SECRET", "super-secret-key"),
		Port:          getEnv("PORT", "8000"),
		Environment:   getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}
