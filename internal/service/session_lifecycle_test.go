package service

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"

	"wzap/internal/dto"
	"wzap/internal/model"
)

type stubLifecycleManager struct {
	connectClient   *whatsmeow.Client
	connectQR       <-chan whatsmeow.QRChannelItem
	connectErr      error
	connectCalls    int
	disconnectErr   error
	disconnectCalls int
	logoutErr       error
	logoutCalls     int
	reconnectErr    error
	reconnectCalls  int
	qrCode          string
	qrErr           error
	pairCode        string
	pairErr         error
	pairCalls       int
}

func (m *stubLifecycleManager) Connect(ctx context.Context, sessionID string) (*whatsmeow.Client, <-chan whatsmeow.QRChannelItem, error) {
	m.connectCalls++
	return m.connectClient, m.connectQR, m.connectErr
}

func (m *stubLifecycleManager) Disconnect(ctx context.Context, sessionID string) error {
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

func TestLifecycleOrchestratorConnectWhatsmeowPairing(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{ID: "sess-wa", Engine: "whatsmeow", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	manager := &stubLifecycleManager{connectQR: make(chan whatsmeow.QRChannelItem)}
	orchestrator := newSessionLifecycle(&RuntimeResolver{repo: repo}, manager, nil)

	result, err := orchestrator.Connect(context.Background(), "sess-wa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PAIRING" {
		t.Fatalf("expected PAIRING, got %s", result.Status)
	}
	if result.Support != model.SupportComplete {
		t.Fatalf("expected complete support, got %s", result.Support)
	}
	if manager.connectCalls != 1 {
		t.Fatalf("expected manager Connect to be called once, got %d", manager.connectCalls)
	}
}

func TestLifecycleOrchestratorQR(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{ID: "sess-wa", Engine: "whatsmeow", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	orchestrator := newSessionLifecycle(&RuntimeResolver{repo: repo}, &stubLifecycleManager{qrCode: "qr-code"}, nil)

	result, err := orchestrator.QR(context.Background(), "sess-wa")
	if err != nil {
		t.Fatalf("unexpected QR error: %v", err)
	}
	if result.QRCode != "qr-code" {
		t.Fatalf("expected qr-code, got %s", result.QRCode)
	}
}

func TestLifecycleOrchestratorPair(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{ID: "sess-wa", Engine: "whatsmeow", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	manager := &stubLifecycleManager{pairCode: "123-456"}
	orchestrator := newSessionLifecycle(&RuntimeResolver{repo: repo}, manager, nil)

	result, err := orchestrator.Pair(context.Background(), "sess-wa", "5511999999999")
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

func TestLifecycleOrchestratorRestart(t *testing.T) {
	repo := &stubSessionLookup{
		session: &model.Session{ID: "sess-wa", Engine: "whatsmeow", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	reader := &stubSessionReader{resp: &dto.SessionResp{ID: "sess-wa", Engine: "whatsmeow"}}
	manager := &stubLifecycleManager{}
	orchestrator := newSessionLifecycle(&RuntimeResolver{repo: repo}, manager, reader)

	result, err := orchestrator.Restart(context.Background(), "sess-wa")
	if err != nil {
		t.Fatalf("unexpected restart error: %v", err)
	}
	if result.Session == nil || result.Session.ID != "sess-wa" {
		t.Fatal("expected restart to return refreshed session")
	}
	if manager.disconnectCalls != 1 {
		t.Fatalf("expected Disconnect to be called once, got %d", manager.disconnectCalls)
	}
	if manager.connectCalls != 1 {
		t.Fatalf("expected Connect to be called once, got %d", manager.connectCalls)
	}
}
