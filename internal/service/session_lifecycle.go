package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"

	"wzap/internal/dto"
	"wzap/internal/metrics"
	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/wa"
)

type lifecycleManager interface {
	Connect(ctx context.Context, sessionID string) (*whatsmeow.Client, <-chan whatsmeow.QRChannelItem, error)
	Disconnect(ctx context.Context, sessionID string) error
	Logout(ctx context.Context, sessionID string) error
	Reconnect(ctx context.Context, sessionID string) error
	GetQRCode(ctx context.Context, sessionID string) (string, error)
	PairPhone(ctx context.Context, sessionID, phone string) (string, error)
}

type sessionReader interface {
	Get(ctx context.Context, id string) (*dto.SessionResp, error)
}

type LifecycleOrchestrator struct {
	runtimeResolver *RuntimeResolver
	manager         lifecycleManager
	sessions        sessionReader
}

type ConnectResult struct {
	Status  string
	Support model.CapabilitySupport
}

type DisconnectResult struct {
	Support model.CapabilitySupport
}

type LogoutResult struct {
	Support model.CapabilitySupport
}

type ReconnectResult struct {
	Support model.CapabilitySupport
}

type RestartResult struct {
	Session *dto.SessionResp
	Support model.CapabilitySupport
}

type QRResult struct {
	QRCode  string
	Support model.CapabilitySupport
}

type PairResult struct {
	PairingCode string
	Support     model.CapabilitySupport
}

type LifecycleConflictError struct {
	Message string
	Cause   error
}

func (e *LifecycleConflictError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *LifecycleConflictError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type LifecycleNotFoundError struct {
	Message string
}

func (e *LifecycleNotFoundError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NewLifecycleOrchestrator(runtimeResolver *RuntimeResolver, manager *wa.Manager, sessions *SessionService) *LifecycleOrchestrator {
	return newLifecycleOrchestrator(runtimeResolver, manager, sessions)
}

func newLifecycleOrchestrator(runtimeResolver *RuntimeResolver, manager lifecycleManager, sessions sessionReader) *LifecycleOrchestrator {
	return &LifecycleOrchestrator{
		runtimeResolver: runtimeResolver,
		manager:         manager,
		sessions:        sessions,
	}
}

func (o *LifecycleOrchestrator) Connect(ctx context.Context, sessionID string) (*ConnectResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionConnect)
	if err != nil {
		return nil, err
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}

	client, qrChan, err := o.manager.Connect(runtime.WithContext(ctx), runtime.Session().ID)
	if err != nil {
		if errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			return nil, &LifecycleConflictError{
				Message: "A QR code connection is already pending for this session",
				Cause:   err,
			}
		}
		return nil, err
	}

	status := "CONNECTED"
	if qrChan != nil {
		status = "PAIRING"
	} else if client != nil && !client.IsConnected() {
		status = "CONNECTING"
	}
	if status == "CONNECTED" {
		metrics.SessionsConnected.Inc()
	}

	return &ConnectResult{Status: status, Support: support}, nil
}

func (o *LifecycleOrchestrator) Disconnect(ctx context.Context, sessionID string) (*DisconnectResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionDisconnect)
	if err != nil {
		return nil, err
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}
	if err := o.manager.Disconnect(ctx, runtime.Session().ID); err != nil {
		return nil, err
	}
	metrics.SessionsConnected.Dec()
	return &DisconnectResult{Support: support}, nil
}

func (o *LifecycleOrchestrator) Logout(ctx context.Context, sessionID string) (*LogoutResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionLogout)
	if err != nil {
		return nil, err
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}
	if err := o.manager.Logout(runtime.WithContext(ctx), runtime.Session().ID); err != nil {
		return nil, err
	}
	return &LogoutResult{Support: support}, nil
}

func (o *LifecycleOrchestrator) Reconnect(ctx context.Context, sessionID string) (*ReconnectResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionReconnect)
	if err != nil {
		return nil, err
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}
	if err := o.manager.Reconnect(runtime.WithContext(ctx), runtime.Session().ID); err != nil {
		return nil, err
	}
	return &ReconnectResult{Support: support}, nil
}

func (o *LifecycleOrchestrator) Restart(ctx context.Context, sessionID string) (*RestartResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionRestart)
	if err != nil {
		return nil, err
	}
	if o.sessions == nil {
		return nil, fmt.Errorf("session reader is nil")
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}

	if err := o.manager.Disconnect(ctx, runtime.Session().ID); err != nil {
		return nil, err
	}
	time.Sleep(1 * time.Second)
	if _, _, err := o.manager.Connect(runtime.WithContext(ctx), runtime.Session().ID); err != nil {
		return nil, err
	}

	session, err := o.sessions.Get(ctx, runtime.Session().ID)
	if err != nil {
		return nil, normalizeLifecycleError(err)
	}
	return &RestartResult{Session: session, Support: support}, nil
}

func (o *LifecycleOrchestrator) QR(ctx context.Context, sessionID string) (*QRResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionQR)
	if err != nil {
		return nil, err
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}

	qrCode, err := o.manager.GetQRCode(runtime.WithContext(ctx), runtime.Session().ID)
	if err != nil {
		return nil, normalizeLifecycleError(err)
	}
	if qrCode == "" {
		return nil, &LifecycleNotFoundError{Message: "No QR code available. Call connect first, then poll this endpoint."}
	}
	return &QRResult{QRCode: qrCode, Support: support}, nil
}

func (o *LifecycleOrchestrator) Pair(ctx context.Context, sessionID, phone string) (*PairResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionPair)
	if err != nil {
		return nil, err
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}

	code, err := o.manager.PairPhone(runtime.WithContext(ctx), runtime.Session().ID, phone)
	if err != nil {
		return nil, normalizeLifecycleError(err)
	}
	return &PairResult{PairingCode: code, Support: support}, nil
}

func (o *LifecycleOrchestrator) resolveCapability(ctx context.Context, sessionID string, capability model.EngineCapability) (*SessionRuntime, model.CapabilitySupport, error) {
	if o == nil || o.runtimeResolver == nil {
		return nil, model.SupportUnavailable, fmt.Errorf("session lifecycle orchestrator is nil")
	}

	runtime, err := o.runtimeResolver.Resolve(ctx, sessionID)
	if err != nil {
		return nil, model.SupportUnavailable, normalizeLifecycleError(err)
	}
	support, err := runtime.RequireCapability(capability)
	if err != nil {
		return nil, support, err
	}
	return runtime, support, nil
}

func normalizeLifecycleError(err error) error {
	if err == nil {
		return nil
	}
	var capabilityErr *CapabilityError
	if errors.As(err, &capabilityErr) {
		return err
	}
	var conflictErr *LifecycleConflictError
	if errors.As(err, &conflictErr) {
		return err
	}
	var notFoundErr *LifecycleNotFoundError
	if errors.As(err, &notFoundErr) {
		return err
	}
	if errors.Is(err, repo.ErrSessionNotFound) {
		return &LifecycleNotFoundError{Message: err.Error()}
	}
	return err
}
