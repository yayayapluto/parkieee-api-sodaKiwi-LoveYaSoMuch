// pkg/midtrans/client.go
package midtrans

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yyypluto/parkieee-api/pkg/resilience"
)

type Client struct {
	serverKey  string
	clientKey  string
	baseURL    string
	httpClient *http.Client
	requester  *resilience.HTTPClient
}

func NewClient(serverKey, clientKey, env string) *Client {
	baseURL := "https://api.sandbox.midtrans.com"
	if env == "production" {
		baseURL = "https://api.midtrans.com"
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	policy := resilience.HTTPPolicy{
		Name:                       "midtrans-qris",
		MaxRetries:                 2,
		BaseBackoff:                250 * time.Millisecond,
		MaxBackoff:                 1500 * time.Millisecond,
		RequestTimeout:             15 * time.Second,
		BreakerMaxRequests:         5,
		BreakerInterval:            10 * time.Second,
		BreakerTimeout:             30 * time.Second,
		BreakerConsecutiveFailures: 6,
	}

	return &Client{
		serverKey:  serverKey,
		clientKey:  clientKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		requester:  resilience.NewHTTPClient(httpClient, policy),
	}
}

type QRISResult struct {
	QRISString   string
	QRISImageURL string
	ExpiresAt    time.Time
}

func (c *Client) CreateQRIS(ctx context.Context, orderID string, amount int) (*QRISResult, error) {
	payload := map[string]any{
		"payment_type": "qris",
		"transaction_details": map[string]any{
			"order_id":     orderID,
			"gross_amount": amount,
		},
		"qris": map[string]any{"acquirer": "gopay"},
	}

	body, _ := json.Marshal(payload)
	resp, err := c.requester.Do(ctx, func(reqCtx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(reqCtx, "POST", c.baseURL+"/v2/charge", bytes.NewReader(body))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.serverKey+":")))
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("midtrans: CreateQRIS request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("midtrans: unexpected http status %d body=%s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var result struct {
		StatusCode string `json:"status_code"`
		Actions    []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"actions"`
		TransactionTime string `json:"transaction_time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("midtrans: decode response: %w", err)
	}
	if result.StatusCode != "201" {
		return nil, fmt.Errorf("midtrans: unexpected status %s", result.StatusCode)
	}

	qr := &QRISResult{ExpiresAt: time.Now().Add(15 * time.Minute)}
	for _, a := range result.Actions {
		switch a.Name {
		case "generate-qr-code":
			qr.QRISImageURL = a.URL
		case "qr-code":
			qr.QRISString = a.URL
		}
	}
	return qr, nil
}

// ValidateCallback verifies the Midtrans signature.
// signature = SHA512(order_id + status_code + gross_amount + server_key)
func (c *Client) ValidateCallback(orderID, statusCode, grossAmount, signature string) bool {
	raw := orderID + statusCode + grossAmount + c.serverKey
	h := sha512.Sum512([]byte(raw))
	expected := fmt.Sprintf("%x", h)
	return strings.EqualFold(expected, signature)
}

// CallbackPayload is the parsed Midtrans webhook body.
type CallbackPayload struct {
	OrderID           string `json:"order_id"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
}

func ParseCallback(r io.Reader) (*CallbackPayload, error) {
	var p CallbackPayload
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}
