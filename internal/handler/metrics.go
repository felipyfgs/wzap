package handler

import (
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsHandler struct{}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// Serve godoc
// @Summary     Prometheus metrics endpoint
// @Description Exposes Prometheus metrics for monitoring wzap instance
// @Tags        Metrics
// @Produce     plain
// @Success     200 {string} string "Prometheus text format metrics"
// @Router      /metrics [get]
func (h *MetricsHandler) Serve(c *fiber.Ctx) error {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Accept", "text/plain")
	w := httptest.NewRecorder()

	handler := promhttp.Handler()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	c.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.Status(resp.StatusCode)

	_, _ = io.Copy(c.Response().BodyWriter(), resp.Body)

	return nil
}
