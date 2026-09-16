package models

// HealthCheckResult is one dependency check's outcome. Error carries a
// short operational message only — never connection strings or PII.
type HealthCheckResult struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

// HealthResponse is the livez and readyz response body.
type HealthResponse struct {
	Status string              `json:"status"`
	Checks []HealthCheckResult `json:"checks,omitempty"`
}

const (
	HealthStatusOK          = "ok"
	HealthStatusUnavailable = "unavailable"
)
