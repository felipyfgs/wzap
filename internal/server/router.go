package server

import (
	ws "github.com/gofiber/contrib/websocket"
	"github.com/gofiber/swagger"

	_ "wzap/docs"
	"wzap/internal/handler"
	"wzap/internal/middleware"
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

	disp := webhook.New(webhookRepo, s.nats, s.Config.GlobalWebhookURL)
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

	// Initialize Services
	sessionSvc := service.NewSessionService(sessionRepo, webhookRepo, engine)
	messageSvc := service.NewMessageService(engine)
	contactSvc := service.NewContactService(engine)
	groupSvc := service.NewGroupService(engine)
	webhookSvc := service.NewWebhookService(webhookRepo)
	labelSvc := service.NewLabelService(engine)
	newsletterSvc := service.NewNewsletterService(engine)
	communitySvc := service.NewCommunityService(engine)
	chatSvc := service.NewChatService(engine)
	mediaSvc := service.NewMediaService(engine, s.minio)
	historySvc := service.NewHistoryService(messageRepo)

	engine.SetMediaAutoUpload(mediaSvc.AutoUploadMedia)
	engine.SetMessagePersist(historySvc.PersistMessage)

	// Initialize Handlers
	healthHandler := handler.NewHealthHandler(s.db, s.nats, s.minio)
	sessionHandler := handler.NewSessionHandler(sessionSvc, engine)
	messageHandler := handler.NewMessageHandler(messageSvc)
	contactHandler := handler.NewContactHandler(contactSvc)
	groupHandler := handler.NewGroupHandler(groupSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	labelHandler := handler.NewLabelHandler(labelSvc)
	newsletterHandler := handler.NewNewsletterHandler(newsletterSvc)
	communityHandler := handler.NewCommunityHandler(communitySvc)
	chatHandler := handler.NewChatHandler(chatSvc)
	mediaHandler := handler.NewMediaHandler(mediaSvc)
	historyHandler := handler.NewHistoryHandler(messageRepo)

	wsHandler := handler.NewWebSocketHandler(hub, s.Config)

	// Swagger UI (No Auth)
	s.App.Get("/swagger/*", swagger.HandlerDefault)

	// Health (No Auth)
	s.App.Get("/health", healthHandler.Check)

	// WebSocket (token via query param or Authorization header)
	s.App.Use("/ws", wsHandler.Upgrade())
	s.App.Get("/ws/:sessionId", ws.New(wsHandler.Handle()))
	s.App.Get("/ws", ws.New(wsHandler.Handle()))

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

	// 3.1. Media & History
	sess.Get("/media/:messageId", mediaHandler.GetMedia)
	sess.Get("/messages", historyHandler.ListMessages)

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

	return nil
}
