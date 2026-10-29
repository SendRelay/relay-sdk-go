package relay

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// AuthService mints tokens for apps to track tasks live.
type AuthService struct{ c *Client }

// CreateWebSocketToken returns a short-lived token for one of the UI SDKs. Scope it to the tasks
// the app may see: []string{"task:" + taskID}.
func (s *AuthService) CreateWebSocketToken(ctx context.Context, req CreateWebSocketTokenRequest) (*WebSocketToken, error) {
	if len(req.Scope) == 0 {
		return nil, &ValidationError{Field: "scope", Message: "scope must contain at least one resource"}
	}
	for _, resource := range req.Scope {
		parts := strings.Split(resource, ":")
		if len(parts) != 2 {
			return nil, &ValidationError{Field: "scope", Message: fmt.Sprintf(`Invalid scope format: "%s". Expected "type:id" (e.g., "task:abc123")`, resource)}
		}
		if parts[0] == "" || parts[1] == "" {
			return nil, &ValidationError{Field: "scope", Message: fmt.Sprintf(`Invalid scope format: "%s". Both type and id must be non-empty`, resource)}
		}
		if parts[0] == "task" && len(strings.Split(parts[1], ",")) > 5 {
			return nil, &ValidationError{Field: "scope", Message: fmt.Sprintf(`Scoped tasks cannot exceed 5 per scope entry: "%s"`, resource)}
		}
	}
	if req.ExpiresIn == 0 {
		req.ExpiresIn = 1800
	} else if req.ExpiresIn < 60 || req.ExpiresIn > 7200 {
		return nil, &ValidationError{Field: "expiresIn", Message: "expiresIn must be between 60 and 7200 seconds"}
	}
	out := new(WebSocketToken)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: "/auth/websocket-token", body: req}, out)
}

// PresenceService shows anonymous rider availability, for maps shown before creating a task.
type PresenceService struct{ c *Client }

// Get returns rider density around a point.
func (s *PresenceService) Get(ctx context.Context, params PresenceParams) (*PresenceResponse, error) {
	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(params.Latitude, 'f', -1, 64))
	q.Set("longitude", strconv.FormatFloat(params.Longitude, 'f', -1, 64))
	if params.Resolution != "" {
		q.Set("resolution", params.Resolution)
	}
	out := new(PresenceResponse)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/presence", query: q}, out)
}
