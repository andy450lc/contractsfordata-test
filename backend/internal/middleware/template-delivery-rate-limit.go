package middleware

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// TemplateDeliveryRateLimiter limits public document generation by client address.
func TemplateDeliveryRateLimiter(rps float64, burst int) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:  rps,
			Burst: burst,
		}),
	})
}
