package server

import (
	"context"
	"net/http"

	ws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/swagger"

	_ "wzap/docs"
	"wzap/internal/async"
	"wzap/internal/handler"
	"wzap/internal/integrations/chatwoot"
	"wzap/internal/logger"
	"wzap/internal/middleware"
	cloudWA "wzap/internal/provider/whatsapp"
	"wzap/internal/repo"
	"wzap/internal/service"
	"wzap/internal/wa"
	"wzap/internal/webhook"
	wsHub "wzap/internal/websocket"
)

func (s *Server) SetupRoutes() error {
	// Initialize Repositories
	sessionRepo := repo.NewSessionRepository(s.db.Pool)
	webhookRepo := repo.NewWebhookRepository(s.db.Pool)

	// Initialize Dispatcher
	hub := wsHub.NewHub()

	webhookPool := async.NewPool("webhook", 4, 100)
	disp := webhook.New(webhookRepo, s.nats, s.Config.GlobalWebhookURL, webhookPool)
	disp.SetWSBroadcaster(hub)
	go disp.StartConsumer(s.ctx)

	// Initialize Engine
	engine, err := wa.NewManager(s.ctx, s.Config, sessionRepo, s.nats, disp)
	if err != nil {
		return err
	}

	if err := engine.ReconnectAll(s.ctx); err != nil {
		return err
	}

	messageRepo := repo.NewMessageRepository(s.db.Pool)
	chatRepo := repo.NewChatRepository(s.db.Pool)
	statusRepo := repo.NewStatusRepository(s.db.Pool)

	// Initialize Cloud API Provider
	runtimeResolver := service.NewRuntimeResolver(sessionRepo, engine, nil)
	configReader := service.NewSessionConfigReader(runtimeResolver)
	cloudProvider := cloudWA.NewClient(&http.Client{Timeout: s.Config.HTTPTimeout}, configReader)
	runtimeResolver.SetProvider(cloudProvider)

	// Initialize Services
	sessionSvc := service.NewSessionService(sessionRepo, webhookRepo, engine, cloudProvider, runtimeResolver)
	lifecycleSvc := service.NewLifecycleOrchestrator(runtimeResolver, engine, sessionSvc)
	messageSvc := service.NewMessageService(engine, cloudProvider, sessionRepo, runtimeResolver)
	statusSvc := service.NewStatusService(runtimeResolver, statusRepo)
	contactSvc := service.NewContactService(engine)
	groupSvc := service.NewGroupService(engine)
	webhookSvc := service.NewWebhookService(webhookRepo)
	labelSvc := service.NewLabelService(engine)
	newsletterSvc := service.NewNewsletterService(engine)
	communitySvc := service.NewCommunityService(engine)
	chatSvc := service.NewChatService(engine)
	mediaPool := s.async.AddPool("media", 4, 50)
	mediaSvc := service.NewMediaService(engine, s.minio, cloudProvider, sessionRepo, mediaPool, runtimeResolver)
	historyPool := s.async.AddPool("history", 2, 100)
	historySvc := service.NewHistoryService(messageRepo, chatRepo, historyPool)
	historySvc.SetMediaDownloader(engine)
	if s.minio != nil {
		historySvc.SetMediaStorage(service.NewMinioMediaStorage(s.minio))
	}

	chatwootRepo := chatwoot.NewRepository(s.db.Pool)
	chatwootSvc := chatwoot.NewService(s.ctx, chatwootRepo, messageRepo, messageSvc)
	chatwootSvc.SetJIDResolver(engine)
	chatwootSvc.SetMediaDownloader(engine)
	chatwootSvc.SetSessionConnector(chatwoot.NewSessionConnector(engine))
	chatwootSvc.SetAvatarGetter(engine)
	chatwootSvc.SetNumberChecker(engine)
	chatwootSvc.SetContactNameGetter(engine)
	if s.minio != nil {
		chatwootSvc.SetMediaPresigner(mediaSvc)
	}
	chatwootSvc.SetServerURL(s.Config.ServerURL)
	chatwootSvc.SetCache(chatwoot.NewCache(s.ctx, s.Config.RedisURL))
	if s.nats != nil {
		chatwootSvc.SetJetStream(s.nats.JS)
		cwConsumer, cwErr := chatwoot.NewConsumer(s.nats.JS, chatwootSvc)
		if cwErr != nil {
			logger.Warn().Str("component", "server").Err(cwErr).Msg("Failed to create Chatwoot NATS consumer, falling back to sync mode")
		} else {
			if err := cwConsumer.Start(s.ctx); err != nil {
				logger.Warn().Str("component", "server").Err(err).Msg("Failed to start Chatwoot NATS consumer")
			}
		}
	}
	chatwootHandler := chatwoot.NewHandler(chatwootSvc, chatwootRepo)
	disp.AddListener(chatwootSvc)

	mediaSvc.SetMessageRepo(messageRepo)
	engine.SetMediaAutoUpload(mediaSvc.AutoUploadMedia)
	engine.SetMediaRetry(mediaSvc.RetryMediaUpload)
	engine.SetMessagePersist(historySvc.PersistMessage)
	engine.SetHistorySyncPersist(historySvc.PersistHistorySync)
	historySvc.SetMediaRetryRequester(engine)
	messageSvc.SetMessagePersist(historySvc.PersistMessage)
	engine.SetStatusReceived(statusSvc.PersistStatusReceived)
	engine.SetShouldIgnoreStatus(func(sessionID string) bool {
		sess, err := sessionRepo.FindByID(context.Background(), sessionID)
		if err != nil || sess == nil {
			return false
		}
		return sess.Settings.IgnoreStatus
	})
	engine.StartCacheGC()
	s.engine = engine
	s.dispatcher = disp

	// Initialize Handlers
	healthHandler := handler.NewHealthHandler(s.db, s.nats, s.minio)
	sessionHandler := handler.NewSessionHandler(sessionSvc, lifecycleSvc, chatwootRepo)
	messageHandler := handler.NewMessageHandler(messageSvc)
	statusHandler := handler.NewStatusHandler(statusSvc)
	contactHandler := handler.NewContactHandler(contactSvc)
	groupHandler := handler.NewGroupHandler(groupSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	labelHandler := handler.NewLabelHandler(labelSvc)
	newsletterHandler := handler.NewNewsletterHandler(newsletterSvc)
	communityHandler := handler.NewCommunityHandler(communitySvc)
	chatHandler := handler.NewChatHandler(chatSvc)
	mediaHandler := handler.NewMediaHandler(mediaSvc, messageRepo)
	historyHandler := handler.NewHistoryHandler(messageRepo)

	wsHandler := handler.NewWebSocketHandler(hub, s.Config)
	cloudWebhookHandler := handler.NewCloudWebhookHandler(sessionRepo, cloudProvider, disp)

	// Swagger UI (No Auth)
	s.App.Get("/swagger/*", swagger.HandlerDefault)

	// Health (No Auth)
	s.App.Get("/health", healthHandler.Check)

	// Metrics (No Auth - Prometheus)
	metricsHandler := handler.NewMetricsHandler()
	s.App.Get("/metrics", metricsHandler.Serve)

	// WebSocket (token via query param or Authorization header)
	s.App.Use("/ws", wsHandler.Upgrade())
	s.App.Get("/ws/:sessionId", ws.New(wsHandler.Handle()))
	s.App.Get("/ws", ws.New(wsHandler.Handle()))

	// Chatwoot Webhook (No Auth - validated via HMAC signature)
	s.App.Post("/chatwoot/webhook/:sessionId", chatwootHandler.IncomingWebhook)

	// Cloud API Webhooks (No Auth - validated via HMAC signature)
	s.App.Post("/webhooks/cloud/:sessionId", cloudWebhookHandler.Handle)
	s.App.Get("/webhooks/cloud/:sessionId", cloudWebhookHandler.Verify)

	// API Group with Auth (admin token or session token)
	grp := s.App.Group("/", middleware.Auth(s.Config, sessionRepo))

	// 1. Session Management
	grp.Post("/sessions", sessionHandler.Create) // Admin only
	grp.Get("/sessions", sessionHandler.List)    // Admin only

	// Session-scoped routes — :sessionId resolved by RequiredSession middleware
	reqSession := middleware.RequiredSession(sessionRepo)
	sess := grp.Group("/sessions/:sessionId", reqSession)

	// 2. Session lifecycle
	sess.Get("/", sessionHandler.Get)
	sess.Put("/", sessionHandler.Update)
	sess.Delete("/", sessionHandler.Delete)
	sess.Get("/status", sessionHandler.Status)
	sess.Post("/connect", sessionHandler.Connect)
	sess.Post("/disconnect", sessionHandler.Disconnect)
	sess.Post("/reconnect", sessionHandler.Reconnect)
	sess.Post("/restart", sessionHandler.Restart)
	sess.Post("/logout", sessionHandler.Logout)
	sess.Post("/pair", sessionHandler.Pair)
	sess.Get("/qr", sessionHandler.QR)
	sess.Get("/profile", sessionHandler.Profile)

	// 3. Messaging
	sess.Post("/messages/text", messageHandler.SendText)
	sess.Post("/messages/image", messageHandler.SendImage)
	sess.Post("/messages/video", messageHandler.SendVideo)
	sess.Post("/messages/document", messageHandler.SendDocument)
	sess.Post("/messages/audio", messageHandler.SendAudio)
	sess.Post("/messages/contact", messageHandler.SendContact)
	sess.Post("/messages/location", messageHandler.SendLocation)
	sess.Post("/messages/poll", messageHandler.SendPoll)
	sess.Post("/messages/sticker", messageHandler.SendSticker)
	sess.Post("/messages/link", messageHandler.SendLink)
	sess.Post("/messages/edit", messageHandler.EditMessage)
	sess.Post("/messages/delete", messageHandler.DeleteMessage)
	sess.Post("/messages/reaction", messageHandler.ReactMessage)
	sess.Post("/messages/read", messageHandler.MarkRead)
	sess.Post("/messages/presence", messageHandler.SetPresence)
	sess.Post("/messages/button", messageHandler.SendButton)
	sess.Post("/messages/list", messageHandler.SendList)
	sess.Post("/messages/forward", messageHandler.ForwardMessage)

	// 3.1. Media & History
	sess.Get("/media", historyHandler.ListMedia)
	sess.Get("/media/:messageId", mediaHandler.GetMedia)
	sess.Get("/messages", historyHandler.ListMessages)

	// 3.2. Status (WhatsApp Stories)
	sess.Post("/status/text", statusHandler.SendText)
	sess.Post("/status/image", statusHandler.SendImage)
	sess.Post("/status/video", statusHandler.SendVideo)
	sess.Get("/status", statusHandler.ListStatus)
	sess.Get("/status/:senderJid", statusHandler.ListContactStatus)

	// 4. Contacts
	sess.Get("/contacts", contactHandler.List)
	sess.Post("/contacts/check", contactHandler.Check)
	sess.Post("/contacts/avatar", contactHandler.GetAvatar)
	sess.Post("/contacts/block", contactHandler.Block)
	sess.Post("/contacts/unblock", contactHandler.Unblock)
	sess.Get("/contacts/blocklist", contactHandler.GetBlocklist)
	sess.Post("/contacts/info", contactHandler.GetUserInfo)
	sess.Get("/contacts/privacy", contactHandler.GetPrivacySettings)
	sess.Post("/contacts/profile-picture", contactHandler.SetProfilePicture)
	sess.Post("/contacts/presence", contactHandler.SubscribePresence)
	sess.Post("/contacts/privacy", contactHandler.SetPrivacy)
	sess.Post("/contacts/status", contactHandler.SetStatusMessage)
	sess.Post("/profile/name", contactHandler.UpdateProfileName)

	// 5. Groups
	sess.Get("/groups", groupHandler.List)
	sess.Post("/groups/create", groupHandler.Create)
	sess.Post("/groups/info", groupHandler.Info)
	sess.Post("/groups/invite-info", groupHandler.GetInfoFromLink)
	sess.Post("/groups/join", groupHandler.JoinWithLink)
	sess.Post("/groups/invite-link", groupHandler.GetInviteLink)
	sess.Post("/groups/leave", groupHandler.Leave)
	sess.Post("/groups/participants", groupHandler.UpdateParticipants)
	sess.Post("/groups/requests", groupHandler.GetRequests)
	sess.Post("/groups/requests/action", groupHandler.UpdateRequests)
	sess.Post("/groups/name", groupHandler.UpdateName)
	sess.Post("/groups/description", groupHandler.UpdateDescription)
	sess.Post("/groups/photo", groupHandler.UpdatePhoto)
	sess.Post("/groups/announce", groupHandler.SetAnnounce)
	sess.Post("/groups/locked", groupHandler.SetLocked)
	sess.Post("/groups/join-approval", groupHandler.SetJoinApproval)
	sess.Post("/groups/photo/remove", groupHandler.RemovePhoto)
	sess.Post("/groups/ephemeral", groupHandler.SetEphemeral)

	// 6. Chat
	sess.Post("/chat/archive", chatHandler.Archive)
	sess.Post("/chat/mute", chatHandler.Mute)
	sess.Post("/chat/pin", chatHandler.Pin)
	sess.Post("/chat/unpin", chatHandler.Unpin)
	sess.Post("/chat/unarchive", chatHandler.Unarchive)
	sess.Post("/chat/unmute", chatHandler.Unmute)
	sess.Post("/chat/delete", chatHandler.DeleteChat)
	sess.Post("/chat/read", chatHandler.MarkRead)
	sess.Post("/chat/unread", chatHandler.MarkUnread)

	// 7. Labels
	sess.Post("/label/chat", labelHandler.AddToChat)
	sess.Post("/label/edit", labelHandler.EditLabel)
	sess.Post("/label/message", labelHandler.AddToMessage)
	sess.Post("/unlabel/chat", labelHandler.RemoveFromChat)
	sess.Post("/unlabel/message", labelHandler.RemoveFromMessage)

	// 8. Newsletter
	sess.Post("/newsletter/create", newsletterHandler.Create)
	sess.Post("/newsletter/info", newsletterHandler.Info)
	sess.Post("/newsletter/invite", newsletterHandler.Invite)
	sess.Get("/newsletter/list", newsletterHandler.List)
	sess.Post("/newsletter/messages", newsletterHandler.Messages)
	sess.Post("/newsletter/subscribe", newsletterHandler.Subscribe)
	sess.Post("/newsletter/unsubscribe", newsletterHandler.Unsubscribe)
	sess.Post("/newsletter/mute", newsletterHandler.Mute)
	sess.Post("/newsletter/react", newsletterHandler.React)
	sess.Post("/newsletter/viewed", newsletterHandler.MarkViewed)

	// 9. Community
	sess.Post("/community/create", communityHandler.Create)
	sess.Post("/community/participant/add", communityHandler.AddParticipant)
	sess.Post("/community/participant/remove", communityHandler.RemoveParticipant)

	// 10. Webhooks
	sess.Post("/webhooks", webhookHandler.Create)
	sess.Get("/webhooks", webhookHandler.List)
	sess.Put("/webhooks/:wid", webhookHandler.Update)
	sess.Delete("/webhooks/:wid", webhookHandler.Delete)

	// 12. Chatwoot Integration
	sess.Put("/integrations/chatwoot", chatwootHandler.Configure)
	sess.Get("/integrations/chatwoot", chatwootHandler.GetConfig)
	sess.Delete("/integrations/chatwoot", chatwootHandler.DeleteConfig)
	sess.Post("/integrations/chatwoot/import", chatwootHandler.ImportHistory)

	return nil
}
