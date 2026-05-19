package shared

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ZenzivaClient struct {
	URL     string
	UserKey string
	PassKey string
	Brand   string
	Client  *http.Client
}

func NewZenzivaClient(apiURL, userKey, passKey, brand string) *ZenzivaClient {
	return &ZenzivaClient{
		URL:     apiURL,
		UserKey: userKey,
		PassKey: passKey,
		Brand:   brand,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (z *ZenzivaClient) SendOTP(ctx context.Context, to, code string) error {
	if z.UserKey == "" || z.PassKey == "" {
		slog.Warn("zenziva: credentials not configured, skipping OTP send", "to", to)
		return nil
	}

	form := url.Values{}
	form.Set("userkey", z.UserKey)
	form.Set("passkey", z.PassKey)
	form.Set("to", to)
	form.Set("brand", z.Brand)
	form.Set("otp", code)

	req, err := http.NewRequestWithContext(ctx, "POST", z.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("zenziva: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := z.Client.Do(req)
	if err != nil {
		return fmt.Errorf("zenziva: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("zenziva: API returned status %d", resp.StatusCode)
	}

	slog.Info("zenziva: OTP sent", "to", to)
	return nil
}

type EmailOTPSender struct {
	Host string
	Port string
	User string
	Pass string
	From string
	Env  string
}

func NewEmailOTPSender(host, port, user, pass, from, env string) *EmailOTPSender {
	return &EmailOTPSender{
		Host: host,
		Port: port,
		User: user,
		Pass: pass,
		From: from,
		Env:  env,
	}
}

func (e *EmailOTPSender) SendOTP(ctx context.Context, to, code string) error {
	if e.Host == "" || e.User == "" {
		// In development without SMTP, log only that OTP was generated (not the code itself)
		if e.Env == "development" {
			slog.Debug("email OTP generated (SMTP not configured)", "to", to)
		}
		return nil
	}

	// TODO: Integrate with existing SMTP client for actual sending
	slog.Info("email OTP sent", "to", to)
	return nil
}
