package server

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"wzap/internal/broker"
	"wzap/internal/config"
	"wzap/internal/middleware"
	"wzap/internal/storage"
)

type Server struct {
	App    *fiber.App
	Config *config.Config

	db    *pgxpool.Pool
	nats  *broker.Nats
	minio *storage.Minio
}

func New(cfg *config.Config, dbPool *pgxpool.Pool, n *broker.Nats, m *storage.Minio) *Server {
	app := fiber.New(fiber.Config{
		ServerHeader:          "wzap",
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
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

	return &Server{
		App:    app,
		Config: cfg,
		db:     dbPool,
		nats:   n,
		minio:  m,
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.Config.ServerHost, s.Config.Port)
	log.Info().Str("addr", addr).Msg("Starting API server")
	return s.App.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("Shutting down API server")

	// Fiber shutdown might block, we wrap it in a channel with context timeout
	done := make(chan error, 1)
	go func() {
		done <- s.App.Shutdown()
	}()

	select {
	case <-ctx.Done():
		log.Warn().Msg("API server shutdown timed out")
		return ctx.Err()
	case err := <-done:
		log.Info().Msg("API server stopped gracefully")
		return err
	}
}
