package shared

import (
	"context"
	"fmt"
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

	return nil
}

type EmailOTPSender struct {
	// Uses the existing SMTP client
	Host     string
	Port     string
	User     string
	Pass     string
	From     string
}

func NewEmailOTPSender(host, port, user, pass, from string) *EmailOTPSender {
	return &EmailOTPSender{
		Host: host,
		Port: port,
		User: user,
		Pass: pass,
		From: from,
	}
}

func (e *EmailOTPSender) SendOTP(ctx context.Context, to, code string) error {
	// TODO: Integrate with existing SMTP client
	// For now, log the OTP
	fmt.Printf("[EMAIL OTP] To: %s, Code: %s\n", to, code)
	return nil
}
