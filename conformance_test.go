package relay

// Runs the shared SDK conformance fixtures (packages/sdks/conformance) against this SDK. Every
// Relay SDK runs the same fixtures, which is how they stay at parity.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func conformanceDir() string {
	if dir := os.Getenv("RELAY_CONFORMANCE_DIR"); dir != "" {
		return dir
	}
	return filepath.Join("..", "conformance")
}

func readJSON(t *testing.T, name string, out any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(conformanceDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

type exchange struct {
	Request struct {
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Query   map[string]string `json:"query"`
		Body    any               `json:"body"`
		Headers map[string]string `json:"headers"`
	} `json:"request"`
	Response struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	} `json:"response"`
}

type fixture struct {
	Name      string `json:"name"`
	Operation string `json:"operation"`
	Input     struct {
		TaskID         string          `json:"taskId"`
		WebhookID      string          `json:"webhookId"`
		Body           json.RawMessage `json:"body"`
		Query          map[string]any  `json:"query"`
		IdempotencyKey string          `json:"idempotencyKey"`
	} `json:"input"`
	Exchanges []exchange `json:"exchanges"`
	Expect    struct {
		Result map[string]any `json:"result"`
		Items  []string       `json:"items"`
		Error  map[string]any `json:"error"`
	} `json:"expect"`
}

func decode[T any](raw json.RawMessage) T {
	var v T
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	return v
}

func queryInt(q map[string]any, key string) int {
	if v, ok := q[key].(float64); ok {
		return int(v)
	}
	return 0
}

func queryString(q map[string]any, key string) string {
	v, _ := q[key].(string)
	return v
}

// bindings maps every manifest operation to a call on this SDK.
var bindings = map[string]func(ctx context.Context, c *Client, f fixture) (any, error){
	"tasks.quote": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Tasks.Quote(ctx, decode[TaskQuoteRequest](f.Input.Body))
	},
	"tasks.create": func(ctx context.Context, c *Client, f fixture) (any, error) {
		var opts *CreateOptions
		if f.Input.IdempotencyKey != "" {
			opts = &CreateOptions{IdempotencyKey: f.Input.IdempotencyKey}
		}
		return c.Tasks.Create(ctx, decode[CreateTaskRequest](f.Input.Body), opts)
	},
	"tasks.list": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Tasks.List(ctx, &ListTasksParams{Limit: queryInt(f.Input.Query, "limit"), Order: queryString(f.Input.Query, "order"), Cursor: queryString(f.Input.Query, "cursor")})
	},
	"tasks.listAll": func(ctx context.Context, c *Client, f fixture) (any, error) {
		ids := []string{}
		for task, err := range c.Tasks.ListAll(ctx, &ListTasksParams{Limit: queryInt(f.Input.Query, "limit")}) {
			if err != nil {
				return nil, err
			}
			ids = append(ids, task.ID)
		}
		return ids, nil
	},
	"tasks.get": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Tasks.Get(ctx, f.Input.TaskID)
	},
	"tasks.cancel": func(ctx context.Context, c *Client, f fixture) (any, error) {
		var req *CancelTaskRequest
		if len(f.Input.Body) > 0 {
			r := decode[CancelTaskRequest](f.Input.Body)
			req = &r
		}
		return c.Tasks.Cancel(ctx, f.Input.TaskID, req)
	},
	"tasks.assign": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Tasks.Assign(ctx, f.Input.TaskID, decode[ManualAssignRequest](f.Input.Body))
	},
	"tasks.availableRiders": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Tasks.AvailableRiders(ctx, f.Input.TaskID, &AvailableRidersParams{Tier: queryInt(f.Input.Query, "tier")})
	},
	"tasks.rate": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Tasks.Rate(ctx, f.Input.TaskID, decode[RatingRequest](f.Input.Body))
	},
	"tasks.dispute": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Tasks.Dispute(ctx, f.Input.TaskID, decode[DisputeRequest](f.Input.Body))
	},
	"webhooks.create": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Webhooks.Create(ctx, decode[CreateWebhookRequest](f.Input.Body))
	},
	"webhooks.list": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Webhooks.List(ctx)
	},
	"webhooks.get": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Webhooks.Get(ctx, f.Input.WebhookID)
	},
	"webhooks.update": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Webhooks.Update(ctx, f.Input.WebhookID, decode[UpdateWebhookRequest](f.Input.Body))
	},
	"webhooks.delete": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Webhooks.Delete(ctx, f.Input.WebhookID)
	},
	"auth.createWebSocketToken": func(ctx context.Context, c *Client, f fixture) (any, error) {
		return c.Auth.CreateWebSocketToken(ctx, decode[CreateWebSocketTokenRequest](f.Input.Body))
	},
	"presence.get": func(ctx context.Context, c *Client, f fixture) (any, error) {
		lat, _ := f.Input.Query["latitude"].(float64)
		lng, _ := f.Input.Query["longitude"].(float64)
		return c.Presence.Get(ctx, PresenceParams{Latitude: lat, Longitude: lng, Resolution: queryString(f.Input.Query, "resolution")})
	},
}

