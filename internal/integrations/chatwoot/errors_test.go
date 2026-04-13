package chatwoot

import (
	"context"
	"errors"
	"testing"
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
