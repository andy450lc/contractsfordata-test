package dto

// PublicSOWDownloadDto binds a public configured download request.
type PublicSOWDownloadDto struct {
	PublicSOWDownloadRequest
}

// StrictJSON marks this request for unknown-field rejection.
func (d *PublicSOWDownloadDto) StrictJSON() {}

// PublicSOWEmailDto binds a public configured email request and idempotency header.
type PublicSOWEmailDto struct {
	PublicSOWEmailRequest
	IdempotencyKey string `header:"Idempotency-Key" validate:"required,min=8,max=256,printascii"`
}

// StrictJSON marks this request for unknown-field rejection.
func (d *PublicSOWEmailDto) StrictJSON() {}
