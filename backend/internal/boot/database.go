package boot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/config"
)

// NewDatabasePool builds the pgx pool from config. Pings and fails fast
// at startup on errors. Closes the pool on OnStop.
func NewDatabasePool(lc fx.Lifecycle, cfg config.AppConfig) (*pgxpool.Pool, error) {
	// The pgx parse error embeds the connection string. Return a static
	// message so the value never reaches logs.
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, errors.New(
			"parsing DATABASE_URL: malformed connection string (value omitted). " +
				"Percent-encode special characters in the password and single-quote the value in .env")
	}

	poolCfg.MaxConns = cfg.DBPoolMaxConns
	poolCfg.MinConns = cfg.DBPoolMinConns

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating database pool: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := pool.Ping(pingCtx); err != nil {
				return fmt.Errorf("pinging database at startup: %w", err)
			}
			return nil
		},
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})
	return pool, nil
}
