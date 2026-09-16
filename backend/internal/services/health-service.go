package services

import (
	"context"

	"github.com/pixels-two/sow/backend/internal/models"
	"github.com/pixels-two/sow/backend/internal/stores"
)

// HealthService aggregates dependency checks for readiness. Each check
// bounds its own execution.
type HealthService struct {
	healthStore *stores.HealthStore
}

func NewHealthService(healthStore *stores.HealthStore) *HealthService {
	return &HealthService{healthStore: healthStore}
}

// Readiness runs every registered check and reports overall readiness.
func (s *HealthService) Readiness(ctx context.Context) (bool, []models.HealthCheckResult) {
	checks := []models.HealthCheckResult{
		s.checkDatabase(ctx),
	}

	ready := true
	for _, check := range checks {
		if !check.Healthy {
			ready = false
		}
	}
	return ready, checks
}

func (s *HealthService) checkDatabase(ctx context.Context) models.HealthCheckResult {
	if err := s.healthStore.Probe(ctx); err != nil {
		return models.HealthCheckResult{
			Name: "database",
			// The error chain never contains the DSN.
			Error: err.Error(),
		}
	}
	return models.HealthCheckResult{Name: "database", Healthy: true}
}
