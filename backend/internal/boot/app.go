package boot

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/pixels-two/sow/backend/internal/config"
	"github.com/pixels-two/sow/backend/internal/controllers"
	"github.com/pixels-two/sow/backend/internal/logging"
	"github.com/pixels-two/sow/backend/internal/metrics"
	"github.com/pixels-two/sow/backend/internal/middleware"
	"github.com/pixels-two/sow/backend/internal/routes"
	"github.com/pixels-two/sow/backend/internal/services"
	"github.com/pixels-two/sow/backend/internal/stores"
	"github.com/pixels-two/sow/backend/internal/stores/sqlcgen"
)

// NewLogger builds the root logger and installs it as the process default.
func NewLogger(cfg config.AppConfig) *slog.Logger {
	logger := logging.New(cfg.Stage)
	slog.SetDefault(logger)
	return logger
}

// AppOptions is the complete fx dependency graph. Both main and the
// integration suite boot it.
func AppOptions() fx.Option {
	return fx.Options(
		// Bootstrap
		fx.Provide(
			NewAppConfig,
			NewLogger,
			NewDatabasePool,
			func(pool *pgxpool.Pool) sqlcgen.DBTX { return pool },
			NewTracerProvider,
			NewPropagator,
			metrics.New,
			NewServer,
		),

		// Identity provider and session primitives
		fx.Provide(
			NewWorkOSClient,
			NewWorkOSUserManagement,
			NewWebhookVerifier,
			NewSessionKeyring,
			NewAuthMiddleware,
			NewMailer,
			fx.Annotate(services.NewSDKWorkOSClient, fx.As(new(services.WorkOSClient))),
		),

		// Stores
		fx.Provide(
			stores.NewHealthStore,
			stores.NewUserStore,
		),

		// Services
		fx.Provide(
			services.NewHealthService,
			services.NewUserService,
			services.NewAuthService,
			fx.Annotate(services.NewDocumentGenerator, fx.As(new(services.DocumentGenerator))),
			fx.Annotate(services.NewPublicSOWDocumentGenerator, fx.As(new(services.PublicSOWDocumentGenerator))),
			services.NewTemplateDeliveryService,
			services.NewPublicSOWConfiguratorService,
		),

		// Controllers
		fx.Provide(
			controllers.NewHealthController,
			controllers.NewAuthController,
			controllers.NewUserController,
			controllers.NewWebhookController,
			controllers.NewTemplateDeliveryController,
			controllers.NewPublicSOWConfiguratorController,
		),

		// Route registration + server start
		fx.Invoke(
			InitSentry,
			InstallOTelGlobals,
			func(s *Server, hc *controllers.HealthController) {
				routes.AddHealthRoutes(s.Echo, hc)
			},
			func(s *Server, cfg config.AppConfig, ac *controllers.AuthController) {
				routes.AddAuthRoutes(s.Echo, cfg, ac)
			},
			func(s *Server, am *middleware.AuthMiddleware, uc *controllers.UserController) {
				routes.AddUserRoutes(s.Echo, am, uc)
			},
			func(s *Server, wc *controllers.WebhookController) {
				routes.AddWebhookRoutes(s.Echo, wc)
			},
			func(s *Server, cfg config.AppConfig, tc *controllers.TemplateDeliveryController) {
				routes.AddTemplateDeliveryRoutes(s.Echo, cfg, tc)
			},
			func(s *Server, cfg config.AppConfig, pc *controllers.PublicSOWConfiguratorController) {
				routes.AddPublicSOWConfiguratorRoutes(s.Echo, cfg, pc)
			},
			RegisterServerLifecycle,
		),

		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),
	)
}
