package stores

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/pixels-two/sow/backend/internal/models"
	"github.com/pixels-two/sow/backend/internal/stores/sqlcgen"
)

// UserStore reads and writes platform user records.
type UserStore struct {
	queries *sqlcgen.Queries
}

func NewUserStore(db sqlcgen.DBTX) *UserStore {
	return &UserStore{queries: sqlcgen.New(db)}
}

// WithTx returns a copy bound to the transaction.
func (s *UserStore) WithTx(tx pgx.Tx) *UserStore {
	return &UserStore{queries: s.queries.WithTx(tx)}
}

// Get returns the user with the given id. A missing user returns nil
// with a nil error.
func (s *UserStore) Get(ctx context.Context, id string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row, err := s.queries.GetUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "selecting user row")
	}

	user := userFromRow(row)
	return &user, nil
}

// UpsertIfNewer inserts the user or updates email, name, and
// updated_at in place. A write older than the stored row changes
// nothing, so replayed and out-of-order writes are safe. Returns the
// stored row afterwards.
func (s *UserStore) UpsertIfNewer(ctx context.Context, user models.User) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row, err := s.queries.UpsertUserIfNewer(ctx, sqlcgen.UpsertUserIfNewerParams{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: pgtype.Timestamptz{Time: user.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: user.UpdatedAt, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return s.Get(ctx, user.ID)
	}
	if err != nil {
		return nil, errors.Wrap(err, "upserting user row")
	}

	stored := userFromRow(row)
	return &stored, nil
}

// Delete removes the user with the given id. Deleting an absent user is
// a no-op.
func (s *UserStore) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.queries.DeleteUser(ctx, id); err != nil {
		return errors.Wrap(err, "deleting user row")
	}
	return nil
}

func userFromRow(row sqlcgen.User) models.User {
	return models.User{
		ID:        row.ID,
		Email:     row.Email,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
