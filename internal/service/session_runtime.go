package service

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"

	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/wa"
)

type sessionLookup interface {
	FindByID(ctx context.Context, id string) (*model.Session, error)
}

type runtimeCtxKey struct{}

type RuntimeResolver struct {
	repo   sessionLookup
	engine *wa.Manager
}

type SessionRuntime struct {
	session *model.Session
	engine  *wa.Manager
}

type MessageRuntime struct {
	*SessionRuntime
	support model.CapabilitySupport
}

type MediaRuntime struct {
	*SessionRuntime
	support model.CapabilitySupport
}

type StatusRuntime struct {
	*SessionRuntime
	support model.CapabilitySupport
}

type ProfileRuntime struct {
	*SessionRuntime
	support model.CapabilitySupport
}

func NewRuntimeResolver(repo *repo.SessionRepository, engine *wa.Manager) *RuntimeResolver {
	return &RuntimeResolver{repo: repo, engine: engine}
}

func (r *RuntimeResolver) Resolve(ctx context.Context, sessionID string) (*SessionRuntime, error) {
	if runtime, ok := sessionRuntimeFromContext(ctx, sessionID); ok {
		return runtime, nil
	}
	if r.repo == nil {
		return nil, fmt.Errorf("session repository is nil")
	}

	session, err := r.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	runtime := &SessionRuntime{
		session: session,
		engine:   r.engine,
	}

	return runtime, nil
}

func (r *RuntimeResolver) ResolveMessage(ctx context.Context, sessionID string, capability model.EngineCapability) (*MessageRuntime, error) {
	runtime, support, err := r.resolveCapability(ctx, sessionID, capability)
	if err != nil {
		return nil, err
	}
	return &MessageRuntime{SessionRuntime: runtime, support: support}, nil
}

func (r *RuntimeResolver) ResolveMedia(ctx context.Context, sessionID string, capability model.EngineCapability) (*MediaRuntime, error) {
	runtime, support, err := r.resolveCapability(ctx, sessionID, capability)
	if err != nil {
		return nil, err
	}
	return &MediaRuntime{SessionRuntime: runtime, support: support}, nil
}

func (r *RuntimeResolver) ResolveStatus(ctx context.Context, sessionID string) (*StatusRuntime, error) {
	runtime, support, err := r.resolveCapability(ctx, sessionID, model.CapabilitySessionStatus)
	if err != nil {
		return nil, err
	}
	return &StatusRuntime{SessionRuntime: runtime, support: support}, nil
}

func (r *RuntimeResolver) ResolveProfile(ctx context.Context, sessionID string) (*ProfileRuntime, error) {
	runtime, support, err := r.resolveCapability(ctx, sessionID, model.CapabilitySessionProfile)
	if err != nil {
		return nil, err
	}
	return &ProfileRuntime{SessionRuntime: runtime, support: support}, nil
}

func (r *RuntimeResolver) resolveCapability(ctx context.Context, sessionID string, capability model.EngineCapability) (*SessionRuntime, model.CapabilitySupport, error) {
	runtime, err := r.Resolve(ctx, sessionID)
	if err != nil {
		return nil, model.SupportUnavailable, err
	}

	support, err := runtime.RequireCapability(capability)
	if err != nil {
		return nil, support, err
	}

	return runtime, support, nil
}

func (r *SessionRuntime) Session() *model.Session {
	return r.session
}

func (r *SessionRuntime) Engine() string {
	if r.session == nil {
		return ""
	}
	return r.session.Engine
}

func (r *SessionRuntime) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, runtimeCtxKey{}, r)
}

func (r *SessionRuntime) Client() (*whatsmeow.Client, error) {
	if r.session == nil {
		return nil, fmt.Errorf("session runtime is nil")
	}
	if r.engine == nil {
		return nil, fmt.Errorf("whatsmeow manager unavailable")
	}
	return r.engine.GetClient(r.session.ID)
}

func (r *SessionRuntime) ConnectedClient() (*whatsmeow.Client, error) {
	client, err := r.Client()
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}
	return client, nil
}

func (r *SessionRuntime) RequireCapability(capability model.EngineCapability) (model.CapabilitySupport, error) {
	if r.session == nil {
		return model.SupportUnavailable, fmt.Errorf("session runtime is nil")
	}
	return requireCapability(r.session.Engine, capability)
}

func (r *MessageRuntime) Support() model.CapabilitySupport {
	return r.support
}

func (r *MediaRuntime) Support() model.CapabilitySupport {
	return r.support
}

func (r *StatusRuntime) Support() model.CapabilitySupport {
	return r.support
}

func (r *ProfileRuntime) Support() model.CapabilitySupport {
	return r.support
}

func sessionRuntimeFromContext(ctx context.Context, sessionID string) (*SessionRuntime, bool) {
	if ctx == nil {
		return nil, false
	}
	runtime, ok := ctx.Value(runtimeCtxKey{}).(*SessionRuntime)
	if !ok || runtime == nil || runtime.session == nil {
		return nil, false
	}
	if sessionID != "" && runtime.session.ID != sessionID {
		return nil, false
	}
	return runtime, true
}

type clientResolver func() (*whatsmeow.Client, error)

func runClientRuntime[T any](ctx context.Context, runtime *SessionRuntime, resolveClient clientResolver, whatsmeowFn func(context.Context, *model.Session, *whatsmeow.Client) (T, error)) (T, error) {
	var zero T
	if runtime == nil || runtime.session == nil {
		return zero, fmt.Errorf("session runtime is nil")
	}
	ctx = runtime.WithContext(ctx)
	if whatsmeowFn == nil {
		return zero, fmt.Errorf("whatsmeow runtime handler is nil")
	}
	client, err := resolveClient()
	if err != nil {
		return zero, err
	}
	return whatsmeowFn(ctx, runtime.session, client)
}

func runSessionRuntime[T any](ctx context.Context, runtime *SessionRuntime, whatsmeowFn func(context.Context, *model.Session, *whatsmeow.Client) (T, error)) (T, error) {
	return runClientRuntime(ctx, runtime, runtime.Client, whatsmeowFn)
}

func runConnectedRuntime[T any](ctx context.Context, runtime *SessionRuntime, whatsmeowFn func(context.Context, *model.Session, *whatsmeow.Client) (T, error)) (T, error) {
	return runClientRuntime(ctx, runtime, runtime.ConnectedClient, whatsmeowFn)
}

func runRuntimeErr(ctx context.Context, runtime *SessionRuntime, whatsmeowFn func(context.Context, *model.Session, *whatsmeow.Client) error) error {
	_, err := runSessionRuntime(ctx, runtime, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (struct{}, error) {
		return struct{}{}, whatsmeowFn(ctx, session, client)
	})
	return err
}
