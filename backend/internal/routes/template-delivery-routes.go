package routes

import (
	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/controllers"
	appmw "github.com/pixels-two/sow/backend/internal/middleware"
	"github.com/pixels-two/sow/backend/internal/models/dto"
)

// AddTemplateDeliveryRoutes registers public Word delivery with origin and rate controls.
func AddTemplateDeliveryRoutes(e *echo.Echo, cfg config.AppConfig, controller *controllers.TemplateDeliveryController) {
	group := e.Group("/v1/template-deliveries",
		appmw.TemplateDeliveryRateLimiter(cfg.TemplateDeliveryRateLimitRPS, cfg.TemplateDeliveryRateLimitBurst),
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
	)
	group.POST("/download", controller.Download, appmw.ValidateRequest(new(dto.TemplateDownloadDto)))
	group.POST("/email", controller.Email, appmw.ValidateRequest(new(dto.TemplateEmailDto)))
}
