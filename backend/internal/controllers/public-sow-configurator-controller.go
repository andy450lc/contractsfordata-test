package controllers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/models/dto"
	"github.com/pixels-two/sow/backend/internal/services"
)

// PublicSOWConfiguratorController serves the public four-question configurator.
type PublicSOWConfiguratorController struct {
	service *services.PublicSOWConfiguratorService
}

// NewPublicSOWConfiguratorController builds the public configurator controller.
func NewPublicSOWConfiguratorController(service *services.PublicSOWConfiguratorService) *PublicSOWConfiguratorController {
	return &PublicSOWConfiguratorController{service: service}
}

// Download returns a configured public Word document as an attachment.
func (c *PublicSOWConfiguratorController) Download(e *echo.Context) error {
	request := e.Get(common.EchoContextKeyValidatedDTO).(*dto.PublicSOWDownloadDto)
	artifact, err := c.service.Download(e.Request().Context(), request.PublicSOWDownloadRequest)
	if err != nil {
		return fmt.Errorf("downloading public SOW document: %w", err)
	}
	e.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+artifact.Filename+`"`)
	e.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return e.Blob(http.StatusOK, artifact.MIMEType, artifact.Bytes)
}

// Email sends the configured public Word document through the provider.
func (c *PublicSOWConfiguratorController) Email(e *echo.Context) error {
	request := e.Get(common.EchoContextKeyValidatedDTO).(*dto.PublicSOWEmailDto)
	_, err := c.service.Email(e.Request().Context(), request.PublicSOWEmailRequest, request.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("emailing public SOW document: %w", err)
	}
	return e.JSON(http.StatusAccepted, dto.EmailDeliveryAccepted{Accepted: dto.EmailDeliveryAcceptedAcceptedTrue})
}
