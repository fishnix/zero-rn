package config

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	PollInterval time.Duration
	ScanTimeout  time.Duration
	ReadTimeout  time.Duration
	HTTPAddr     string
	NamePrefix   string
	LogLevel     string
	DeviceMAC    string
	HTTPToken    string
}

func Load() Config {
	return Config{
		PollInterval: mustDuration("RADON_POLL_INTERVAL", 30*time.Minute),
		ScanTimeout:  mustDuration("RADON_SCAN_TIMEOUT", 30*time.Second),
		ReadTimeout:  mustDuration("RADON_READ_TIMEOUT", 10*time.Second),
		HTTPAddr:     mustString("RADON_HTTP_ADDR", ":8080"),
		NamePrefix:   mustString("RADON_NAME_PREFIX", "FR:R20:"),
		LogLevel:     mustString("RADON_LOG_LEVEL", "info"),
		DeviceMAC:    strings.TrimSpace(os.Getenv("RADON_DEVICE_MAC")),
		HTTPToken:    strings.TrimSpace(os.Getenv("RADON_HTTP_TOKEN")),
	}
}

func NewLogger(level string) *slog.Logger {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
}

func mustDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func mustString(key, fallback string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	return raw
}