func lookup(value any, dotted string) any {
	node := value
	for _, key := range strings.Split(dotted, ".") {
		switch n := node.(type) {
		case map[string]any:
			node = n[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i >= len(n) {
				return nil
			}
			node = n[i]
		default:
			return nil
		}
	}
	return node
}

func asJSON(v any) any {
	data, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(data, &out)
	return out
}

func expectPaths(t *testing.T, actual any, expected map[string]any) {
	t.Helper()
	for key, want := range expected {
		got := lookup(actual, key)
		if !reflect.DeepEqual(got, asJSON(want)) {
			t.Errorf("%s: got %#v, want %#v", key, got, want)
		}
	}
}

func errorJSON(err error) (string, map[string]any) {
	var apiErr *APIError
	var netErr *NetworkError
	var valErr *ValidationError
	switch {
	case errors.As(err, &apiErr):
		var details any
		_ = json.Unmarshal(apiErr.Details, &details)
		return "ApiError", map[string]any{"statusCode": float64(apiErr.StatusCode), "code": apiErr.Code, "message": apiErr.Message, "details": details}
	case errors.As(err, &netErr):
		return "NetworkError", map[string]any{"message": netErr.Message}
	case errors.As(err, &valErr):
		return "ValidationError", map[string]any{"message": valErr.Message}
	}
	return fmt.Sprintf("%T", err), nil
}

// roundTrip answers a fixture's exchanges in order and checks each request.
type roundTrip struct {
	t       *testing.T
	queue   []exchange
	common  map[string]string
	calls   int
	failNet bool
}

func (r *roundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	r.calls++
	if len(r.queue) == 0 {
		r.t.Fatalf("unexpected request %s %s", req.Method, req.URL)
	}
	want := r.queue[0]
	r.queue = r.queue[1:]
	if req.Method != want.Request.Method {
		r.t.Errorf("method: got %s, want %s", req.Method, want.Request.Method)
	}
	if req.URL.Path != want.Request.Path {
		r.t.Errorf("path: got %s, want %s", req.URL.Path, want.Request.Path)
	}
	query := map[string]string{}
	for k, v := range req.URL.Query() {
		query[k] = v[0]
	}
	if !reflect.DeepEqual(query, want.Request.Query) && !(len(query) == 0 && len(want.Request.Query) == 0) {
		r.t.Errorf("query: got %v, want %v", query, want.Request.Query)
	}
	var body any
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		if len(data) > 0 {
			_ = json.Unmarshal(data, &body)
		}
	}
	if !reflect.DeepEqual(body, want.Request.Body) {
		got, _ := json.Marshal(body)
		exp, _ := json.Marshal(want.Request.Body)
		r.t.Errorf("body:\n got  %s\n want %s", got, exp)
	}
	headers := map[string]string{}
	for k, v := range r.common {
		headers[k] = v
	}
	for k, v := range want.Request.Headers {
		headers[k] = v
	}
	for name, value := range headers {
		got := req.Header.Get(name)
		switch value {
		case "<absent>":
			if got != "" {
				r.t.Errorf("header %s should be absent, got %q", name, got)
			}
		case "<any>":
			if got == "" {
				r.t.Errorf("header %s missing", name)
			}
		default:
			if got != value {
				r.t.Errorf("header %s: got %q, want %q", name, got, value)
			}
		}
	}
	return &http.Response{
		StatusCode: want.Response.Status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(want.Response.Body))),
		Request:    req,
	}, nil
}

