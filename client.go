// Package relay is the official Relay server SDK for Go: quote and create deliveries, read them
// back with the rider's live ETA, manage webhooks and mint tracking tokens for your apps.
//
//	client, err := relay.NewClient(os.Getenv("RELAY_API_KEY"))
//	quote, err := client.Tasks.Quote(ctx, relay.TaskQuoteRequest{...})
//
// Keep the API key on your server. Every call takes a context for cancellation and deadlines.
package relay

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	mathrand "math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL    = "https://api.sendrelay.com.ng"
	DefaultAPIVersion = "v1"
)

// RetryConfig controls how failed requests are retried: timeouts, network failures, and 408,
// 429 and 5xx responses. Test-mode quota errors are never retried.
type RetryConfig struct {
	Enabled              bool
	MaxRetries           int
	InitialDelay         time.Duration
	MaxDelay             time.Duration
	BackoffMultiplier    float64
	RetryableStatusCodes []int
}

// DefaultRetryConfig is used unless you pass WithRetry or WithMaxRetries.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Enabled:              true,
		MaxRetries:           3,
		InitialDelay:         500 * time.Millisecond,
		MaxDelay:             10 * time.Second,
		BackoffMultiplier:    2,
		RetryableStatusCodes: []int{408, 429, 500, 502, 503, 504},
	}
}

var nonRetryableErrorCodes = map[string]bool{"TEST_QUOTA_EXCEEDED": true, "TEST_CONCURRENCY_EXCEEDED": true}

type config struct {
	apiKey          string
	baseURL         string
	apiVersion      string
	timeout         time.Duration
	retry           RetryConfig
	autoIdempotency bool
	headers         map[string]string
	httpClient      *http.Client
}

// Option configures a Client.
type Option func(*config)

// WithBaseURL points the client at another deployment, such as a local API.
func WithBaseURL(baseURL string) Option {
	return func(c *config) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithAPIVersion overrides the API version path segment (default "v1").
func WithAPIVersion(version string) Option { return func(c *config) { c.apiVersion = version } }

// WithTimeout sets the per-request timeout (default 30s).
func WithTimeout(timeout time.Duration) Option { return func(c *config) { c.timeout = timeout } }

// WithMaxRetries sets how many times a failed request is retried (default 3).
func WithMaxRetries(n int) Option { return func(c *config) { c.retry.MaxRetries = n } }

// WithRetry replaces the whole retry configuration.
func WithRetry(retry RetryConfig) Option { return func(c *config) { c.retry = retry } }

// WithoutAutoIdempotency stops the client generating an Idempotency-Key for POST requests.
func WithoutAutoIdempotency() Option { return func(c *config) { c.autoIdempotency = false } }

// WithHeaders adds headers to every request.
func WithHeaders(headers map[string]string) Option {
	return func(c *config) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// WithHTTPClient uses your own *http.Client, for proxies, tracing or tests.
func WithHTTPClient(client *http.Client) Option { return func(c *config) { c.httpClient = client } }

// Client is the Relay API client. It is safe for concurrent use.
type Client struct {
	cfg      config
	Tasks    *TasksService
	Webhooks *WebhooksService
	Auth     *AuthService
	Presence *PresenceService
}

// NewClient creates a client for a live (sk_live_...) or test (sk_test_...) API key.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, &ValidationError{Field: "apiKey", Message: "API key is required"}
	}
	if !strings.HasPrefix(apiKey, "sk_live_") && !strings.HasPrefix(apiKey, "sk_test_") {
		return nil, &ValidationError{Field: "apiKey", Message: "Invalid API key format. Must start with sk_live_ or sk_test_"}
	}
	cfg := config{
		apiKey:          apiKey,
		baseURL:         DefaultBaseURL,
		apiVersion:      DefaultAPIVersion,
		timeout:         30 * time.Second,
		retry:           DefaultRetryConfig(),
		autoIdempotency: true,
		headers:         map[string]string{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{}
	}
	c := &Client{cfg: cfg}
	c.Tasks = &TasksService{c}
	c.Webhooks = &WebhooksService{c}
	c.Auth = &AuthService{c}
	c.Presence = &PresenceService{c}
	return c, nil
}

// Environment is "live" for a live key and "test" for a test key.
func (c *Client) Environment() string {
	if strings.HasPrefix(c.cfg.apiKey, "sk_live_") {
		return "live"
	}
	return "test"
}

type request struct {
	method         string
	path           string
	query          url.Values
	body           any
	idempotencyKey string
}

func (c *Client) do(ctx context.Context, req request, out any) error {
	var payload []byte
	if req.body != nil {
		var err error
		if payload, err = json.Marshal(req.body); err != nil {
			return fmt.Errorf("relay: encode request: %w", err)
		}
	}
	idempotencyKey := req.idempotencyKey
	if idempotencyKey == "" && req.method == http.MethodPost && c.cfg.autoIdempotency {
		idempotencyKey = generateIdempotencyKey()
	}

	for attempt := 0; ; attempt++ {
		resp, err := c.send(ctx, req, payload, idempotencyKey)
		if err != nil {
			if ctx.Err() != nil {
				return &NetworkError{Message: "request cancelled", Err: ctx.Err()}
			}
			if c.cfg.retry.Enabled && attempt < c.cfg.retry.MaxRetries {
				if waitErr := c.wait(ctx, attempt); waitErr != nil {
					return waitErr
				}
				continue
			}
			var timeout interface{ Timeout() bool }
			if errors.As(err, &timeout) && timeout.Timeout() {
				return &NetworkError{Message: fmt.Sprintf("Request timeout after %s", c.cfg.timeout), Err: err}
			}
			return &NetworkError{Message: "Network error - connection failed", Err: err}
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return &NetworkError{Message: "reading response failed", Err: readErr}
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out == nil || len(body) == 0 {
				return nil
			}
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("relay: decode response: %w", err)
			}
			return nil
		}

		apiErr := toAPIError(resp, body)
		if c.retryable(apiErr, attempt) {
			if waitErr := c.wait(ctx, attempt); waitErr != nil {
				return waitErr
			}
			continue
		}
		return apiErr
	}
}

