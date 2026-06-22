package skysql

import "errors"

// ErrorResponse struct
type ErrorResponse struct {
	Errors    []ErrorDetails `json:"errors"`
	Exception string         `json:"exception"`
	Path      string         `json:"path"`
	Code      int            `json:"code"`
	Timestamp FlexInt        `json:"timestamp"`
	TraceID   string         `json:"trace_id"`
}

// ErrorDetails for detailed error and message
type ErrorDetails struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	Solution string `json:"solution,omitempty"`
	Type     string `json:"type,omitempty"`
	Location string `json:"location,omitempty"`
}

var ErrorServiceNotFound = errors.New("service not found")

var ErrorUnauthorized = errors.New("skysql returns unauthorized error")

// ErrorServiceInPendingState indicates the backend (DPS status gate,
// ensureThatServiceStatusAllowsUpdates) rejected a mutating operation because
// the target service is currently in a pending_* state — another operation is
// already in flight. DPS surfaces this as a 400 with a distinctive message in
// current versions and as a 409 Conflict in newer ones; handleError classifies
// both into this sentinel so mutating calls can wait it out and retry
// (see doWithPendingRetry, MCDEV-3899).
var ErrorServiceInPendingState = errors.New("service is in a pending state")
