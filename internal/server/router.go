package server

import (
	"github.com/gofiber/swagger"

	"wzap/internal/handler"
	"wzap/internal/middleware"
	"wzap/internal/repo"
	"wzap/internal/service"
	"wzap/internal/wa"
)

func (s *Server) SetupRoutes() error {
	// Initialize Repositories
	sessionRepo := repo.NewSessionRepository(s.db)
	webhookRepo := repo.NewWebhookRepository(s.db)

	// Initialize Engine
	engine, err := wa.NewManager(s.Config, sessionRepo, s.nats)
	if err != nil {
		return err
	}

	// Initialize Services
	sessionSvc := service.NewSessionService(sessionRepo, engine)
	messageSvc := service.NewMessageService(engine)
	contactSvc := service.NewContactService(engine)
	groupSvc := service.NewGroupService(engine)
	webhookSvc := service.NewWebhookService(webhookRepo)
	labelSvc := service.NewLabelService(engine)
	newsletterSvc := service.NewNewsletterService(engine)
	communitySvc := service.NewCommunityService(engine)
	chatSvc := service.NewChatService(engine)

	// Initialize Handlers
	healthHandler := handler.NewHealthHandler(s.db != nil, s.nats != nil, s.minio != nil)
	sessionHandler := handler.NewSessionHandler(sessionSvc, engine)
	messageHandler := handler.NewMessageHandler(messageSvc)
	contactHandler := handler.NewContactHandler(contactSvc)
	groupHandler := handler.NewGroupHandler(groupSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)
	labelHandler := handler.NewLabelHandler(labelSvc)
	newsletterHandler := handler.NewNewsletterHandler(newsletterSvc)
	communityHandler := handler.NewCommunityHandler(communitySvc)
	chatHandler := handler.NewChatHandler(chatSvc)

	// Swagger UI (No Auth)
	s.App.Get("/swagger/*", swagger.HandlerDefault)

	// Health (No Auth)
	s.App.Get("/health", healthHandler.Check)

	// API Group with Auth (admin token or session token)
	grp := s.App.Group("/", middleware.Auth(s.Config, sessionRepo))

	// 1. Session Management
	grp.Post("/sessions", sessionHandler.Create) // Admin only
	grp.Get("/sessions", sessionHandler.List)    // Admin only

	// Session-scoped routes — :sessionName resolved by RequiredSession middleware
	reqSession := middleware.RequiredSession(sessionRepo)
	sess := grp.Group("/sessions/:sessionId", reqSession)

	// 2. Session lifecycle
	sess.Get("/", sessionHandler.Get)
	sess.Delete("/", sessionHandler.Delete)
	sess.Post("/connect", sessionHandler.Connect)
	sess.Post("/disconnect", sessionHandler.Disconnect)
	sess.Get("/qr", sessionHandler.QR)

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

	// 6. Chat
	sess.Post("/chat/archive", chatHandler.Archive)
	sess.Post("/chat/mute", chatHandler.Mute)
	sess.Post("/chat/pin", chatHandler.Pin)
	sess.Post("/chat/unpin", chatHandler.Unpin)

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

	// 9. Community
	sess.Post("/community/create", communityHandler.Create)
	sess.Post("/community/participant/add", communityHandler.AddParticipant)
	sess.Post("/community/participant/remove", communityHandler.RemoveParticipant)

	// 10. Webhooks
	sess.Post("/webhooks", webhookHandler.Create)
	sess.Get("/webhooks", webhookHandler.List)
	sess.Delete("/webhooks/:wid", webhookHandler.Delete)

	return nil
}
