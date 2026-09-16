package dto

// TemplateDownloadDto binds the generated download request.
type TemplateDownloadDto struct {
	DocumentDownloadRequest
}

// StrictJSON marks this request for unknown-field rejection.
func (d *TemplateDownloadDto) StrictJSON() {}

// TemplateEmailDto binds the generated email request and idempotency header.
type TemplateEmailDto struct {
	EmailDocumentRequest
	IdempotencyKey string `header:"Idempotency-Key" validate:"required,min=8,max=256,printascii"`
}

// StrictJSON marks this request for unknown-field rejection.
func (d *TemplateEmailDto) StrictJSON() {}
