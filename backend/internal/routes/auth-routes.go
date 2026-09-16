package routes

import (
	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/controllers"
	appmw "github.com/pixels-two/sow/backend/internal/middleware"
	"github.com/pixels-two/sow/backend/internal/models/dto"
)

// AddAuthRoutes registers the session endpoints under /v1/auth with the
// stricter limiter. The cookie endpoints require an allowed Origin.
func AddAuthRoutes(e *echo.Echo, cfg config.AppConfig, authController *controllers.AuthController) {
	group := e.Group("/v1/auth",
		appmw.AuthRateLimiter(cfg.AuthRateLimitRPS, cfg.AuthRateLimitBurst),
	)

	group.GET("/login",
		authController.Login,
		appmw.ValidateRequest(new(dto.LoginQueryDto)),
	)
	group.GET("/callback",
		authController.Callback,
		appmw.ValidateRequest(new(dto.CallbackQueryDto)),
	)
	group.POST("/sign-in",
		authController.SignIn,
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
		appmw.ValidateRequest(new(dto.SignInDto)),
	)
	group.POST("/sign-up",
		authController.SignUp,
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
		appmw.ValidateRequest(new(dto.SignUpDto)),
	)
	group.POST("/verify-email",
		authController.VerifyEmail,
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
		appmw.ValidateRequest(new(dto.VerifyEmailDto)),
	)
	group.POST("/resend-verification",
		authController.ResendVerification,
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
		appmw.ValidateRequest(new(dto.EmailDto)),
	)
	group.POST("/forgot-password",
		authController.ForgotPassword,
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
		appmw.ValidateRequest(new(dto.EmailDto)),
	)
	group.POST("/reset-password",
		authController.ResetPassword,
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
		appmw.ValidateRequest(new(dto.ResetPasswordDto)),
	)
	group.POST("/refresh",
		authController.Refresh,
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
	)
	group.POST("/logout",
		authController.Logout,
		appmw.RequireOrigin(cfg.CORSAllowedOrigins),
	)
}
