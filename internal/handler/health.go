package handler

import (
	"wzap/internal/dto"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	dbConn    bool
	natsConn  bool
	minioConn bool
}

func NewHealthHandler(db, nats, minio bool) *HealthHandler {
	return &HealthHandler{dbConn: db, natsConn: nats, minioConn: minio}
}

// Check godoc
// @Summary     Health check
// @Description Returns the health status of the API and its dependencies
// @Tags        Health
// @Produce     json
// @Success     200 {object} dto.APIResponse
// @Router      /health [get]
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	status := map[string]interface{}{
		"status": "UP",
		"services": map[string]bool{
			"database": h.dbConn,
			"nats":     h.natsConn,
			"minio":    h.minioConn,
		},
	}
	return c.JSON(dto.SuccessResp(status, "wzap is running"))
}
