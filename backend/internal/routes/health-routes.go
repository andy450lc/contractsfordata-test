package routes

import (
	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/controllers"
)

// AddHealthRoutes registers the operational probes outside the /v1 prefix.
func AddHealthRoutes(e *echo.Echo, healthController *controllers.HealthController) {
	e.GET("/livez",
		healthController.Livez,
	)
	e.GET("/readyz",
		healthController.Readyz,
	)
}
