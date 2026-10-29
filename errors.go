package relay

import (
	"encoding/json"
	"fmt"
)

// APIError means the API answered with an error status. Code is the API's error code, such as
// NOT_FOUND or CONFLICT. Use errors.As to get one from a returned error.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	// Details is anything extra the API sent, such as validation problems.
	Details json.RawMessage
}

func (e *APIError) Error() string {
	return fmt.Sprintf("relay: %d %s: %s", e.StatusCode, e.Code, e.Message)
}

// NetworkError means the request never got an answer: the connection failed or timed out.
type NetworkError struct {
	Message string
	Err     error
}

func (e *NetworkError) Error() string { return "relay: " + e.Message }
func (e *NetworkError) Unwrap() error { return e.Err }

// ValidationError means the SDK rejected the input before sending anything.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return "relay: " + e.Message }
