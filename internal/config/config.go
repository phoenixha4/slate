// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration values.
type Config struct {
	AppEnv             string
	LogLevel           slog.Level
	LogFormat          string
	Port               string
	DatabaseURL        string
	CORSAllowedOrigins []string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	ShutdownTimeout    time.Duration
	ReadinessTimeout   time.Duration

	// Telemetry (OpenTelemetry)
	OTLPEndpoint   string // OTEL_EXPORTER_OTLP_ENDPOINT — empty disables OTLP export
	ServiceName    string // OTEL_SERVICE_NAME
	ServiceVersion string // OTEL_SERVICE_VERSION
}

// Load reads configuration from environment variables.
// Returns an error if any required variable is absent.
func Load() (*Config, error) {
	port := getenv("PORT", "8080")
	if err := validatePort(port); err != nil {
		return nil, err
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	logLevel, err := parseLogLevel(getenv("LOG_LEVEL", "info"))
	if err != nil {
		return nil, err
	}
	logFormat, err := parseLogFormat(getenv("LOG_FORMAT", "json"))
	if err != nil {
		return nil, err
	}
	readTimeout, err := parseDuration("READ_TIMEOUT", "15s")
	if err != nil {
		return nil, err
	}
	writeTimeout, err := parseDuration("WRITE_TIMEOUT", "30s")
	if err != nil {
		return nil, err
	}
	idleTimeout, err := parseDuration("IDLE_TIMEOUT", "60s")
	if err != nil {
		return nil, err
	}
	shutdownTimeout, err := parseDuration("SHUTDOWN_TIMEOUT", "10s")
	if err != nil {
		return nil, err
	}
	readinessTimeout, err := parseDuration("READINESS_TIMEOUT", "2s")
	if err != nil {
		return nil, err
	}

	return &Config{
		AppEnv:             getenv("APP_ENV", "development"),
		LogLevel:           logLevel,
		LogFormat:          logFormat,
		Port:               port,
		DatabaseURL:        dbURL,
		CORSAllowedOrigins: parseCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		ReadTimeout:        readTimeout,
		WriteTimeout:       writeTimeout,
		IdleTimeout:        idleTimeout,
		ShutdownTimeout:    shutdownTimeout,
		ReadinessTimeout:   readinessTimeout,
		OTLPEndpoint:       os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		ServiceName:        getenv("OTEL_SERVICE_NAME", "todo"),
		ServiceVersion:     getenv("OTEL_SERVICE_VERSION", "dev"),
	}, nil
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func validatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("PORT must be a number between 1 and 65535")
	}
	return nil
}

func parseLogLevel(raw string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(raw)))); err != nil {
		return slog.LevelInfo, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, or error")
	}
	return level, nil
}

func parseLogFormat(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "json":
		return "json", nil
	case "text":
		return "text", nil
	case "pretty":
		return "pretty", nil
	default:
		return "", fmt.Errorf("LOG_FORMAT must be one of json, text, or pretty")
	}
}

func parseDuration(key, fallback string) (time.Duration, error) {
	raw := getenv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration such as 5s or 1m", key)
	}
	return d, nil
}

func parseCSV(raw string) []string {
	values := strings.Split(raw, ",")
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
