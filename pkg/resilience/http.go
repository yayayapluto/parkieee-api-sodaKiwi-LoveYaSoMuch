package resilience

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/sony/gobreaker/v2"
)

var (
	defaultRetryableStatusCodes = StatusSet(http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout)
)

type HTTPPolicy struct {
	Name                       string
	MaxRetries                 int
	BaseBackoff                time.Duration
	MaxBackoff                 time.Duration
	RequestTimeout             time.Duration
	BreakerMaxRequests         uint32
	BreakerInterval            time.Duration
	BreakerTimeout             time.Duration
	BreakerConsecutiveFailures uint32
	RetryableStatusCodes       map[int]struct{}
}

type HTTPClient struct {
	httpClient *http.Client
	policy     HTTPPolicy
	breaker    *gobreaker.CircuitBreaker[*http.Response]
}

type RetryableStatusError struct {
	StatusCode int
}

func (e *RetryableStatusError) Error() string {
	return "retryable http status: " + http.StatusText(e.StatusCode)
}

func StatusSet(codes ...int) map[int]struct{} {
	out := make(map[int]struct{}, len(codes))
	for _, code := range codes {
		out[code] = struct{}{}
	}
	return out
}

func NewHTTPClient(httpClient *http.Client, policy HTTPPolicy) *HTTPClient {
	normalized := normalizePolicy(policy)
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	breakerSettings := gobreaker.Settings{
		Name:        normalized.Name,
		MaxRequests: normalized.BreakerMaxRequests,
		Interval:    normalized.BreakerInterval,
		Timeout:     normalized.BreakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(normalized.BreakerConsecutiveFailures)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			slog.Info("circuit breaker state changed", "name", name, "from", from.String(), "to", to.String())
		},
	}

	return &HTTPClient{
		httpClient: httpClient,
		policy:     normalized,
		breaker:    gobreaker.NewCircuitBreaker[*http.Response](breakerSettings),
	}
}

func (c *HTTPClient) Do(ctx context.Context, buildRequest func(context.Context) (*http.Request, error)) (*http.Response, error) {
	attemptCount := c.policy.MaxRetries + 1
	if attemptCount < 1 {
		attemptCount = 1
	}

	var lastErr error
	for attempt := 0; attempt < attemptCount; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		resp, err := c.breaker.Execute(func() (*http.Response, error) {
			attemptCtx := ctx
			if c.policy.RequestTimeout > 0 {
				var cancel context.CancelFunc
				attemptCtx, cancel = context.WithTimeout(ctx, c.policy.RequestTimeout)
				defer cancel()
			}

			req, err := buildRequest(attemptCtx)
			if err != nil {
				return nil, err
			}

			res, err := c.httpClient.Do(req)
			if err != nil {
				return nil, err
			}

			if c.isRetryableStatus(res.StatusCode) {
				_, _ = io.Copy(io.Discard, res.Body)
				_ = res.Body.Close()
				return nil, &RetryableStatusError{StatusCode: res.StatusCode}
			}

			return res, nil
		})
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if attempt == attemptCount-1 || !c.isRetryableError(err) {
			return nil, err
		}

		delay := c.backoffDuration(attempt)
		if waitErr := waitWithContext(ctx, delay); waitErr != nil {
			return nil, waitErr
		}
	}

	return nil, lastErr
}

func (c *HTTPClient) isRetryableStatus(status int) bool {
	_, ok := c.policy.RetryableStatusCodes[status]
	return ok
}

func (c *HTTPClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var statusErr *RetryableStatusError
	if errors.As(err, &statusErr) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}

func (c *HTTPClient) backoffDuration(attempt int) time.Duration {
	base := c.policy.BaseBackoff
	delay := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if delay > c.policy.MaxBackoff {
		delay = c.policy.MaxBackoff
	}
	if delay <= 0 {
		return base
	}

	jitterWindow := delay / 4
	if jitterWindow > 0 {
		jitter := time.Duration(time.Now().UnixNano() % int64(jitterWindow))
		delay += jitter
	}
	return delay
}

func normalizePolicy(in HTTPPolicy) HTTPPolicy {
	out := in
	if out.Name == "" {
		out.Name = "http-client"
	}
	if out.MaxRetries < 0 {
		out.MaxRetries = 0
	}
	if out.BaseBackoff <= 0 {
		out.BaseBackoff = 200 * time.Millisecond
	}
	if out.MaxBackoff <= 0 {
		out.MaxBackoff = 2 * time.Second
	}
	if out.MaxBackoff < out.BaseBackoff {
		out.MaxBackoff = out.BaseBackoff
	}
	if out.RequestTimeout <= 0 {
		out.RequestTimeout = 15 * time.Second
	}
	if out.BreakerMaxRequests == 0 {
		out.BreakerMaxRequests = 5
	}
	if out.BreakerInterval <= 0 {
		out.BreakerInterval = 10 * time.Second
	}
	if out.BreakerTimeout <= 0 {
		out.BreakerTimeout = 30 * time.Second
	}
	if out.BreakerConsecutiveFailures == 0 {
		out.BreakerConsecutiveFailures = 5
	}
	if out.RetryableStatusCodes == nil {
		out.RetryableStatusCodes = make(map[int]struct{}, len(defaultRetryableStatusCodes))
		for code := range defaultRetryableStatusCodes {
			out.RetryableStatusCodes[code] = struct{}{}
		}
	}
	return out
}

func waitWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
