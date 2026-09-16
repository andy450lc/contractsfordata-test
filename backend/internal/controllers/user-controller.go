package controllers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/middleware"
	"github.com/pixels-two/sow/backend/internal/models/dto"
	"github.com/pixels-two/sow/backend/internal/services"
)

// UserController serves the authenticated user's own record.
type UserController struct {
	users *services.UserService
}

func NewUserController(users *services.UserService) *UserController {
	return &UserController{users: users}
}

// Me returns the caller's user record.
func (uc *UserController) Me(c *echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := middleware.UserID(ctx)
	if !ok {
		return fmt.Errorf("reading authenticated user: %w", common.ErrUnauthorized)
	}

	user, err := uc.users.Me(ctx, userID)
	if err != nil {
		return fmt.Errorf("loading current user: %w", err)
	}

	return c.JSON(http.StatusOK, dto.UserFrom(user))
}
