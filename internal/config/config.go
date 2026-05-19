package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	AppPort     string
	AppSecret   string
	DatabaseURL string
	RedisURL    string
	NatsURL     string

	// CORS
	CORSAllowedOrigins []string

	// Auth
	RefreshTokenSecret       string
	AccessTokenExpiryMinutes int
	RefreshTokenExpiryDays   int

	// Cloudflare R2
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2Endpoint        string

	// AI
	OpenAIApiKey string
	OpenAIModel  string

	// Zenziva WhatsApp OTP
	ZenzivaURL     string
	ZenzivaUserKey string
	ZenzivaPassKey string
	ZenzivaBrand   string

	// Xendit Payment
	XenditAPIKey        string
	XenditWebhookSecret string
	XenditCallbackURL   string
	XenditSuccessURL    string
	XenditFailureURL    string

	// SMTP
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// Worker
	WorkerConcurrency int
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load .env: %w", err)
	}

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		AppPort:     getEnv("APP_PORT", "8080"),
		AppSecret:   getEnv("APP_SECRET", ""),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		NatsURL:     getEnv("NATS_URL", "nats://localhost:4222"),

		CORSAllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),

		RefreshTokenSecret:       getEnv("REFRESH_TOKEN_SECRET", ""),
		AccessTokenExpiryMinutes: getEnvInt("ACCESS_TOKEN_EXPIRY_MINUTES", 15),
		RefreshTokenExpiryDays:   getEnvInt("REFRESH_TOKEN_EXPIRY_DAYS", 7),

		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", "azzet-documents"),
		R2Endpoint:        getEnv("R2_ENDPOINT", ""),

		OpenAIApiKey: getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:  getEnv("OPENAI_MODEL", "gpt-4-turbo"),

		ZenzivaURL:     getEnv("ZENZIVA_URL", "https://console.zenziva.net/waofficial/api/sendWAOfficial/"),
		ZenzivaUserKey: getEnv("ZENZIVA_USERKEY", ""),
		ZenzivaPassKey: getEnv("ZENZIVA_PASSKEY", ""),
		ZenzivaBrand:   getEnv("ZENZIVA_BRAND", "Azzet"),

		XenditAPIKey:        getEnv("XENDIT_API_KEY", ""),
		XenditWebhookSecret: getEnv("XENDIT_WEBHOOK_SECRET", ""),
		XenditCallbackURL:   getEnv("XENDIT_CALLBACK_URL", ""),
		XenditSuccessURL:    getEnv("XENDIT_SUCCESS_URL", ""),
		XenditFailureURL:    getEnv("XENDIT_FAILURE_URL", ""),

		SMTPHost: getEnv("SMTP_HOST", ""),
		SMTPPort: getEnv("SMTP_PORT", "587"),
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPass: getEnv("SMTP_PASS", ""),
		SMTPFrom: getEnv("SMTP_FROM", "noreply@azzet.com"),

		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 50),
	}

	if cfg.AppSecret == "" {
		return nil, errors.New("APP_SECRET is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.RefreshTokenSecret == "" {
		return nil, errors.New("REFRESH_TOKEN_SECRET is required")
	}

	return cfg, nil
}

// RedisAddr returns the host:port for Redis (used by asynq)
func (c *Config) RedisAddr() string {
	url := c.RedisURL
	url = strings.TrimPrefix(url, "redis://")
	url = strings.TrimPrefix(url, "rediss://")
	// Remove auth if present (user:pass@host:port)
	if idx := strings.LastIndex(url, "@"); idx != -1 {
		url = url[idx+1:]
	}
	// Remove path/db if present
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}
	return url
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		var result []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}
