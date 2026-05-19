package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	AppPort     string
	AppSecret   string
	DatabaseURL string
	RedisURL    string
	NatsURL     string

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2Endpoint        string

	OpenAIApiKey string
	OpenAIModel  string

	ZenzivaURL     string
	ZenzivaUserKey string
	ZenzivaPassKey string
	ZenzivaBrand   string

	XenditAPIKey        string
	XenditWebhookSecret string
	XenditCallbackURL   string
	XenditSuccessURL    string
	XenditFailureURL    string

	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	AccessTokenExpiryMinutes int
	RefreshTokenExpiryDays   int

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
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		NatsURL:     getEnv("NATS_URL", "nats://localhost:4222"),

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

		AccessTokenExpiryMinutes: getEnvInt("ACCESS_TOKEN_EXPIRY_MINUTES", 15),
		RefreshTokenExpiryDays:   getEnvInt("REFRESH_TOKEN_EXPIRY_DAYS", 7),

		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 50),
	}

	if cfg.AppSecret == "" {
		return nil, errors.New("APP_SECRET is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	return cfg, nil
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
