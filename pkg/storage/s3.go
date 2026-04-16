// pkg/storage/s3.go
package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yyypluto/parkieee-api/pkg/resilience"
)

type Client struct {
	endpoint      string
	bucket        string
	accessKey     string
	secretKey     string
	region        string
	publicBaseURL string
	httpClient    *http.Client
	requester     *resilience.HTTPClient
}

func NewClient(endpoint, bucket, accessKey, secretKey, region, publicBaseURL string) *Client {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	policy := resilience.HTTPPolicy{
		Name:                       "s3-client",
		MaxRetries:                 2,
		BaseBackoff:                250 * time.Millisecond,
		MaxBackoff:                 1500 * time.Millisecond,
		RequestTimeout:             30 * time.Second,
		BreakerMaxRequests:         5,
		BreakerInterval:            15 * time.Second,
		BreakerTimeout:             30 * time.Second,
		BreakerConsecutiveFailures: 5,
	}

	return &Client{
		endpoint:      strings.TrimRight(endpoint, "/"),
		bucket:        bucket,
		accessKey:     accessKey,
		secretKey:     secretKey,
		region:        region,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
		httpClient:    httpClient,
		requester:     resilience.NewHTTPClient(httpClient, policy),
	}
}

func (c *Client) PutObject(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	resp, err := c.requester.Do(ctx, func(reqCtx context.Context) (*http.Request, error) {
		return c.buildPutObjectRequest(reqCtx, key, data, contentType)
	})
	if err != nil {
		return "", fmt.Errorf("storage: PUT request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("storage: PUT failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return fmt.Sprintf("%s/%s", c.publicBaseURL, key), nil
}

func (c *Client) buildPutObjectRequest(ctx context.Context, key string, data []byte, contentType string) (*http.Request, error) {
	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzDatetime := now.Format("20060102T150405Z")

	url := fmt.Sprintf("%s/%s/%s", c.endpoint, c.bucket, key)

	bodyHash := hashSHA256(data)

	headers := map[string]string{
		"content-type":         contentType,
		"host":                 hostFromURL(c.endpoint),
		"x-amz-content-sha256": bodyHash,
		"x-amz-date":           amzDatetime,
	}

	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf(
		"content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		headers["content-type"], headers["host"],
		headers["x-amz-content-sha256"], headers["x-amz-date"],
	)

	canonicalRequest := strings.Join([]string{
		"PUT",
		"/" + c.bucket + "/" + key,
		"",
		canonicalHeaders,
		signedHeaders,
		bodyHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", datestamp, c.region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDatetime,
		credentialScope,
		hashSHA256([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(c.secretKey, datestamp, c.region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.accessKey, credentialScope, signedHeaders, signature,
	)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("storage: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-date", amzDatetime)
	req.Header.Set("x-amz-content-sha256", bodyHash)
	req.Header.Set("Authorization", authHeader)

	return req, nil
}

func hashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func deriveSigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hostFromURL(u string) string {
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	return strings.Split(u, "/")[0]
}
