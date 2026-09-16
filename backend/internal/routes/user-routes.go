package routes

import (
	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/controllers"
	appmw "github.com/pixels-two/sow/backend/internal/middleware"
)

// AddUserRoutes registers the authenticated user's own endpoints.
func AddUserRoutes(e *echo.Echo, auth *appmw.AuthMiddleware, userController *controllers.UserController) {
	e.GET("/v1/me",
		userController.Me,
		auth.RequireSession,
	)
}
