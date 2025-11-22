package main

import "os"

type Config struct {
	Port         string
	OtelEndpoint string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
}

func LoadConfig() *Config {
	return &Config{
		Port:         getEnv("PORT", "8080"),
		OtelEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector.lgtm.svc.cluster.local:4318"),
		DBHost:       getEnv("DB_HOST", "postgres.default.svc.cluster.local"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "demouser"),
		DBPassword:   getEnv("DB_PASSWORD", "demopass"),
		DBName:       getEnv("DB_NAME", "demoapp"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
