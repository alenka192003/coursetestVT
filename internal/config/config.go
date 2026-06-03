package config

import "os"

type Config struct {
	Port    string
	Version string
}

func Load() Config {
	return Config{
		Port:    getenv("PORT", "8080"),
		Version: getenv("APP_VERSION", "dev"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
