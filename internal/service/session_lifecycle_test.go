package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"

	"wzap/internal/dto"
	"wzap/internal/model"
)

type stubLifecycleManager struct {
	connectClient    *whatsmeow.Client
	connectQR        <-chan whatsmeow.QRChannelItem
	connectErr       error
	connectCalls     int
	disconnectErr    error
	disconnectCalls  int
	logoutErr        error
	logoutCalls      int
	reconnectErr     error
	reconnectCalls   int
	qrCode           string
	qrErr            error
	pairCode         string
	pairErr          error
	pairCalls        int
}

func (m *stubLifecycleManager) Connect(ctx context.Context, sessionID string) (*whatsmeow.Client, <-chan whatsmeow.QRChannelItem, error) {
	m.connectCalls++
	return m.connectClient, m.connectQR, m.connectErr
}

func (m *stubLifecycleManager) Disconnect(sessionID string) error {
	m.disconnectCalls++
	return m.disconnectErr
}

func (m *stubLifecycleManager) Logout(ctx context.Context, sessionID string) error {
	m.logoutCalls++
	return m.logoutErr
}

func (m *stubLifecycleManager) Reconnect(ctx context.Context, sessionID string) error {
	m.reconnectCalls++
	return m.reconnectErr
}

func (m *stubLifecycleManager) GetQRCode(ctx context.Context, sessionID string) (string, error) {
	return m.qrCode, m.qrErr
}

func (m *stubLifecycleManager) PairPhone(ctx context.Context, sessionID, phone string) (string, error) {
	m.pairCalls++
	return m.pairCode, m.pairErr
}

type stubSessionReader struct {
	resp  *dto.SessionResp
	err   error
	calls int
}

