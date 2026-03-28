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
	sessionRepo := repository.NewSessionRepository(s.db)
	webhookRepo := repository.NewWebhookRepository(s.db)

	// Initialize Engine
	engine, err := service.NewEngine(s.Config, sessionRepo, s.nats)
	if err != nil {
		return err
	}

	// Initialize Services
	sessionSvc := service.NewSessionService(sessionRepo, engine)
	messageSvc := service.NewMessageService(engine)
	contactSvc := service.NewContactService(engine)
	groupSvc := service.NewGroupService(engine)
	webhookSvc := service.NewWebhookService(webhookRepo)

	// Initialize Handlers
	healthHandler := handler.NewHealthHandler(s.db != nil, s.nats != nil, s.minio != nil)
	sessionHandler := handler.NewSessionHandler(sessionSvc, engine)
	messageHandler := handler.NewMessageHandler(messageSvc)
	contactHandler := handler.NewContactHandler(contactSvc)
	groupHandler := handler.NewGroupHandler(groupSvc)
	webhookHandler := handler.NewWebhookHandler(webhookSvc)

	// Swagger UI (No Auth)
	s.App.Get("/swagger/*", swagger.HandlerDefault)

	// Health (No Auth)
	s.App.Get("/health", healthHandler.Check)

	// API Group with Auth
	api := s.App.Group("/", middleware.Auth(s.Config, sessionRepo))

	// 1. Session Management
	// Admin (List/Create):
	api.Post("/sessions", sessionHandler.Create)
	api.Get("/sessions", sessionHandler.List)
	// Sub-Group Middleware mapping
	reqSession := middleware.RequiredSession()

	// 1. Active Session Management
	sessGrp := api.Group("/session", reqSession)
	sessGrp.Get("/", sessionHandler.Get)
	sessGrp.Delete("/", sessionHandler.Delete)
	sessGrp.Post("/connect", sessionHandler.Connect)
	sessGrp.Post("/disconnect", sessionHandler.Disconnect)
	sessGrp.Get("/qr", sessionHandler.QR)

	// 2. Messaging
	msgGrp := api.Group("/messages", reqSession)
	msgGrp.Post("/text", messageHandler.SendText)
	msgGrp.Post("/image", messageHandler.SendImage)
	msgGrp.Post("/video", messageHandler.SendVideo)
	msgGrp.Post("/document", messageHandler.SendDocument)
	msgGrp.Post("/audio", messageHandler.SendAudio)

	// 3. Contacts
	contactGrp := api.Group("/contacts", reqSession)
	contactGrp.Get("/", contactHandler.List)
	contactGrp.Post("/check", contactHandler.Check)

	// 4. Groups
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

	// 5. Webhooks
	api.Post("/webhooks", webhookHandler.Create)
	api.Get("/webhooks", webhookHandler.List)
	api.Delete("/webhooks/:wid", webhookHandler.Delete)

	return nil
}