type manifest struct {
	Server struct {
		APIKey        string            `json:"apiKey"`
		BaseURL       string            `json:"baseUrl"`
		CommonHeaders map[string]string `json:"commonHeaders"`
		Operations    []struct {
			ID    string `json:"id"`
			Local bool   `json:"local"`
		} `json:"operations"`
	} `json:"server"`
}

func TestConformance(t *testing.T) {
	if _, err := os.Stat(filepath.Join(conformanceDir(), "manifest.json")); err != nil {
		t.Skip("conformance folder not found")
	}
	var m manifest
	readJSON(t, "manifest.json", &m)

	var fixtures []fixture
	_ = filepath.WalkDir(filepath.Join(conformanceDir(), "http"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".json") {
			var f fixture
			rel, _ := filepath.Rel(conformanceDir(), path)
			readJSON(t, rel, &f)
			fixtures = append(fixtures, f)
		}
		return nil
	})

	t.Run("manifest is fully bound", func(t *testing.T) {
		for _, op := range m.Server.Operations {
			if op.Local {
				continue
			}
			if bindings[op.ID] == nil {
				t.Errorf("missing binding for %s", op.ID)
			}
			found := false
			for _, f := range fixtures {
				found = found || f.Operation == op.ID
			}
			if !found {
				t.Errorf("no fixture for %s", op.ID)
			}
		}
	})

	for _, f := range fixtures {
		t.Run(f.Operation+": "+f.Name, func(t *testing.T) {
			rt := &roundTrip{t: t, queue: f.Exchanges, common: m.Server.CommonHeaders}
			client, err := NewClient(m.Server.APIKey, WithBaseURL(m.Server.BaseURL), WithRetry(RetryConfig{Enabled: false}), WithHTTPClient(&http.Client{Transport: rt}))
			if err != nil {
				t.Fatal(err)
			}
			result, err := bindings[f.Operation](context.Background(), client, f)
			if f.Expect.Error != nil {
				want := map[string]any{}
				for k, v := range f.Expect.Error {
					want[k] = v
				}
				wantType := want["type"]
				delete(want, "type")
				kind, fields := errorJSON(err)
				if kind != wantType {
					t.Fatalf("error type: got %s (%v), want %v", kind, err, wantType)
				}
				expectPaths(t, fields, want)
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if f.Expect.Items != nil && !reflect.DeepEqual(result, f.Expect.Items) {
					t.Errorf("items: got %v, want %v", result, f.Expect.Items)
				}
				if f.Expect.Result != nil {
					expectPaths(t, asJSON(result), f.Expect.Result)
				}
			}
			if len(rt.queue) != 0 {
				t.Errorf("%d unanswered exchanges", len(rt.queue))
			}
		})
	}
}

type clientFixtures struct {
	Defaults struct {
		BaseURL    string `json:"baseUrl"`
		APIVersion string `json:"apiVersion"`
		TimeoutMs  int    `json:"timeoutMs"`
		Retry      struct {
			Enabled              bool    `json:"enabled"`
			MaxRetries           int     `json:"maxRetries"`
			InitialDelayMs       int     `json:"initialDelayMs"`
			MaxDelayMs           int     `json:"maxDelayMs"`
			BackoffMultiplier    float64 `json:"backoffMultiplier"`
			RetryableStatusCodes []int   `json:"retryableStatusCodes"`
		} `json:"retry"`
		Idempotency struct {
			AutoGenerate bool `json:"autoGenerate"`
		} `json:"idempotency"`
	} `json:"defaults"`
	APIKeys []struct {
		APIKey      string `json:"apiKey"`
		Environment string `json:"environment"`
		Error       string `json:"error"`
	} `json:"apiKeys"`
	Errors struct {
		Cases []struct {
			Name           string          `json:"name"`
			Status         int             `json:"status"`
			Body           json.RawMessage `json:"body"`
			RawBody        *string         `json:"rawBody"`
			NetworkFailure bool            `json:"networkFailure"`
			Attempts       int             `json:"attempts"`
			Expect         map[string]any  `json:"expect"`
		} `json:"cases"`
	} `json:"errors"`
}

type answerFunc func(*http.Request) (*http.Response, error)