func (c *Client) send(ctx context.Context, req request, payload []byte, idempotencyKey string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.timeout)
	u := c.cfg.baseURL + "/" + c.cfg.apiVersion + req.path
	if len(req.query) > 0 {
		u += "?" + req.query.Encode()
	}
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.method, u, body)
	if err != nil {
		cancel()
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Relay-Key", c.cfg.apiKey)
	httpReq.Header.Set("User-Agent", "relay-sdk-go/"+Version)
	for k, v := range c.cfg.headers {
		httpReq.Header.Set(k, v)
	}
	if idempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := c.cfg.httpClient.Do(httpReq)
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = cancelOnClose{resp.Body, cancel}
	return resp, nil
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b cancelOnClose) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

func (c *Client) retryable(err *APIError, attempt int) bool {
	if !c.cfg.retry.Enabled || attempt >= c.cfg.retry.MaxRetries || nonRetryableErrorCodes[err.Code] {
		return false
	}
	for _, code := range c.cfg.retry.RetryableStatusCodes {
		if code == err.StatusCode {
			return true
		}
	}
	return false
}

func (c *Client) wait(ctx context.Context, attempt int) error {
	r := c.cfg.retry
	delay := math.Min(float64(r.InitialDelay)*math.Pow(r.BackoffMultiplier, float64(attempt)), float64(r.MaxDelay))
	// ±25% jitter so many clients don't retry in lockstep.
	delay += delay * 0.25 * (mathrand.Float64()*2 - 1)
	timer := time.NewTimer(time.Duration(math.Max(0, delay)))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return &NetworkError{Message: "request cancelled", Err: ctx.Err()}
	case <-timer.C:
		return nil
	}
}

func toAPIError(resp *http.Response, body []byte) *APIError {
	var envelope struct {
		Error struct {
			Code    string          `json:"code"`
			Message string          `json:"message"`
			Details json.RawMessage `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	e := &APIError{StatusCode: resp.StatusCode, Code: envelope.Error.Code, Message: envelope.Error.Message, Details: envelope.Error.Details}
	if e.Code == "" {
		e.Code = "UNKNOWN_ERROR"
	}
	if e.Message == "" {
		e.Message = http.StatusText(resp.StatusCode)
	}
	if string(e.Details) == "null" {
		e.Details = nil
	}
	return e
}

func generateIdempotencyKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("sdk-%d-%s", time.Now().UnixMilli(), hex.EncodeToString(b))
}

func pathID(name, value string) (string, error) {
	if value == "" {
		return "", &ValidationError{Field: name, Message: name + " is required"}
	}
	return url.PathEscape(value), nil
}