func (s *stubSessionReader) Get(ctx context.Context, id string) (*dto.SessionResp, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func TestSessionLifecycleOrchestratorConnectCloudPartial(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{ID: "sess-cloud", Engine: "cloud_api", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	manager := &stubLifecycleManager{}
	orchestrator := newSessionLifecycleOrchestrator(newSessionRuntimeResolver(repo, nil, nil), manager, nil)

	result, err := orchestrator.Connect(context.Background(), "sess-cloud")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "CONNECTED" {
		t.Fatalf("expected CONNECTED, got %s", result.Status)
	}
	if result.Support != model.CapabilitySupportPartial {
		t.Fatalf("expected partial support, got %s", result.Support)
	}
	if manager.connectCalls != 0 {
		t.Fatalf("expected manager Connect not to be called, got %d", manager.connectCalls)
	}
}

func TestSessionLifecycleOrchestratorConnectWhatsmeowPairing(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{ID: "sess-wa", Engine: "whatsmeow", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	manager := &stubLifecycleManager{connectQR: make(chan whatsmeow.QRChannelItem)}
	orchestrator := newSessionLifecycleOrchestrator(newSessionRuntimeResolver(repo, nil, nil), manager, nil)

	result, err := orchestrator.Connect(context.Background(), "sess-wa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PAIRING" {
		t.Fatalf("expected PAIRING, got %s", result.Status)
	}
	if result.Support != model.CapabilitySupportComplete {
		t.Fatalf("expected complete support, got %s", result.Support)
	}
	if manager.connectCalls != 1 {
		t.Fatalf("expected manager Connect to be called once, got %d", manager.connectCalls)
	}
}

func TestSessionLifecycleOrchestratorQRByEngine(t *testing.T) {
	cloudRepo := &stubSessionLookup{
		session: &model.Session{ID: "sess-cloud", Engine: "cloud_api", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	cloudOrchestrator := newSessionLifecycleOrchestrator(newSessionRuntimeResolver(cloudRepo, nil, nil), &stubLifecycleManager{}, nil)

	_, err := cloudOrchestrator.QR(context.Background(), "sess-cloud")
	if err == nil {
		t.Fatal("expected capability error for cloud_api QR")
	}
	var capabilityErr *CapabilityError
	if !errors.As(err, &capabilityErr) {
		t.Fatalf("expected CapabilityError, got %T", err)
	}

	waRepo := &stubSessionLookup{
		session: &model.Session{ID: "sess-wa", Engine: "whatsmeow", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	waOrchestrator := newSessionLifecycleOrchestrator(newSessionRuntimeResolver(waRepo, nil, nil), &stubLifecycleManager{qrCode: "qr-code"}, nil)

	result, err := waOrchestrator.QR(context.Background(), "sess-wa")
	if err != nil {
		t.Fatalf("unexpected QR error: %v", err)
	}
	if result.QRCode != "qr-code" {
		t.Fatalf("expected qr-code, got %s", result.QRCode)
	}
}

func TestSessionLifecycleOrchestratorPairByEngine(t *testing.T) {
	cloudRepo := &stubSessionLookup{
		session: &model.Session{ID: "sess-cloud", Engine: "cloud_api", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	cloudOrchestrator := newSessionLifecycleOrchestrator(newSessionRuntimeResolver(cloudRepo, nil, nil), &stubLifecycleManager{}, nil)

	_, err := cloudOrchestrator.Pair(context.Background(), "sess-cloud", "5511999999999")
	if err == nil {
		t.Fatal("expected capability error for cloud_api pair")
	}
	var capabilityErr *CapabilityError
	if !errors.As(err, &capabilityErr) {
		t.Fatalf("expected CapabilityError, got %T", err)
	}

	waRepo := &stubSessionLookup{
		session: &model.Session{ID: "sess-wa", Engine: "whatsmeow", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	manager := &stubLifecycleManager{pairCode: "123-456"}
	waOrchestrator := newSessionLifecycleOrchestrator(newSessionRuntimeResolver(waRepo, nil, nil), manager, nil)

	result, err := waOrchestrator.Pair(context.Background(), "sess-wa", "5511999999999")
	if err != nil {
		t.Fatalf("unexpected pair error: %v", err)
	}
	if result.PairingCode != "123-456" {
		t.Fatalf("expected pairing code 123-456, got %s", result.PairingCode)
	}
	if manager.pairCalls != 1 {
		t.Fatalf("expected PairPhone to be called once, got %d", manager.pairCalls)
	}
}

func TestSessionLifecycleOrchestratorRestartByEngine(t *testing.T) {
	cloudRepo := &stubSessionLookup{
		session: &model.Session{ID: "sess-cloud", Engine: "cloud_api", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	cloudReader := &stubSessionReader{resp: &dto.SessionResp{ID: "sess-cloud", Engine: "cloud_api"}}
	cloudManager := &stubLifecycleManager{}
	cloudOrchestrator := newSessionLifecycleOrchestrator(newSessionRuntimeResolver(cloudRepo, nil, nil), cloudManager, cloudReader)

	cloudResult, err := cloudOrchestrator.Restart(context.Background(), "sess-cloud")
	if err != nil {
		t.Fatalf("unexpected restart error for cloud_api: %v", err)
	}
	if cloudResult.Session == nil || cloudResult.Session.ID != "sess-cloud" {
		t.Fatal("expected cloud restart to return existing session")
	}
	if cloudManager.disconnectCalls != 0 || cloudManager.connectCalls != 0 {
		t.Fatal("expected cloud_api restart not to call manager disconnect/connect")
	}

	waRepo := &stubSessionLookup{
		session: &model.Session{ID: "sess-wa", Engine: "whatsmeow", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	waReader := &stubSessionReader{resp: &dto.SessionResp{ID: "sess-wa", Engine: "whatsmeow"}}
	waManager := &stubLifecycleManager{}
	waOrchestrator := newSessionLifecycleOrchestrator(newSessionRuntimeResolver(waRepo, nil, nil), waManager, waReader)

	waResult, err := waOrchestrator.Restart(context.Background(), "sess-wa")
	if err != nil {
		t.Fatalf("unexpected restart error for whatsmeow: %v", err)
	}
	if waResult.Session == nil || waResult.Session.ID != "sess-wa" {
		t.Fatal("expected whatsmeow restart to return refreshed session")
	}
	if waManager.disconnectCalls != 1 {
		t.Fatalf("expected Disconnect to be called once, got %d", waManager.disconnectCalls)
	}
	if waManager.connectCalls != 1 {
		t.Fatalf("expected Connect to be called once, got %d", waManager.connectCalls)
	}
}
