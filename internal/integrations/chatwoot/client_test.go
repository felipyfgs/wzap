package chatwoot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
