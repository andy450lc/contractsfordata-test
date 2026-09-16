package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/workos/workos-go/v10"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/logging"
	"github.com/pixels-two/sow/backend/internal/models"
	"github.com/pixels-two/sow/backend/internal/models/dto"
	"github.com/pixels-two/sow/backend/internal/stores"
)

// UserService syncs and serves platform user records.
type UserService struct {
	users    *stores.UserStore
	provider WorkOSClient
}

func NewUserService(users *stores.UserStore, provider WorkOSClient) *UserService {
	return &UserService{users: users, provider: provider}
}

// UpsertFromProvider stores the provider's view of a user through the
// order-safe upsert.
func (s *UserService) UpsertFromProvider(ctx context.Context, remote *workos.User) (models.User, error) {
	user, err := userFromProvider(remote)
	if err != nil {
		return models.User{}, fmt.Errorf("mapping provider user: %w", err)
	}

	stored, err := s.users.UpsertIfNewer(ctx, user)
	if err != nil {
		return models.User{}, fmt.Errorf("storing user: %w", err)
	}
	return *stored, nil
}

// Me returns the user record for the authenticated caller. A missing
// record is provisioned from the provider before responding.
func (s *UserService) Me(ctx context.Context, userID string) (models.User, error) {
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return models.User{}, fmt.Errorf("loading user record: %w", err)
	}
	if user != nil {
		return *user, nil
	}

	remote, err := s.provider.GetUser(ctx, userID)
	if err != nil {
		var apiErr *workos.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return models.User{}, fmt.Errorf("user absent at provider: %w", common.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("fetching user from provider: %w", err)
	}

	provisioned, err := s.UpsertFromProvider(ctx, remote)
	if err != nil {
		return models.User{}, fmt.Errorf("provisioning user record: %w", err)
	}

	logging.Info(ctx, "user record provisioned from provider", "user_id", provisioned.ID)
	return provisioned, nil
}

// ApplyWebhookEvent applies a verified provider event to the user
// store. Replayed and out-of-order deliveries are safe. Event types
// outside user.created, user.updated, and user.deleted are ignored.
func (s *UserService) ApplyWebhookEvent(ctx context.Context, event dto.WorkOSEventDto) error {
	switch event.Event {
	case "user.created", "user.updated":
		user, err := userFromEventData(event.Data)
		if err != nil {
			return fmt.Errorf("parsing user event data: %w", err)
		}
		if _, err := s.users.UpsertIfNewer(ctx, user); err != nil {
			return fmt.Errorf("applying user event: %w", err)
		}
		logging.Info(ctx, "user event applied", "event_type", event.Event, "event_id", event.ID, "user_id", user.ID)
		return nil

	case "user.deleted":
		user, err := userFromEventData(event.Data)
		if err != nil {
			return fmt.Errorf("parsing user event data: %w", err)
		}
		if err := s.users.Delete(ctx, user.ID); err != nil {
			return fmt.Errorf("applying user deletion: %w", err)
		}
		logging.Info(ctx, "user event applied", "event_type", event.Event, "event_id", event.ID, "user_id", user.ID)
		return nil

	default:
		logging.Debug(ctx, "webhook event ignored", "event_type", event.Event, "event_id", event.ID)
		return nil
	}
}

func userFromEventData(data json.RawMessage) (models.User, error) {
	var payload dto.WorkOSUserDataDto
	if err := json.Unmarshal(data, &payload); err != nil {
		return models.User{}, &common.ValidationError{Details: []common.ValidationDetail{
			{Field: "data", Message: "not a user object"},
		}}
	}
	return userFromPayload(payload)
}

func userFromProvider(remote *workos.User) (models.User, error) {
	if remote == nil {
		return models.User{}, &common.ValidationError{Details: []common.ValidationDetail{
			{Field: "user", Message: "required"},
		}}
	}
	return userFromPayload(dto.WorkOSUserDataDto{
		ID:        remote.ID,
		Email:     remote.Email,
		FirstName: remote.FirstName,
		LastName:  remote.LastName,
		CreatedAt: remote.CreatedAt,
		UpdatedAt: remote.UpdatedAt,
	})
}

func userFromPayload(payload dto.WorkOSUserDataDto) (models.User, error) {
	if payload.ID == "" || payload.Email == "" {
		return models.User{}, &common.ValidationError{Details: []common.ValidationDetail{
			{Field: "data.id", Message: "id and email are required"},
		}}
	}

	createdAt, err := time.Parse(time.RFC3339Nano, payload.CreatedAt)
	if err != nil {
		return models.User{}, &common.ValidationError{Details: []common.ValidationDetail{
			{Field: "data.created_at", Message: "not an ISO 8601 timestamp"},
		}}
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, payload.UpdatedAt)
	if err != nil {
		return models.User{}, &common.ValidationError{Details: []common.ValidationDetail{
			{Field: "data.updated_at", Message: "not an ISO 8601 timestamp"},
		}}
	}

	return models.User{
		ID:        payload.ID,
		Email:     payload.Email,
		Name:      displayName(payload.FirstName, payload.LastName),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func displayName(parts ...*string) string {
	var kept []string
	for _, p := range parts {
		if p != nil && strings.TrimSpace(*p) != "" {
			kept = append(kept, strings.TrimSpace(*p))
		}
	}
	return strings.Join(kept, " ")
}
