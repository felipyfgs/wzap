package wa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"

	"wzap/internal/logger"
)

const (
	waVersionUpdateTimeout     = 10 * time.Second
	waVersionRequestTimeout    = 5 * time.Second
	wppConnectVersionsURL      = "https://raw.githubusercontent.com/wppconnect-team/wa-version/main/versions.json"
	maxVersionResponseBodySize = 1 << 20
)

// ResolveWAVersion resolves an explicit version or fetches the latest available
// WhatsApp Web version. The official WhatsApp page is the primary source and
// WPPConnect is used as a fallback.
func ResolveWAVersion(ctx context.Context, configured string, httpClient *http.Client) (store.WAVersionContainer, string, error) {
	if configured = strings.TrimSpace(configured); configured != "" {
		version, err := parseWAVersion(configured)
		if err != nil {
			return store.WAVersionContainer{}, "", fmt.Errorf("invalid WA_VERSION: %w", err)
		}
		return version, "WA_VERSION", nil
	}

	latest, err := whatsmeow.GetLatestVersion(ctx, httpClient)
	if err == nil && latest != nil {
		return *latest, "web.whatsapp.com", nil
	}

	latest, fallbackErr := fetchWPPConnectVersion(ctx, httpClient)
	if fallbackErr != nil {
		return store.WAVersionContainer{}, "", fmt.Errorf("failed to fetch WhatsApp version from web.whatsapp.com and WPPConnect: %w", fallbackErr)
	}
	return *latest, "WPPConnect", nil
}

// ConfigureWAVersion resolves and applies the WhatsApp client version before
// any clients are created. Automatic lookup failures are non-fatal.
func ConfigureWAVersion(ctx context.Context, configured string, httpClient *http.Client) error {
	current := store.GetWAVersion()
	updateCtx, cancel := context.WithTimeout(ctx, waVersionUpdateTimeout)
	defer cancel()

	httpClient = versionHTTPClient(httpClient)
	version, source, err := ResolveWAVersion(updateCtx, configured, httpClient)
	if err != nil {
		if strings.TrimSpace(configured) != "" {
			return err
		}
		logger.Warn().
			Str("component", "wa").
			Err(err).
			Str("version", current.String()).
			Msg("Failed to update WhatsApp client version, using embedded version")
		return nil
	}

	if strings.TrimSpace(configured) == "" && version.LessThan(current) {
		logger.Warn().
			Str("component", "wa").
			Str("fetched_version", version.String()).
			Str("embedded_version", current.String()).
			Str("source", source).
			Msg("Fetched WhatsApp client version is older, using embedded version")
		return nil
	}

	store.SetWAVersion(version)
	logger.Info().
		Str("component", "wa").
		Str("version", version.String()).
		Str("source", source).
		Msg("Configured WhatsApp client version")
	return nil
}

func versionHTTPClient(httpClient *http.Client) *http.Client {
	if httpClient == nil {
		return &http.Client{Timeout: waVersionRequestTimeout}
	}
	if httpClient.Timeout > 0 && httpClient.Timeout <= waVersionRequestTimeout {
		return httpClient
	}
	cloned := *httpClient
	cloned.Timeout = waVersionRequestTimeout
	return &cloned
}

func fetchWPPConnectVersion(ctx context.Context, httpClient *http.Client) (*store.WAVersionContainer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wppConnectVersionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare WPPConnect version request: %w", err)
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request WPPConnect version: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WPPConnect version request returned status %d", resp.StatusCode)
	}

	var payload struct {
		CurrentVersion string `json:"currentVersion"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxVersionResponseBodySize)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode WPPConnect version response: %w", err)
	}

	version, err := parseWAVersion(payload.CurrentVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid WPPConnect version: %w", err)
	}
	return &version, nil
}

func parseWAVersion(value string) (store.WAVersionContainer, error) {
	value = strings.TrimSpace(value)
	lowerValue := strings.ToLower(value)
	for _, suffix := range []string{"-alpha", "-beta"} {
		if strings.HasSuffix(lowerValue, suffix) {
			value = value[:len(value)-len(suffix)]
			break
		}
	}

	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return store.WAVersionContainer{}, fmt.Errorf("%q must contain three dot-separated numbers", value)
	}

	var version store.WAVersionContainer
	for i, part := range parts {
		parsed, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return store.WAVersionContainer{}, fmt.Errorf("invalid numeric component %q in %q: %w", part, value, err)
		}
		version[i] = uint32(parsed)
	}
	if version.IsZero() {
		return store.WAVersionContainer{}, fmt.Errorf("version cannot be zero")
	}
	return version, nil
}
