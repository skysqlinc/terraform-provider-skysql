package skysql

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
