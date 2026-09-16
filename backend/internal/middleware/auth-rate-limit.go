package middleware

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// AuthRateLimiter is the stricter per-IP limiter for the session
// endpoints. It runs on top of the global limiter.
func AuthRateLimiter(rps float64, burst int) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:  rps,
			Burst: burst,
		}),
	})
}
