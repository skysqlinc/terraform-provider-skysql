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
