package controllers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/models/dto"
	"github.com/pixels-two/sow/backend/internal/services"
)

// TemplateDeliveryController serves public request-scoped template delivery.
type TemplateDeliveryController struct {
	service *services.TemplateDeliveryService
}

// NewTemplateDeliveryController builds the template delivery controller.
func NewTemplateDeliveryController(service *services.TemplateDeliveryService) *TemplateDeliveryController {
	return &TemplateDeliveryController{service: service}
}

// Download returns an acknowledged Word document as an attachment.
func (c *TemplateDeliveryController) Download(e *echo.Context) error {
	ctx := e.Request().Context()
	request := e.Get(common.EchoContextKeyValidatedDTO).(*dto.TemplateDownloadDto)

	artifact, err := c.service.Download(ctx, bool(request.Acknowledged), request.Configuration)
	if err != nil {
		return fmt.Errorf("downloading template document: %w", err)
	}

	e.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+artifact.Filename+`"`)
	e.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return e.Blob(http.StatusOK, artifact.MIMEType, artifact.Bytes)
}

// Email sends an acknowledged Word document through the configured provider.
func (c *TemplateDeliveryController) Email(e *echo.Context) error {
	ctx := e.Request().Context()
	request := e.Get(common.EchoContextKeyValidatedDTO).(*dto.TemplateEmailDto)

	_, err := c.service.Email(ctx, bool(request.Acknowledged), string(request.Email), request.IdempotencyKey, request.Configuration)
	if err != nil {
		return fmt.Errorf("emailing template document: %w", err)
	}

	return e.JSON(http.StatusAccepted, dto.EmailDeliveryAccepted{Accepted: dto.EmailDeliveryAcceptedAcceptedTrue})
}
