package stores

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v5"

	"github.com/pixels-two/sow/backend/internal/stores/sqlcgen"
)

// HealthStore probes the database through the standard query stack.
type HealthStore struct {
	queries *sqlcgen.Queries
}

func NewHealthStore(db sqlcgen.DBTX) *HealthStore {
	return &HealthStore{queries: sqlcgen.New(db)}
}

// WithTx returns a copy bound to the transaction.
func (s *HealthStore) WithTx(tx pgx.Tx) *HealthStore {
	return &HealthStore{queries: s.queries.WithTx(tx)}
}

// Probe verifies the database answers a trivial query within one second.
func (s *HealthStore) Probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if _, err := s.queries.HealthProbe(ctx); err != nil {
		return errors.Wrap(err, "probing database")
	}
	return nil
}
