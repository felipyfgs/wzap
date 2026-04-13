package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"wzap/internal/async"
	"wzap/internal/broker"
	"wzap/internal/config"
	"wzap/internal/database"
	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/middleware"
	"wzap/internal/storage"
)

type Server struct {
	App    *fiber.App
	Config *config.Config

	db     *database.DB
	nats   *broker.NATS
	minio  *storage.Minio
	ctx    context.Context
	cancel context.CancelFunc
	async  *async.Runtime
}

func New(cfg *config.Config, db *database.DB, n *broker.NATS, m *storage.Minio) *Server {
	app := fiber.New(fiber.Config{
		ServerHeader:          "wzap",
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(dto.ErrorResp("Error", err.Error()))
		},
	})

	// Middlewares
	app.Use(middleware.Recovery())
	app.Use(middleware.Logger())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))
	app.Use(middleware.RateLimit(300, time.Minute))

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		App:    app,
		Config: cfg,
		db:     db,
		nats:   n,
		minio:  m,
		ctx:    ctx,
		cancel: cancel,
		async:  async.NewRuntime(),
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.Config.ServerHost, s.Config.Port)
	logger.Info().Str("component", "server").Str("addr", addr).Msg("Starting API server")
	return s.App.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info().Str("component", "server").Msg("Shutting down API server")
	s.cancel()

	if s.async != nil {
		s.async.Shutdown(ctx)
	}

	// Fiber shutdown might block, we wrap it in a channel with context timeout
	done := make(chan error, 1)
	go func() {
		done <- s.App.Shutdown()
	}()

	select {
	case <-ctx.Done():
		logger.Warn().Str("component", "server").Msg("API server shutdown timed out")
		return ctx.Err()
	case err := <-done:
		logger.Info().Str("component", "server").Msg("API server stopped gracefully")
		return err
	}
}
