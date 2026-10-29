package relay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WebhooksService manages the endpoints Relay sends events to.
type WebhooksService struct{ c *Client }

// Create registers an endpoint. Store the returned Secret: it is shown only once.
func (s *WebhooksService) Create(ctx context.Context, req CreateWebhookRequest) (*CreateWebhookResponse, error) {
	out := new(CreateWebhookResponse)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: "/sdk/webhooks", body: req}, out)
}

// List returns your endpoints.
func (s *WebhooksService) List(ctx context.Context) (*ListWebhooksResponse, error) {
	out := new(ListWebhooksResponse)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/webhooks"}, out)
}

// Get returns one endpoint and its delivery counts.
func (s *WebhooksService) Get(ctx context.Context, webhookID string) (*Webhook, error) {
	id, err := pathID("webhookId", webhookID)
	if err != nil {
		return nil, err
	}
	out := new(Webhook)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/webhooks/" + id}, out)
}

// Update changes an endpoint's URL, events or description.
func (s *WebhooksService) Update(ctx context.Context, webhookID string, req UpdateWebhookRequest) (*Webhook, error) {
	id, err := pathID("webhookId", webhookID)
	if err != nil {
		return nil, err
	}
	out := new(Webhook)
	return out, s.c.do(ctx, request{method: http.MethodPut, path: "/sdk/webhooks/" + id, body: req}, out)
}

// Delete removes an endpoint and returns it as it was.
func (s *WebhooksService) Delete(ctx context.Context, webhookID string) (*Webhook, error) {
	id, err := pathID("webhookId", webhookID)
	if err != nil {
		return nil, err
	}
	out := new(Webhook)
	return out, s.c.do(ctx, request{method: http.MethodDelete, path: "/sdk/webhooks/" + id}, out)
}

// DefaultWebhookTolerance is how old a delivery may be before VerifyWebhookSignature rejects it.
const DefaultWebhookTolerance = 300 * time.Second

// WebhookVerification is the result of checking a webhook signature.
type WebhookVerification struct {
	Valid bool
	// Error says why the signature was rejected.
	Error string
}

// VerifyWebhookSignature checks that a webhook came from Relay and wasn't changed on the way.
// Pass the raw request body exactly as received, the X-Relay-Signature header, and the whsec_
// secret from Create.
//
//	result := relay.VerifyWebhookSignature(body, r.Header.Get("X-Relay-Signature"), secret, relay.DefaultWebhookTolerance)
func VerifyWebhookSignature(payload []byte, signatureHeader, secret string, tolerance time.Duration) WebhookVerification {
	var timestampPart, signaturePart string
	for _, part := range strings.Split(signatureHeader, ",") {
		switch {
		case strings.HasPrefix(part, "t=") && timestampPart == "":
			timestampPart = part[2:]
		case strings.HasPrefix(part, "v1=") && signaturePart == "":
			signaturePart = part[3:]
		}
	}
	if timestampPart == "" || signaturePart == "" {
		return WebhookVerification{Error: "Invalid signature format"}
	}
	timestamp, err := strconv.ParseInt(timestampPart, 10, 64)
	if err != nil {
		return WebhookVerification{Error: "Invalid signature format"}
	}
	age := time.Since(time.Unix(timestamp, 0))
	if age < 0 {
		age = -age
	}
	if age > tolerance {
		return WebhookVerification{Error: "Timestamp outside tolerance window"}
	}

	// The key is everything after `whsec_` (mode prefix included), base64url-decoded.
	key, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(strings.Replace(secret, "whsec_", "", 1), "="))
	if err != nil {
		return WebhookVerification{Error: "Invalid webhook secret"}
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signaturePart), []byte(expected)) {
		return WebhookVerification{Error: "Signature verification failed"}
	}
	return WebhookVerification{Valid: true}
}

// VerifySignature is VerifyWebhookSignature, for when a client is already in hand.
func (s *WebhooksService) VerifySignature(payload []byte, signatureHeader, secret string, tolerance time.Duration) WebhookVerification {
	return VerifyWebhookSignature(payload, signatureHeader, secret, tolerance)
}
