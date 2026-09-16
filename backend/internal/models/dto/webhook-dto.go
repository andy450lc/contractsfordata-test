package dto

import "encoding/json"

// WorkOSEventDto is the event envelope read from the raw, verified
// webhook body. Data stays raw until the event type is known.
type WorkOSEventDto struct {
	ID    string          `json:"id"`
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// WorkOSUserDataDto is the user object carried by user.* events.
// Timestamps are ISO 8601 strings owned by the provider.
type WorkOSUserDataDto struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}