func (f answerFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConformanceClient(t *testing.T) {
	if _, err := os.Stat(filepath.Join(conformanceDir(), "client.json")); err != nil {
		t.Skip("conformance folder not found")
	}
	var c clientFixtures
	readJSON(t, "client.json", &c)

	t.Run("defaults", func(t *testing.T) {
		client, _ := NewClient("sk_test_abc")
		d, cfg := c.Defaults, client.cfg
		if cfg.baseURL != d.BaseURL || cfg.apiVersion != d.APIVersion || cfg.timeout != time.Duration(d.TimeoutMs)*time.Millisecond {
			t.Errorf("connection defaults differ: %+v", cfg)
		}
		r := cfg.retry
		if r.Enabled != d.Retry.Enabled || r.MaxRetries != d.Retry.MaxRetries ||
			r.InitialDelay != time.Duration(d.Retry.InitialDelayMs)*time.Millisecond ||
			r.MaxDelay != time.Duration(d.Retry.MaxDelayMs)*time.Millisecond ||
			r.BackoffMultiplier != d.Retry.BackoffMultiplier ||
			!reflect.DeepEqual(r.RetryableStatusCodes, d.Retry.RetryableStatusCodes) {
			t.Errorf("retry defaults differ: %+v", r)
		}
		if cfg.autoIdempotency != d.Idempotency.AutoGenerate {
			t.Error("idempotency default differs")
		}
	})

	for _, k := range c.APIKeys {
		t.Run("api key "+strconv.Quote(k.APIKey), func(t *testing.T) {
			client, err := NewClient(k.APIKey)
			if k.Error != "" {
				var v *ValidationError
				if !errors.As(err, &v) {
					t.Fatalf("want ValidationError, got %v", err)
				}
				return
			}
			if err != nil || client.Environment() != k.Environment {
				t.Fatalf("environment: got %v (%v), want %s", client, err, k.Environment)
			}
		})
	}

	for _, tc := range c.Errors.Cases {
		t.Run("errors: "+tc.Name, func(t *testing.T) {
			calls := 0
			transport := answerFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.NetworkFailure {
					return nil, errors.New("connection refused")
				}
				body := string(tc.Body)
				if tc.RawBody != nil {
					body = *tc.RawBody
				}
				return &http.Response{StatusCode: tc.Status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: r}, nil
			})
			client, _ := NewClient("sk_test_abc", WithRetry(RetryConfig{Enabled: true, MaxRetries: 1, BackoffMultiplier: 1, RetryableStatusCodes: DefaultRetryConfig().RetryableStatusCodes}), WithHTTPClient(&http.Client{Transport: transport}))
			_, err := client.Tasks.Get(context.Background(), "t_1")
			want := map[string]any{}
			for k, v := range tc.Expect {
				want[k] = v
			}
			wantType := want["type"]
			delete(want, "type")
			kind, fields := errorJSON(err)
			if kind != wantType {
				t.Fatalf("error type: got %s (%v), want %v", kind, err, wantType)
			}
			expectPaths(t, fields, want)
			if calls != tc.Attempts {
				t.Errorf("attempts: got %d, want %d", calls, tc.Attempts)
			}
		})
	}

	var sigs struct {
		Cases []struct {
			Name      string `json:"name"`
			Secret    string `json:"secret"`
			Payload   string `json:"payload"`
			Header    string `json:"header"`
			Tolerance int64  `json:"tolerance"`
			Valid     bool   `json:"valid"`
			Error     string `json:"error"`
		} `json:"cases"`
	}
	readJSON(t, "webhooks/signatures.json", &sigs)
	for _, tc := range sigs.Cases {
		t.Run("webhook signature: "+tc.Name, func(t *testing.T) {
			// The vectors use a huge tolerance to mean "any clock"; cap it so it fits a Duration.
			tolerance := time.Duration(math.MaxInt64)
			if tc.Tolerance < int64(tolerance/time.Second) {
				tolerance = time.Duration(tc.Tolerance) * time.Second
			}
			result := VerifyWebhookSignature([]byte(tc.Payload), tc.Header, tc.Secret, tolerance)
			if result.Valid != tc.Valid || (!tc.Valid && result.Error != tc.Error) {
				t.Errorf("got %+v, want valid=%v error=%q", result, tc.Valid, tc.Error)
			}
		})
	}
}
