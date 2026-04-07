package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"

	"wzap/internal/dto"
	"wzap/internal/metrics"
	"wzap/internal/model"
	"wzap/internal/wa"
)

type sessionLifecycleManager interface {
	Connect(ctx context.Context, sessionID string) (*whatsmeow.Client, <-chan whatsmeow.QRChannelItem, error)
	Disconnect(sessionID string) error
	Logout(ctx context.Context, sessionID string) error
	Reconnect(ctx context.Context, sessionID string) error
	GetQRCode(ctx context.Context, sessionID string) (string, error)
	PairPhone(ctx context.Context, sessionID, phone string) (string, error)
}

type sessionReader interface {
	Get(ctx context.Context, id string) (*dto.SessionResp, error)
}

type SessionLifecycleOrchestrator struct {
	runtimeResolver *SessionRuntimeResolver
	manager         sessionLifecycleManager
	sessions        sessionReader
}

type SessionConnectResult struct {
	Status  string
	Support model.CapabilitySupport
}

type SessionDisconnectResult struct {
	Support model.CapabilitySupport
}

type SessionLogoutResult struct {
	Support model.CapabilitySupport
}

type SessionReconnectResult struct {
	Support model.CapabilitySupport
}

type SessionRestartResult struct {
	Session *dto.SessionResp
	Support model.CapabilitySupport
}

type SessionQRResult struct {
	QRCode  string
	Support model.CapabilitySupport
}

type SessionPairResult struct {
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

func NewSessionLifecycleOrchestrator(runtimeResolver *SessionRuntimeResolver, manager *wa.Manager, sessions *SessionService) *SessionLifecycleOrchestrator {
	return newSessionLifecycleOrchestrator(runtimeResolver, manager, sessions)
}

func newSessionLifecycleOrchestrator(runtimeResolver *SessionRuntimeResolver, manager sessionLifecycleManager, sessions sessionReader) *SessionLifecycleOrchestrator {
	return &SessionLifecycleOrchestrator{
		runtimeResolver: runtimeResolver,
		manager:         manager,
		sessions:        sessions,
	}
}

func (o *SessionLifecycleOrchestrator) Connect(ctx context.Context, sessionID string) (*SessionConnectResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionConnect)
	if err != nil {
		return nil, err
	}
	if support == model.CapabilitySupportPartial {
		return &SessionConnectResult{Status: "CONNECTED", Support: support}, nil
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

	return &SessionConnectResult{Status: status, Support: support}, nil
}

func (o *SessionLifecycleOrchestrator) Disconnect(ctx context.Context, sessionID string) (*SessionDisconnectResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionDisconnect)
	if err != nil {
		return nil, err
	}
	if support == model.CapabilitySupportPartial {
		return &SessionDisconnectResult{Support: support}, nil
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}
	if err := o.manager.Disconnect(runtime.Session().ID); err != nil {
		return nil, err
	}
	metrics.SessionsConnected.Dec()
	return &SessionDisconnectResult{Support: support}, nil
}

func (o *SessionLifecycleOrchestrator) Logout(ctx context.Context, sessionID string) (*SessionLogoutResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionLogout)
	if err != nil {
		return nil, err
	}
	if support == model.CapabilitySupportPartial {
		return &SessionLogoutResult{Support: support}, nil
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}
	if err := o.manager.Logout(runtime.WithContext(ctx), runtime.Session().ID); err != nil {
		return nil, err
	}
	return &SessionLogoutResult{Support: support}, nil
}

func (o *SessionLifecycleOrchestrator) Reconnect(ctx context.Context, sessionID string) (*SessionReconnectResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionReconnect)
	if err != nil {
		return nil, err
	}
	if support == model.CapabilitySupportPartial {
		return &SessionReconnectResult{Support: support}, nil
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}
	if err := o.manager.Reconnect(runtime.WithContext(ctx), runtime.Session().ID); err != nil {
		return nil, err
	}
	return &SessionReconnectResult{Support: support}, nil
}

func (o *SessionLifecycleOrchestrator) Restart(ctx context.Context, sessionID string) (*SessionRestartResult, error) {
	runtime, support, err := o.resolveCapability(ctx, sessionID, model.CapabilitySessionRestart)
	if err != nil {
		return nil, err
	}
	if o.sessions == nil {
		return nil, fmt.Errorf("session reader is nil")
	}
	if support == model.CapabilitySupportPartial {
		session, err := o.sessions.Get(ctx, runtime.Session().ID)
		if err != nil {
			return nil, normalizeLifecycleError(err)
		}
		return &SessionRestartResult{Session: session, Support: support}, nil
	}
	if o.manager == nil {
		return nil, fmt.Errorf("session lifecycle manager is nil")
	}

	if err := o.manager.Disconnect(runtime.Session().ID); err != nil {
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
	return &SessionRestartResult{Session: session, Support: support}, nil
}

func (o *SessionLifecycleOrchestrator) QR(ctx context.Context, sessionID string) (*SessionQRResult, error) {
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
	return &SessionQRResult{QRCode: qrCode, Support: support}, nil
}

func (o *SessionLifecycleOrchestrator) Pair(ctx context.Context, sessionID, phone string) (*SessionPairResult, error) {
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
	return &SessionPairResult{PairingCode: code, Support: support}, nil
}

func (o *SessionLifecycleOrchestrator) resolveCapability(ctx context.Context, sessionID string, capability model.EngineCapability) (*SessionRuntime, model.CapabilitySupport, error) {
	if o == nil || o.runtimeResolver == nil {
		return nil, model.CapabilitySupportUnavailable, fmt.Errorf("session lifecycle orchestrator is nil")
	}

	runtime, err := o.runtimeResolver.Resolve(ctx, sessionID)
	if err != nil {
		return nil, model.CapabilitySupportUnavailable, normalizeLifecycleError(err)
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
	if strings.Contains(err.Error(), "session not found") {
		return &LifecycleNotFoundError{Message: err.Error()}
	}
	return err
}
