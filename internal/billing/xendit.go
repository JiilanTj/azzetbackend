package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// XenditClient handles communication with Xendit API
type XenditClient struct {
	APIKey        string
	WebhookSecret string
	CallbackURL   string
	SuccessURL    string
	FailureURL    string
	BaseURL       string
	Client        *http.Client
}

func NewXenditClient(apiKey, webhookSecret, callbackURL, successURL, failureURL string) *XenditClient {
	return &XenditClient{
		APIKey:        apiKey,
		WebhookSecret: webhookSecret,
		CallbackURL:   callbackURL,
		SuccessURL:    successURL,
		FailureURL:    failureURL,
		BaseURL:       "https://api.xendit.co",
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateInvoiceRequest is the Xendit invoice creation request
type XenditCreateInvoiceRequest struct {
	ExternalID      string  `json:"external_id"`
	Amount          float64 `json:"amount"`
	Description     string  `json:"description"`
	InvoiceDuration int     `json:"invoice_duration"` // seconds
	Currency        string  `json:"currency"`
	SuccessRedirectURL string `json:"success_redirect_url,omitempty"`
	FailureRedirectURL string `json:"failure_redirect_url,omitempty"`
	CallbackURL     string  `json:"callback_url,omitempty"`
}

// XenditInvoiceResponse is the Xendit invoice creation response
type XenditInvoiceResponse struct {
	ID         string  `json:"id"`
	ExternalID string  `json:"external_id"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
	InvoiceURL string  `json:"invoice_url"`
	ExpiryDate string  `json:"expiry_date"`
}

// XenditWebhookPayload is the callback payload from Xendit
type XenditWebhookPayload struct {
	ID                 string  `json:"id"`
	ExternalID         string  `json:"external_id"`
	Status             string  `json:"status"`
	Amount             float64 `json:"amount"`
	PaidAmount         float64 `json:"paid_amount"`
	Currency           string  `json:"currency"`
	PaymentMethod      string  `json:"payment_method"`
	PaymentChannel     string  `json:"payment_channel"`
	PaidAt             string  `json:"paid_at"`
	FailureRedirectURL string  `json:"failure_redirect_url"`
}

// CreateInvoice creates a payment invoice on Xendit
func (c *XenditClient) CreateInvoice(ctx context.Context, req *XenditCreateInvoiceRequest) (*XenditInvoiceResponse, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("xendit: API key not configured")
	}

	if req.CallbackURL == "" {
		req.CallbackURL = c.CallbackURL
	}
	if req.SuccessRedirectURL == "" {
		req.SuccessRedirectURL = c.SuccessURL
	}
	if req.FailureRedirectURL == "" {
		req.FailureRedirectURL = c.FailureURL
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("xendit: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v2/invoices", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("xendit: failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.APIKey, "")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("xendit: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("xendit: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("xendit: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var invoiceResp XenditInvoiceResponse
	if err := json.Unmarshal(respBody, &invoiceResp); err != nil {
		return nil, fmt.Errorf("xendit: failed to parse response: %w", err)
	}

	return &invoiceResp, nil
}

// VerifyWebhookSignature verifies the Xendit webhook callback signature
func (c *XenditClient) VerifyWebhookSignature(payload []byte, signature string) bool {
	if c.WebhookSecret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(c.WebhookSecret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// VerifyWebhookToken verifies using x-callback-token header (Xendit's simpler method)
func (c *XenditClient) VerifyWebhookToken(token string) bool {
	if c.WebhookSecret == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(c.WebhookSecret)) == 1
}
