package boot

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"

	"github.com/pixels-two/sow/backend/internal/config"
)

// NewAppConfig parses and validates configuration. Fails on missing or
// invalid config. Errors name the offending variables and omit the values.
func NewAppConfig() (config.AppConfig, error) {
	LoadDotenv()

	cfg, err := env.ParseAs[config.AppConfig]()
	if err != nil {
		return config.AppConfig{}, fmt.Errorf("parsing environment config: %w", err)
	}

	if err := validator.New().Struct(cfg); err != nil {
		return config.AppConfig{}, fmt.Errorf("validating config (values omitted): %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return config.AppConfig{}, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}
