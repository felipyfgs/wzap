package elodesk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPClient_UpsertContact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("api_access_token"); got != "tok-abc" {
			t.Errorf("missing or wrong api_access_token: %q", got)
		}
		if !strings.HasPrefix(r.URL.Path, "/public/api/v1/inboxes/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":42,"name":"Jane","sourceId":"11988887777@s.whatsapp.net"},"message":"success"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "tok-abc", nil)
	got, err := c.UpsertContact(context.Background(), "identifier-xyz", UpsertContactReq{
		Identifier:  "11988887777@s.whatsapp.net",
		Name:        "Jane",
		PhoneNumber: "+11988887777",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("id: got %d, want 42", got.ID)
	}
}

func TestHTTPClient_CreateMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body MessageReq
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Content != "oi" {
			t.Errorf("content: got %q, want %q", body.Content, "oi")
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":99,"content":"oi","conversationId":7,"sourceId":"WAID:ABC"},"message":"success"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, "tok-abc", nil)
	got, err := c.CreateMessage(context.Background(), "identifier-xyz", "contact-src-id", 7, MessageReq{
		Content:     "oi",
		MessageType: "incoming",
		SourceID:    "WAID:ABC",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.ID != 99 || got.ConversationID != 7 {
		t.Errorf("got id=%d conv=%d, want 99/7", got.ID, got.ConversationID)
	}
}

func TestHTTPClient_5xxReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := NewClient(server.URL, "tok-abc", nil)
	_, err := c.CreateMessage(context.Background(), "id", "src", 1, MessageReq{Content: "x"})
	if err == nil {
		t.Fatal("expected err, got nil")
	}
	if !isRetryableError(err) {
		t.Errorf("expected 5xx to be retryable, got non-retryable: %v", err)
	}
}

func TestHTTPClient_4xxNotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	c := NewClient(server.URL, "tok-abc", nil)
	_, err := c.CreateMessage(context.Background(), "id", "src", 1, MessageReq{Content: "x"})
	if err == nil {
		t.Fatal("expected err, got nil")
	}
	if isRetryableError(err) {
		t.Errorf("expected 400 to be non-retryable, got retryable: %v", err)
	}
}

func TestHTTPClient_UpdateConversationStatus(t *testing.T) {
	t.Run("success_204", func(t *testing.T) {
		var gotPath, gotStatus, gotAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("api_access_token")
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotStatus = body["status"]
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		c := NewClient(server.URL, "tok", nil)
		if err := c.UpdateConversationStatus(context.Background(), "xyz", "5511999@s.whatsapp.net", 42, "open"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		want := "/public/api/v1/inboxes/xyz/contacts/5511999@s.whatsapp.net/conversations/42/toggle_status"
		if gotPath != want {
			t.Errorf("path: got %q, want %q", gotPath, want)
		}
		if gotStatus != "open" {
			t.Errorf("status: %s", gotStatus)
		}
		if gotAuth != "tok" {
			t.Errorf("api_access_token header: got %q, want %q", gotAuth, "tok")
		}
	})

	t.Run("error_404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"conversation not found"}`, http.StatusNotFound)
		}))
		defer server.Close()

		c := NewClient(server.URL, "tok", nil)
		err := c.UpdateConversationStatus(context.Background(), "xyz", "src", 42, "open")
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T: %v", err, err)
		}
		if apiErr.StatusCode != http.StatusNotFound {
			t.Errorf("status code: got %d, want 404", apiErr.StatusCode)
		}
	})

	t.Run("error_500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer server.Close()

		c := NewClient(server.URL, "tok", nil)
		err := c.UpdateConversationStatus(context.Background(), "xyz", "src", 42, "open")
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T: %v", err, err)
		}
		if apiErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status code: got %d, want 500", apiErr.StatusCode)
		}
	})
}
