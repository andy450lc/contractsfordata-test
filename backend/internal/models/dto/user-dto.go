package dto

import (
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/pixels-two/sow/backend/internal/models"
)

// UserFrom converts a user record to its API shape.
func UserFrom(user models.User) User {
	return User{
		Id:        user.ID,
		Email:     openapi_types.Email(user.Email),
		Name:      user.Name,
		CreatedAt: user.CreatedAt.UnixMilli(),
		UpdatedAt: user.UpdatedAt.UnixMilli(),
	}
}
