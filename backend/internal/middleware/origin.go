package middleware

import (
	"fmt"

	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/common"
)

// RequireOrigin rejects requests whose Origin header is absent or not
// one of the allowed web origins. It runs before any cookie is read.
func RequireOrigin(allowed []string) echo.MiddlewareFunc {
	set := make(map[string]struct{}, len(allowed))
	for _, origin := range allowed {
		set[origin] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if _, ok := set[c.Request().Header.Get("Origin")]; !ok {
				return fmt.Errorf("checking request origin: %w", common.ErrForbiddenOrigin)
			}
			return next(c)
		}
	}
}
