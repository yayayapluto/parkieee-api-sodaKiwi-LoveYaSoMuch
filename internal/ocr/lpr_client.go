package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yyypluto/parkieee-api/pkg/resilience"
)

type LPRResponse struct {
	PlateText  string  `json:"plate_text"`
	Confidence float64 `json:"confidence"`
	ImageURL   string  `json:"image_url"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	requester  *resilience.HTTPClient
}

func NewLPRClient(baseURL string) *Client {
	policy := resilience.HTTPPolicy{
		Name:                       "lpr-client",
		MaxRetries:                 2,
		BaseBackoff:                200 * time.Millisecond,
		MaxBackoff:                 1 * time.Second,
		RequestTimeout:             15 * time.Second,
		BreakerMaxRequests:         5,
		BreakerInterval:            10 * time.Second,
		BreakerTimeout:             30 * time.Second,
		BreakerConsecutiveFailures: 5,
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		requester:  resilience.NewHTTPClient(httpClient, policy),
	}
}

func (c *Client) Detect(ctx context.Context, imagePath string) (*LPRResponse, error) {
	if strings.TrimSpace(imagePath) == "" {
		return nil, fmt.Errorf("ocr: image_path is required")
	}

	body, err := json.Marshal(map[string]string{"image_path": imagePath})
	if err != nil {
		return nil, fmt.Errorf("ocr: marshal request: %w", err)
	}

	resp, err := c.requester.Do(ctx, func(reqCtx context.Context) (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(reqCtx, http.MethodPost, c.baseURL+"/detect-plate", bytes.NewReader(body))
		if reqErr != nil {
			return nil, fmt.Errorf("ocr: build request: %w", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("ocr: call detect-plate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("ocr: detect-plate returned status %d body=%s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var out LPRResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ocr: decode response: %w", err)
	}

	return &out, nil
}
