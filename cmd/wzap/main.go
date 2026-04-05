// @title           wzap API
// @version         1.0
// @description     WhatsApp Multi-Session API powered by whatsmeow
// @contact.name    wzap Support
// @contact.url     https://github.com/felipyfgs/wzap
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
// @securityDefinitions.apikey Authorization
// @in header
// @name Authorization
// @description Session or admin token (without Bearer prefix)

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wzap/internal/broker"
	"wzap/internal/config"
	"wzap/internal/database"
	"wzap/internal/logger"
	"wzap/internal/server"
	"wzap/internal/storage"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, cfg.Environment)

	ctx := context.Background()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	if err := db.BootstrapBaseline(ctx); err != nil {
		logger.Warn().Err(err).Msg("Failed to bootstrap baseline migrations (may be fresh install)")
	}
	if err := db.Migrate(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	nats, err := broker.New(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to NATS")
	}
	defer nats.Close()

	minio, err := storage.New(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to MinIO")
	}

	srv := server.New(cfg, db, nats, minio)

	if err := srv.SetupRoutes(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to setup routes")
	}

	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}
}
