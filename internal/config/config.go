package config

import "os"

const (
	DefaultAddr     = "localhost"
	DefaultPort     = "8080"
	DefaultDBString = "postgres://postgres:postgres@localhost:5432/org-db?sslmode=disable"
)

type Config struct {
	Addr     string
	Port     string
	DBString string
}

func NewConfig() *Config {
	return &Config{
		Addr:     getEnv("addr", DefaultAddr),
		Port:     getEnv("port", DefaultPort),
		DBString: getEnv("db_string", DefaultDBString),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
