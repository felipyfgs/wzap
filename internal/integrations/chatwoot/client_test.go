package chatwoot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsRetryableError_Nil(t *testing.T) {
	if isRetryableError(nil) {
		t.Error("expected false for nil error")
	}
}

func TestIsRetryableError_Status500(t *testing.T) {
	err := &APIError{StatusCode: 500, Message: "internal server error"}
	if !isRetryableError(err) {
		t.Error("expected true for status=500")
	}
}

func TestIsRetryableError_Status429(t *testing.T) {
	err := &APIError{StatusCode: 429, Message: "too many requests"}
	if !isRetryableError(err) {
		t.Error("expected true for status=429")
	}
}

func TestIsRetryableError_ContextTimeout(t *testing.T) {
	if !isRetryableError(context.DeadlineExceeded) {
		t.Error("expected true for context.DeadlineExceeded")
	}
	if !isRetryableError(context.Canceled) {
		t.Error("expected true for context.Canceled")
	}
}

func TestIsRetryableError_Status404(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "not found"}
	if isRetryableError(err) {
		t.Error("expected false for status=404")
	}
}

func TestIsRetryableError_Status422(t *testing.T) {
	err := &APIError{StatusCode: 422, Message: "unprocessable entity"}
	if isRetryableError(err) {
		t.Error("expected false for status=422")
	}
}

func TestIsRetryableError_NetworkError(t *testing.T) {
	err := errors.New("do request: dial tcp: connection refused")
	if !isRetryableError(err) {
		t.Error("expected true for network/do request error")
	}
}

func TestCircuitBreaker_ClosedByDefault(t *testing.T) {
	cb := newCircuitBreaker()
	if !cb.Allow() {
		t.Error("expected circuit breaker to allow requests in CLOSED state")
	}
	if cb.State() != cbClosed {
		t.Errorf("expected state CLOSED, got %d", cb.State())
	}
}

func TestCircuitBreaker_OpenAfterThreshold(t *testing.T) {
	cb := newCircuitBreaker()
	for range cbThreshold {
		cb.RecordFailure()
	}
	if cb.State() != cbOpen {
		t.Errorf("expected state OPEN after %d failures, got %d", cbThreshold, cb.State())
	}
	if cb.Allow() {
		t.Error("expected circuit breaker to reject requests in OPEN state")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := newCircuitBreaker()
	for range cbThreshold {
		cb.RecordFailure()
	}
	cb.mu.Lock()
	cb.lastFail = time.Now().Add(-(cbTimeout + time.Second))
	cb.mu.Unlock()

	if !cb.Allow() {
		t.Error("expected circuit breaker to allow in HALF_OPEN after timeout")
	}
	if cb.State() != cbHalfOpen {
		t.Errorf("expected state HALF_OPEN, got %d", cb.State())
	}
}

func TestCircuitBreaker_ClosedAfterSuccessInHalfOpen(t *testing.T) {
	cb := newCircuitBreaker()
	for range cbThreshold {
		cb.RecordFailure()
	}
	cb.mu.Lock()
	cb.lastFail = time.Now().Add(-(cbTimeout + time.Second))
	cb.mu.Unlock()
	cb.Allow() // transitions to HALF_OPEN

	cb.RecordSuccess()
	if cb.State() != cbClosed {
		t.Errorf("expected state CLOSED after success in HALF_OPEN, got %d", cb.State())
	}
	if !cb.Allow() {
		t.Error("expected circuit breaker to allow requests after recovery")
	}
}

func TestCircuitBreaker_BackToOpenOnFailureInHalfOpen(t *testing.T) {
	cb := newCircuitBreaker()
	for range cbThreshold {
		cb.RecordFailure()
	}
	cb.mu.Lock()
	cb.lastFail = time.Now().Add(-(cbTimeout + time.Second))
	cb.mu.Unlock()
	cb.Allow() // transitions to HALF_OPEN

	cb.RecordFailure()
	if cb.State() != cbOpen {
		t.Errorf("expected state OPEN after failure in HALF_OPEN, got %d", cb.State())
	}
}

func TestCircuitBreakerManager_PerSession(t *testing.T) {
	mgr := newCircuitBreakerManager()

	if !mgr.Allow("session-1") {
		t.Error("expected session-1 to be allowed initially")
	}
	if !mgr.Allow("session-2") {
		t.Error("expected session-2 to be allowed initially")
	}

	for range cbThreshold {
		mgr.RecordFailure("session-1")
	}

	if mgr.Allow("session-1") {
		t.Error("expected session-1 to be blocked after threshold failures")
	}
	if !mgr.Allow("session-2") {
		t.Error("expected session-2 to still be allowed (independent circuit breaker)")
	}
}

func TestCircuitBreakerManager_RecordSuccess(t *testing.T) {
	mgr := newCircuitBreakerManager()
	for range cbThreshold {
		mgr.RecordFailure("sess")
	}
	mgr.get("sess").mu.Lock()
	mgr.get("sess").lastFail = time.Now().Add(-(cbTimeout + time.Second))
	mgr.get("sess").mu.Unlock()
	mgr.Allow("sess") // HALF_OPEN

	mgr.RecordSuccess("sess")
	if !mgr.Allow("sess") {
		t.Error("expected sess to be allowed after recovery")
	}
}

func TestClient_MergeContacts(t *testing.T) {
	var receivedBody map[string]int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expected := "/api/v1/accounts/1/actions/contact_merge"
		if r.URL.Path != expected {
			t.Errorf("expected path %s, got %s", expected, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 1, "test-token", server.Client())
	err := client.MergeContacts(context.Background(), 10, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["base_contact_id"] != 10 {
		t.Errorf("expected base_contact_id=10, got %d", receivedBody["base_contact_id"])
	}
	if receivedBody["mergee_contact_id"] != 20 {
		t.Errorf("expected mergee_contact_id=20, got %d", receivedBody["mergee_contact_id"])
	}
}

func TestClient_MergeContacts_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"invalid"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 1, "test-token", server.Client())
	err := client.MergeContacts(context.Background(), 10, 20)
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
}
