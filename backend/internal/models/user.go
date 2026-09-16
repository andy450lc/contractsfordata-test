package models

import "time"

// User is the platform record of a registered person. The identity
// provider owns credentials. ID is the provider's opaque user id.
// Timestamps mirror the provider's view of the user.
type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
