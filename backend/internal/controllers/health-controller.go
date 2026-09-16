package controllers

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/models"
	"github.com/pixels-two/sow/backend/internal/services"
)

type HealthController struct {
	healthService *services.HealthService
}

func NewHealthController(healthService *services.HealthService) *HealthController {
	return &HealthController{healthService: healthService}
}

// Livez answers liveness: the process is up. No dependencies consulted.
func (hc *HealthController) Livez(c *echo.Context) error {
	return c.JSON(http.StatusOK, models.HealthResponse{Status: models.HealthStatusOK})
}

// Readyz answers readiness. Returns 503 when any dependency check fails.
func (hc *HealthController) Readyz(c *echo.Context) error {
	ctx := c.Request().Context()

	ready, checks := hc.healthService.Readiness(ctx)

	status := http.StatusOK
	body := models.HealthResponse{Status: models.HealthStatusOK, Checks: checks}
	if !ready {
		status = http.StatusServiceUnavailable
		body.Status = models.HealthStatusUnavailable
	}
	return c.JSON(status, body)
}
