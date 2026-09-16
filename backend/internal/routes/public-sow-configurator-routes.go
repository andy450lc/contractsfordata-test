package routes

import (
	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/controllers"
	appmw "github.com/pixels-two/sow/backend/internal/middleware"
	"github.com/pixels-two/sow/backend/internal/models/dto"
)

// AddPublicSOWConfiguratorRoutes registers the public configurator with origin and rate controls.
func AddPublicSOWConfiguratorRoutes(e *echo.Echo, cfg config.AppConfig, controller *controllers.PublicSOWConfiguratorController) {
	group := e.Group("/v1/public/sow-configurator",
		appmw.TemplateDeliveryRateLimiter(cfg.TemplateDeliveryRateLimitRPS, cfg.TemplateDeliveryRateLimitBurst),
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
	)
	group.POST("/download", controller.Download, appmw.ValidateRequest(new(dto.PublicSOWDownloadDto)))
	group.POST("/email", controller.Email, appmw.ValidateRequest(new(dto.PublicSOWEmailDto)))
}
