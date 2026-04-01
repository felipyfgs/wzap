package handler

import (
	"wzap/internal/broker"
	"wzap/internal/database"
	"wzap/internal/dto"
	"wzap/internal/storage"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	db    *database.DB
	nats  *broker.Nats
	minio *storage.Minio
}

func NewHealthHandler(db *database.DB, nats *broker.Nats, minio *storage.Minio) *HealthHandler {
	return &HealthHandler{db: db, nats: nats, minio: minio}
}

// Check godoc
// @Summary     Health check
// @Description Returns the health status of the API and its dependencies
// @Tags        Health
// @Produce     json
// @Success     200 {object} dto.APIResponse
// @Router      /health [get]
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	ctx := c.Context()

	dbOK := h.db != nil && h.db.Health(ctx) == nil
	natsOK := h.nats != nil && h.nats.Health() == nil
	minioOK := h.minio != nil && h.minio.Health(ctx) == nil

	overall := "UP"
	if !dbOK || !natsOK || !minioOK {
		overall = "DEGRADED"
	}

	status := map[string]interface{}{
		"status": overall,
		"services": map[string]bool{
			"database": dbOK,
			"nats":     natsOK,
			"minio":    minioOK,
		},
	}
	return c.JSON(dto.SuccessResp(status))
}
