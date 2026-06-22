package skysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// pendingStateMessage mirrors the body dbprovision-service returns today in its
// 400 pending-state rejection (ErrServiceModificationInPendingState).
const pendingStateMessage = "service modification not allowed while service is in a pending state"

// pendingErr is what handleError produces once it has classified a pending-state
// rejection into the sentinel.
func pendingErr() error {
	return fmt.Errorf("%w (SkySQL API 400: %s)", ErrorServiceInPendingState, pendingStateMessage)
}

func TestIsPendingStateError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"classified pending state", pendingErr(), true},
		{"bare sentinel", ErrorServiceInPendingState, true},
		{"unrelated validation", errors.New("SkySQL API 400: invalid size sky-99x99"), false},
		// Detection is by sentinel (set in handleError), not by raw string match.
		{"raw message is not classified", errors.New(pendingStateMessage), false},
		{"service not found", ErrorServiceNotFound, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPendingStateError(tc.err); got != tc.want {
				t.Fatalf("isPendingStateError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestDoWithPendingRetry_RetriesUntilSuccess(t *testing.T) {
	c := &Client{pendingRetryInterval: time.Millisecond, pendingRetryTimeout: time.Second}

	calls := 0
	err := c.doWithPendingRetry(context.Background(), func() error {
		calls++
		if calls < 3 {
			return pendingErr()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDoWithPendingRetry_SuccessFirstTry(t *testing.T) {
	c := &Client{pendingRetryInterval: time.Millisecond, pendingRetryTimeout: time.Second}

	calls := 0
	if err := c.doWithPendingRetry(context.Background(), func() error { calls++; return nil }); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got %d", calls)
	}
}

func TestDoWithPendingRetry_NonPendingReturnsImmediately(t *testing.T) {
	c := &Client{pendingRetryInterval: time.Millisecond, pendingRetryTimeout: time.Second}
	other := errors.New("SkySQL API 400: invalid size")

	calls := 0
	err := c.doWithPendingRetry(context.Background(), func() error { calls++; return other })
	if !errors.Is(err, other) {
		t.Fatalf("expected the original error to pass through, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call (no retry), got %d", calls)
	}
}

func TestDoWithPendingRetry_StopsWhenContextCancelled(t *testing.T) {
	c := &Client{pendingRetryInterval: time.Millisecond, pendingRetryTimeout: time.Minute}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	// The HTTP request inside fn observes ctx, so once cancelled it fails fast
	// with a context error rather than another pending-state error.
	err := c.doWithPendingRetry(ctx, func() error {
		calls++
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return pendingErr()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDoWithPendingRetry_TimesOut(t *testing.T) {
	c := &Client{pendingRetryInterval: time.Millisecond, pendingRetryTimeout: 5 * time.Millisecond}

	calls := 0
	err := c.doWithPendingRetry(context.Background(), func() error { calls++; return pendingErr() })
	if err == nil {
		t.Fatal("expected an error after timeout, got nil")
	}
	if !isPendingStateError(err) {
		t.Fatalf("timed-out error should still read as a pending-state error, got %v", err)
	}
	if calls < 1 {
		t.Fatalf("expected at least 1 call, got %d", calls)
	}
}

// TestModifyServiceSize_RetriesWhilePending exercises the full client path
// against current DPS behavior: a scaling call rejected with the pending-state
// 400 a couple of times, then accepted. This is the MCDEV-3899 race between a
// config change and a scaling operation.
func TestModifyServiceSize_RetriesWhilePending(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Errors: []ErrorDetails{{Message: pendingStateMessage}},
			})
			return
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"id": "svc-123"})
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.pendingRetryInterval = time.Millisecond
	client.pendingRetryTimeout = 5 * time.Second

	if err := client.ModifyServiceSize(context.Background(), "svc-123", "sky-4x16"); err != nil {
		t.Fatalf("expected success after pending-state retries, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("expected 3 attempts (2 pending + 1 accepted), got %d", got)
	}
}

// TestModifyServiceSize_RetriesOn409Conflict exercises forward compatibility
// with the planned DPS change: a 409 Conflict is retried even though its body
// does not mention a pending state — detection is by status, not message.
func TestModifyServiceSize_RetriesOn409Conflict(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{
				Errors: []ErrorDetails{{Message: "operation already in progress"}},
			})
			return
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"id": "svc-123"})
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.pendingRetryInterval = time.Millisecond
	client.pendingRetryTimeout = 5 * time.Second

	if err := client.ModifyServiceSize(context.Background(), "svc-123", "sky-4x16"); err != nil {
		t.Fatalf("expected success after 409 retries, got %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("expected 3 attempts (2 conflict + 1 accepted), got %d", got)
	}
}

// TestModifyServiceSize_DoesNotRetryValidation400 guards against the wrapper
// masking real client errors: a genuine validation 400 must fail immediately.
func TestModifyServiceSize_DoesNotRetryValidation400(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Errors: []ErrorDetails{{Message: "invalid size sky-99x99"}},
		})
	}))
	defer srv.Close()

	client := New(srv.URL, "test-key", "")
	client.pendingRetryInterval = time.Millisecond
	client.pendingRetryTimeout = 5 * time.Second

	if err := client.ModifyServiceSize(context.Background(), "svc-123", "sky-99x99"); err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("expected exactly 1 attempt (no retry on validation 400), got %d", got)
	}
}
