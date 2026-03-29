package server

import (
	"github.com/gofiber/swagger"

	"wzap/internal/handler"
	"wzap/internal/middleware"
	"wzap/internal/repository"
	"wzap/internal/service"
)

func (s *Server) SetupRoutes() error {
	// Initialize Repositories
	userRepo := repository.NewUserRepository(s.db)
	sessionRepo := repository.NewSessionRepository(s.db)
	webhookRepo := repository.NewWebhookRepository(s.db)

	// Initialize Engine
	engine, err := service.NewEngine(s.Config, sessionRepo, s.nats)
	if err != nil {
		return err
	}

	// Initialize Services
	userSvc := service.NewUserService(userRepo, sessionRepo, engine)
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
	userHandler := handler.NewUserHandler(userSvc)
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

	// API Group with Auth
	api := s.App.Group("/", middleware.Auth(s.Config, userRepo))

	// 1. User Management (Admin)
	api.Post("/users", userHandler.Create)
	api.Get("/users", userHandler.List)
	api.Get("/users/:id", userHandler.Get)
	api.Delete("/users/:id", userHandler.Delete)

	// 2. Session Management (Admin)
	api.Get("/sessions", sessionHandler.List)

	// RequiredSession middleware resolves user -> session
	reqSession := middleware.RequiredSession(sessionRepo)

	// 3. Active Session Management
	sessGrp := api.Group("/session", reqSession)
	sessGrp.Get("/", sessionHandler.Get)
	sessGrp.Delete("/", sessionHandler.Delete)
	sessGrp.Post("/connect", sessionHandler.Connect)
	sessGrp.Post("/disconnect", sessionHandler.Disconnect)
	sessGrp.Get("/qr", sessionHandler.QR)

	// 4. Messaging
	msgGrp := api.Group("/messages", reqSession)
	msgGrp.Post("/text", messageHandler.SendText)
	msgGrp.Post("/image", messageHandler.SendImage)
	msgGrp.Post("/video", messageHandler.SendVideo)
	msgGrp.Post("/document", messageHandler.SendDocument)
	msgGrp.Post("/audio", messageHandler.SendAudio)
	msgGrp.Post("/contact", messageHandler.SendContact)
	msgGrp.Post("/location", messageHandler.SendLocation)
	msgGrp.Post("/poll", messageHandler.SendPoll)
	msgGrp.Post("/sticker", messageHandler.SendSticker)
	msgGrp.Post("/link", messageHandler.SendLink)
	msgGrp.Post("/edit", messageHandler.EditMessage)
	msgGrp.Post("/delete", messageHandler.DeleteMessage)
	msgGrp.Post("/reaction", messageHandler.ReactMessage)
	msgGrp.Post("/read", messageHandler.MarkRead)
	msgGrp.Post("/presence", messageHandler.SetPresence)

	// 5. Contacts
	contactGrp := api.Group("/contacts", reqSession)
	contactGrp.Get("/", contactHandler.List)
	contactGrp.Post("/check", contactHandler.Check)
	contactGrp.Post("/avatar", contactHandler.GetAvatar)
	contactGrp.Post("/block", contactHandler.Block)
	contactGrp.Post("/unblock", contactHandler.Unblock)
	contactGrp.Get("/blocklist", contactHandler.GetBlocklist)
	contactGrp.Post("/info", contactHandler.GetUserInfo)
	contactGrp.Get("/privacy", contactHandler.GetPrivacySettings)
	contactGrp.Post("/profile-picture", contactHandler.SetProfilePicture)

	// 6. Groups
	groupGrp := api.Group("/groups", reqSession)
	groupGrp.Get("/", groupHandler.List)
	groupGrp.Post("/create", groupHandler.Create)
	groupGrp.Post("/info", groupHandler.Info)
	groupGrp.Post("/invite-info", groupHandler.GetInfoFromLink)
	groupGrp.Post("/join", groupHandler.JoinWithLink)
	groupGrp.Post("/invite-link", groupHandler.GetInviteLink)
	groupGrp.Post("/leave", groupHandler.Leave)
	groupGrp.Post("/participants", groupHandler.UpdateParticipants)
	groupGrp.Post("/requests", groupHandler.GetRequests)
	groupGrp.Post("/requests/action", groupHandler.UpdateRequests)
	groupGrp.Post("/name", groupHandler.UpdateName)
	groupGrp.Post("/description", groupHandler.UpdateDescription)
	groupGrp.Post("/photo", groupHandler.UpdatePhoto)
	groupGrp.Post("/announce", groupHandler.SetAnnounce)
	groupGrp.Post("/locked", groupHandler.SetLocked)
	groupGrp.Post("/join-approval", groupHandler.SetJoinApproval)

	// 7. Chat
	chatGrp := api.Group("/chat", reqSession)
	chatGrp.Post("/archive", chatHandler.Archive)
	chatGrp.Post("/mute", chatHandler.Mute)
	chatGrp.Post("/pin", chatHandler.Pin)
	chatGrp.Post("/unpin", chatHandler.Unpin)

	// 8. Label
	labelGrp := api.Group("/label", reqSession)
	labelGrp.Post("/chat", labelHandler.AddToChat)
	labelGrp.Post("/edit", labelHandler.EditLabel)
	labelGrp.Post("/message", labelHandler.AddToMessage)
	unlabelGrp := api.Group("/unlabel", reqSession)
	unlabelGrp.Post("/chat", labelHandler.RemoveFromChat)
	unlabelGrp.Post("/message", labelHandler.RemoveFromMessage)

	// 9. Newsletter
	newsletterGrp := api.Group("/newsletter", reqSession)
	newsletterGrp.Post("/create", newsletterHandler.Create)
	newsletterGrp.Post("/info", newsletterHandler.Info)
	newsletterGrp.Post("/invite", newsletterHandler.Invite)
	newsletterGrp.Get("/list", newsletterHandler.List)
	newsletterGrp.Post("/messages", newsletterHandler.Messages)
	newsletterGrp.Post("/subscribe", newsletterHandler.Subscribe)

	// 10. Community
	communityGrp := api.Group("/community", reqSession)
	communityGrp.Post("/create", communityHandler.Create)
	communityGrp.Post("/participant/add", communityHandler.AddParticipant)
	communityGrp.Post("/participant/remove", communityHandler.RemoveParticipant)

	// 11. Webhooks
	api.Post("/webhooks", webhookHandler.Create)
	api.Get("/webhooks", webhookHandler.List)
	api.Delete("/webhooks/:wid", webhookHandler.Delete)

	return nil
}
