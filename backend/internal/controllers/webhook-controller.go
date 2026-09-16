package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/workos/workos-go/v10"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/models/dto"
	"github.com/pixels-two/sow/backend/internal/services"
)

// WebhookController receives WorkOS webhook deliveries. Requests are
// authenticated by their signature header. Verification reads the raw
// body before any JSON binding.
type WebhookController struct {
	verifier *workos.WebhookVerifier
	users    *services.UserService
}

func NewWebhookController(verifier *workos.WebhookVerifier, users *services.UserService) *WebhookController {
	return &WebhookController{verifier: verifier, users: users}
}

// ReceiveWorkOS verifies and applies one webhook delivery.
func (wc *WebhookController) ReceiveWorkOS(c *echo.Context) error {
	ctx := c.Request().Context()

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return fmt.Errorf("reading webhook body: %w", err)
	}

	verified, err := wc.verifier.VerifyPayload(c.Request().Header.Get("WorkOS-Signature"), string(body))
	if err != nil {
		return fmt.Errorf("verifying webhook signature: %w", common.ErrUnauthorized)
	}

	var event dto.WorkOSEventDto
	if err := json.Unmarshal([]byte(verified), &event); err != nil || event.Event == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bad_request")
	}

	if err := wc.users.ApplyWebhookEvent(ctx, event); err != nil {
		return fmt.Errorf("handling webhook event: %w", err)
	}

	return c.NoContent(http.StatusOK)
}
