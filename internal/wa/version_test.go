package wa_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow/store"

	wzapwa "wzap/internal/wa"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func response(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}
}

func TestResolveWAVersionOverrideSkipsNetwork(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		expected   string
	}{
		{name: "numeric", configured: "2.3000.1044062641", expected: "2.3000.1044062641"},
		{name: "alpha", configured: "2.3000.1044062641-alpha", expected: "2.3000.1044062641"},
		{name: "beta case insensitive", configured: "2.3000.1044062641-BETA", expected: "2.3000.1044062641"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("HTTP request must not be made when WA_VERSION is configured")
				return nil, errors.New("unexpected request")
			})}

			version, source, err := wzapwa.ResolveWAVersion(context.Background(), tt.configured, client)
			if err != nil {
				t.Fatalf("ResolveWAVersion returned error: %v", err)
			}
			if version.String() != tt.expected {
				t.Fatalf("expected version %s, got %s", tt.expected, version.String())
			}
			if source != "WA_VERSION" {
				t.Fatalf("expected WA_VERSION source, got %q", source)
			}
		})
	}
}

func TestResolveWAVersionRejectsInvalidOverride(t *testing.T) {
	invalid := []string{
		"1.2",
		"1.2.nope",
		"0.0.0",
		"-1.2.3",
		"1.2.4294967296",
		"1.2.3-rc",
	}

	for _, configured := range invalid {
		t.Run(configured, func(t *testing.T) {
			_, _, err := wzapwa.ResolveWAVersion(context.Background(), configured, nil)
			if err == nil {
				t.Fatalf("expected %q to be rejected", configured)
			}
			if !strings.Contains(err.Error(), "invalid WA_VERSION") {
				t.Fatalf("expected WA_VERSION context in error, got %v", err)
			}
		})
	}
}

func TestResolveWAVersionUsesWhatsAppFirst(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "web.whatsapp.com" {
			t.Fatalf("expected web.whatsapp.com request, got %s", req.URL.Host)
		}
		return response(req, http.StatusOK, `{"client_revision":1044062641,}`), nil
	})}

	version, source, err := wzapwa.ResolveWAVersion(context.Background(), "", client)
	if err != nil {
		t.Fatalf("ResolveWAVersion returned error: %v", err)
	}
	if version.String() != "2.3000.1044062641" {
		t.Fatalf("unexpected version %s", version.String())
	}
	if source != "web.whatsapp.com" {
		t.Fatalf("expected primary source, got %q", source)
	}
}

func TestResolveWAVersionFallsBackToWPPConnect(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "web.whatsapp.com":
			return response(req, http.StatusBadGateway, "unavailable"), nil
		case "raw.githubusercontent.com":
			return response(req, http.StatusOK, `{"currentVersion":"2.3000.1044062641-alpha"}`), nil
		default:
			t.Fatalf("unexpected host %s", req.URL.Host)
			return nil, errors.New("unexpected host")
		}
	})}

	version, source, err := wzapwa.ResolveWAVersion(context.Background(), "", client)
	if err != nil {
		t.Fatalf("ResolveWAVersion returned error: %v", err)
	}
	if version.String() != "2.3000.1044062641" {
		t.Fatalf("unexpected version %s", version.String())
	}
	if source != "WPPConnect" {
		t.Fatalf("expected WPPConnect source, got %q", source)
	}
}

func TestConfigureWAVersionKeepsEmbeddedVersionWhenSourcesFail(t *testing.T) {
	original := store.GetWAVersion()
	t.Cleanup(func() { store.SetWAVersion(original) })

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return response(req, http.StatusServiceUnavailable, "unavailable"), nil
	})}

	if err := wzapwa.ConfigureWAVersion(context.Background(), "", client); err != nil {
		t.Fatalf("automatic failure must be non-fatal: %v", err)
	}
	if got := store.GetWAVersion(); got != original {
		t.Fatalf("expected embedded version %s, got %s", original.String(), got.String())
	}
}

func TestConfigureWAVersionPreventsAutomaticDowngrade(t *testing.T) {
	original := store.GetWAVersion()
	t.Cleanup(func() { store.SetWAVersion(original) })

	current := store.WAVersionContainer{2, 3000, 1044062641}
	store.SetWAVersion(current)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return response(req, http.StatusOK, `{"client_revision":1035920091,}`), nil
	})}

	if err := wzapwa.ConfigureWAVersion(context.Background(), "", client); err != nil {
		t.Fatalf("ConfigureWAVersion returned error: %v", err)
	}
	if got := store.GetWAVersion(); got != current {
		t.Fatalf("expected current version %s, got %s", current.String(), got.String())
	}
}

func TestConfigureWAVersionAllowsManualDowngrade(t *testing.T) {
	original := store.GetWAVersion()
	t.Cleanup(func() { store.SetWAVersion(original) })

	store.SetWAVersion(store.WAVersionContainer{2, 3000, 1044062641})
	configured := store.WAVersionContainer{2, 3000, 1035920091}
	if err := wzapwa.ConfigureWAVersion(context.Background(), configured.String(), nil); err != nil {
		t.Fatalf("ConfigureWAVersion returned error: %v", err)
	}
	if got := store.GetWAVersion(); got != configured {
		t.Fatalf("expected configured version %s, got %s", configured.String(), got.String())
	}
}
