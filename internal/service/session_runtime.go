package service

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"

	"wzap/internal/model"
	cloudWA "wzap/internal/provider/whatsapp"
	"wzap/internal/repo"
	"wzap/internal/wa"
)

type sessionLookup interface {
	FindByID(ctx context.Context, id string) (*model.Session, error)
}

type runtimeCtxKey struct{}

type RuntimeResolver struct {
	repo     sessionLookup
	engine   *wa.Manager
	provider *cloudWA.Client
}

type SessionRuntime struct {
	session     *model.Session
	engine      *wa.Manager
	provider    *cloudWA.Client
	cloudConfig *cloudWA.Config
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

func NewRuntimeResolver(repo *repo.SessionRepository, engine *wa.Manager, provider *cloudWA.Client) *RuntimeResolver {
	return newRuntimeResolver(repo, engine, provider)
}

func newRuntimeResolver(repo sessionLookup, engine *wa.Manager, provider *cloudWA.Client) *RuntimeResolver {
	return &RuntimeResolver{repo: repo, engine: engine, provider: provider}
}

func (r *RuntimeResolver) SetProvider(provider *cloudWA.Client) {
	if r == nil {
		return
	}
	r.provider = provider
}

func (r *RuntimeResolver) Resolve(ctx context.Context, sessionID string) (*SessionRuntime, error) {
	if runtime, ok := sessionRuntimeFromContext(ctx, sessionID); ok {
		return runtime, nil
	}
	if r == nil {
		return nil, fmt.Errorf("session runtime resolver is nil")
	}
	if r.repo == nil {
		return nil, fmt.Errorf("session repository is nil")
	}

	session, err := r.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	runtime := &SessionRuntime{
		session:  session,
		engine:   r.engine,
		provider: r.provider,
	}
	if session.Engine == "cloud_api" {
		runtime.cloudConfig = buildCloudConfig(session)
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
	if r == nil {
		return nil
	}
	return r.session
}

func (r *SessionRuntime) Engine() string {
	if r == nil || r.session == nil {
		return ""
	}
	return r.session.Engine
}

func (r *SessionRuntime) IsCloudAPI() bool {
	return r.Engine() == "cloud_api"
}

func (r *SessionRuntime) Provider() *cloudWA.Client {
	if r == nil {
		return nil
	}
	return r.provider
}

func (r *SessionRuntime) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, runtimeCtxKey{}, r)
}

func (r *SessionRuntime) CloudConfig() (*cloudWA.Config, error) {
	if r == nil || r.session == nil {
		return nil, fmt.Errorf("session runtime is nil")
	}
	if !r.IsCloudAPI() {
		return nil, fmt.Errorf("session %s engine is %s, not cloud_api", r.session.ID, r.session.Engine)
	}
	if r.cloudConfig == nil {
		return nil, fmt.Errorf("cloud config unavailable for session %s", r.session.ID)
	}
	return r.cloudConfig, nil
}

func (r *SessionRuntime) Client() (*whatsmeow.Client, error) {
	if r == nil || r.session == nil {
		return nil, fmt.Errorf("session runtime is nil")
	}
	if r.IsCloudAPI() {
		return nil, fmt.Errorf("session %s engine is %s, not whatsmeow", r.session.ID, r.session.Engine)
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
	if r == nil || r.session == nil {
		return model.SupportUnavailable, fmt.Errorf("session runtime is nil")
	}
	return requireCapability(r.session.Engine, capability)
}

func (r *MessageRuntime) Support() model.CapabilitySupport {
	if r == nil {
		return model.SupportUnavailable
	}
	return r.support
}

func (r *MediaRuntime) Support() model.CapabilitySupport {
	if r == nil {
		return model.SupportUnavailable
	}
	return r.support
}

func (r *StatusRuntime) Support() model.CapabilitySupport {
	if r == nil {
		return model.SupportUnavailable
	}
	return r.support
}

func (r *ProfileRuntime) Support() model.CapabilitySupport {
	if r == nil {
		return model.SupportUnavailable
	}
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

func buildCloudConfig(session *model.Session) *cloudWA.Config {
	if session == nil {
		return nil
	}
	cfg := &cloudWA.Config{
		AccessToken:        session.AccessToken,
		PhoneNumberID:      session.PhoneNumberID,
		BusinessAccountID:  session.BusinessAccountID,
		AppSecret:          session.AppSecret,
		WebhookVerifyToken: session.WebhookVerifyToken,
	}
	cfg.ApplyDefaults()
	return cfg
}

func runSessionRuntime[T any](ctx context.Context, runtime *SessionRuntime, cloud func(context.Context, *model.Session, *cloudWA.Client) (T, error), whatsmeowFn func(context.Context, *model.Session, *whatsmeow.Client) (T, error)) (T, error) {
	var zero T
	if runtime == nil || runtime.session == nil {
		return zero, fmt.Errorf("session runtime is nil")
	}

	ctx = runtime.WithContext(ctx)
	if runtime.IsCloudAPI() {
		if cloud == nil {
			return zero, fmt.Errorf("cloud runtime handler is nil")
		}
		if runtime.provider == nil {
			return zero, fmt.Errorf("cloud provider unavailable")
		}
		return cloud(ctx, runtime.session, runtime.provider)
	}
	if whatsmeowFn == nil {
		return zero, fmt.Errorf("whatsmeow runtime handler is nil")
	}
	client, err := runtime.Client()
	if err != nil {
		return zero, err
	}
	return whatsmeowFn(ctx, runtime.session, client)
}

func runConnectedRuntime[T any](ctx context.Context, runtime *SessionRuntime, cloud func(context.Context, *model.Session, *cloudWA.Client) (T, error), whatsmeowFn func(context.Context, *model.Session, *whatsmeow.Client) (T, error)) (T, error) {
	var zero T
	if runtime == nil || runtime.session == nil {
		return zero, fmt.Errorf("session runtime is nil")
	}

	ctx = runtime.WithContext(ctx)
	if runtime.IsCloudAPI() {
		if cloud == nil {
			return zero, fmt.Errorf("cloud runtime handler is nil")
		}
		if runtime.provider == nil {
			return zero, fmt.Errorf("cloud provider unavailable")
		}
		return cloud(ctx, runtime.session, runtime.provider)
	}
	if whatsmeowFn == nil {
		return zero, fmt.Errorf("whatsmeow runtime handler is nil")
	}
	client, err := runtime.ConnectedClient()
	if err != nil {
		return zero, err
	}
	return whatsmeowFn(ctx, runtime.session, client)
}

func runRuntimeErr(ctx context.Context, runtime *SessionRuntime, cloud func(context.Context, *model.Session, *cloudWA.Client) error, whatsmeowFn func(context.Context, *model.Session, *whatsmeow.Client) error) error {
	_, err := runSessionRuntime(ctx, runtime, func(ctx context.Context, session *model.Session, provider *cloudWA.Client) (struct{}, error) {
		if cloud == nil {
			return struct{}{}, fmt.Errorf("cloud runtime handler is nil")
		}
		return struct{}{}, cloud(ctx, session, provider)
	}, func(ctx context.Context, session *model.Session, client *whatsmeow.Client) (struct{}, error) {
		if whatsmeowFn == nil {
			return struct{}{}, fmt.Errorf("whatsmeow runtime handler is nil")
		}
		return struct{}{}, whatsmeowFn(ctx, session, client)
	})
	return err
}
