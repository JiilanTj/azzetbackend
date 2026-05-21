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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("zenziva: API returned status %d", resp.StatusCode)
	}

	slog.Info("zenziva: OTP sent", "to", to)
	return nil
}
