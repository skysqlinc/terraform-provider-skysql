package skysql

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew_WithOrgID_SetsHeader(t *testing.T) {
	var receivedOrgHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedOrgHeader = r.Header.Get("X-MDB-Org")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client := New(ts.URL, "test-api-key", "org-12345")

	// Verify the header is set on the client
	if got := client.HTTPClient.Header.Get("X-MDB-Org"); got != "org-12345" {
		t.Errorf("expected client header X-MDB-Org = %q, got %q", "org-12345", got)
	}

	// Verify the header is sent in actual requests
	_, _ = client.GetProjects(t.Context())
	if receivedOrgHeader != "org-12345" {
		t.Errorf("expected request header X-MDB-Org = %q, got %q", "org-12345", receivedOrgHeader)
	}
}

func TestNew_WithoutOrgID_NoHeader(t *testing.T) {
	var receivedOrgHeader string
	var hasOrgHeader bool

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedOrgHeader = r.Header.Get("X-MDB-Org")
		_, hasOrgHeader = r.Header["X-Mdb-Org"]
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client := New(ts.URL, "test-api-key", "")

	// Verify the header is NOT set on the client
	if got := client.HTTPClient.Header.Get("X-MDB-Org"); got != "" {
		t.Errorf("expected no X-MDB-Org header on client, got %q", got)
	}

	// Verify the header is NOT sent in actual requests
	_, _ = client.GetProjects(t.Context())
	if hasOrgHeader {
		t.Errorf("expected no X-MDB-Org header in request, got %q", receivedOrgHeader)
	}
}

func TestNew_SetsAPIKeyHeader(t *testing.T) {
	var receivedAPIKey string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client := New(ts.URL, "my-secret-key", "")

	_, _ = client.GetProjects(t.Context())
	if receivedAPIKey != "my-secret-key" {
		t.Errorf("expected request header X-API-Key = %q, got %q", "my-secret-key", receivedAPIKey)
	}
}

func TestNew_OrgIDHeaderSentOnAllRequests(t *testing.T) {
	requestCount := 0
	orgHeaders := make([]string, 0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgHeaders = append(orgHeaders, r.Header.Get("X-MDB-Org"))
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch {
		case r.URL.Path == "/organization/v1/projects":
			w.Write([]byte(`[]`))
		case r.URL.Path == "/provisioning/v1/services/svc-123":
			w.Write([]byte(`{"id":"svc-123","name":"test"}`))
		default:
			w.Write([]byte(`{}`))
		}
	}))
	defer ts.Close()

	client := New(ts.URL, "test-api-key", "org-multi")

	// Make two different API calls
	_, _ = client.GetProjects(t.Context())
	_, _ = client.GetServiceByID(t.Context(), "svc-123")

	if requestCount < 2 {
		t.Fatalf("expected at least 2 requests, got %d", requestCount)
	}

	for i, h := range orgHeaders {
		if h != "org-multi" {
			t.Errorf("request %d: expected X-MDB-Org = %q, got %q", i, "org-multi", h)
		}
	}
}

func TestRetryOn500(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Errors: []ErrorDetails{{Message: "transient failure"}},
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// 1 initial + 3 retries = 4 total attempts.
	got := atomic.LoadInt32(&attempts)
	if got != 4 {
		t.Errorf("expected 4 attempts (1 + 3 retries), got %d", got)
	}
}

func TestRetryOn502(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	got := atomic.LoadInt32(&attempts)
	if got != 4 {
		t.Errorf("expected 4 attempts for 502, got %d", got)
	}
}

func TestRetryOn503(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	got := atomic.LoadInt32(&attempts)
	if got != 4 {
		t.Errorf("expected 4 attempts for 503, got %d", got)
	}
}

func TestRetryOn504(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	got := atomic.LoadInt32(&attempts)
	if got != 4 {
		t.Errorf("expected 4 attempts for 504, got %d", got)
	}
}

func TestRetryOn429(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	got := atomic.LoadInt32(&attempts)
	if got != 4 {
		t.Errorf("expected 4 attempts for 429, got %d", got)
	}
}

func TestRetryOn429HonorsRetryAfterHeader(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "svc-123",
			"name":   "test-service",
			"status": "ready",
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(5 * time.Second)

	start := time.Now()
	svc, err := client.GetServiceByID(t.Context(), "svc-123")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if svc.ID != "svc-123" {
		t.Errorf("expected service ID svc-123, got %s", svc.ID)
	}

	got := atomic.LoadInt32(&attempts)
	if got != 2 {
		t.Errorf("expected 2 attempts (1 rate-limited + 1 success), got %d", got)
	}

	if elapsed < 900*time.Millisecond {
		t.Errorf("expected retry to honor Retry-After: 1 (~1s delay), but elapsed was %v", elapsed)
	}
}

func TestRetrySucceedsAfterTransientFailure(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Errors: []ErrorDetails{{Message: "transient"}},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "svc-123",
			"name":   "test-service",
			"status": "ready",
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	svc, err := client.GetServiceByID(t.Context(), "svc-123")
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if svc.ID != "svc-123" {
		t.Errorf("expected service ID svc-123, got %s", svc.ID)
	}

	got := atomic.LoadInt32(&attempts)
	if got != 3 {
		t.Errorf("expected 3 attempts (2 failures + 1 success), got %d", got)
	}
}

func TestNoRetryOn400(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Errors: []ErrorDetails{{Message: "bad request"}},
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	got := atomic.LoadInt32(&attempts)
	if got != 1 {
		t.Errorf("expected exactly 1 attempt for 400 (no retry), got %d", got)
	}
}

func TestNoRetryOn404(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if !errors.Is(err, ErrorServiceNotFound) {
		t.Errorf("expected ErrorServiceNotFound, got: %v", err)
	}

	got := atomic.LoadInt32(&attempts)
	if got != 1 {
		t.Errorf("expected exactly 1 attempt for 404 (no retry), got %d", got)
	}
}

func TestHandleError401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if !errors.Is(err, ErrorUnauthorized) {
		t.Errorf("expected ErrorUnauthorized, got: %v", err)
	}
}

func TestHandleError500IncludesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Errors: []ErrorDetails{{Message: "database connection pool exhausted"}},
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "database connection pool exhausted") {
		t.Errorf("expected error to contain API message, got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain status code, got: %q", err.Error())
	}
}

func TestHandleErrorEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.HTTPClient.SetRetryWaitTime(time.Millisecond)
	client.HTTPClient.SetRetryMaxWaitTime(time.Millisecond)

	_, err := client.GetServiceByID(t.Context(), "svc-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention 500, got: %q", err.Error())
	}
}
