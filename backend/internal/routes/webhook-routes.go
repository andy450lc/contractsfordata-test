package routes

import (
	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/controllers"
)

// AddWebhookRoutes registers the inbound provider webhook. It is
// authenticated by its signature header, so no bearer middleware runs.
func AddWebhookRoutes(e *echo.Echo, webhookController *controllers.WebhookController) {
	e.POST("/v1/webhooks/workos",
		webhookController.ReceiveWorkOS,
	)
}
