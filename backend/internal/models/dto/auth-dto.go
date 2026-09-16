package dto

// LoginQueryDto binds the query of GET /v1/auth/login.
type LoginQueryDto struct {
	ReturnTo string `query:"returnTo" json:"returnTo"`
	Provider string `query:"provider" json:"provider" validate:"required,oneof=google"`
}

// CallbackQueryDto binds the query of GET /v1/auth/callback. Both values
// are optional at binding time. The service decides what an absent value
// means.
type CallbackQueryDto struct {
	Code  string `query:"code" json:"code"`
	State string `query:"state" json:"state"`
}

// SignInDto binds the body of POST /v1/auth/sign-in.
type SignInDto struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,min=8,max=256"`
}

// VerifyEmailDto binds the body of POST /v1/auth/verify-email. The
// pending token comes from a sign-in or sign-up answered with 202.
type VerifyEmailDto struct {
	PendingToken string `json:"pending_token" validate:"required,max=512"`
	Code         string `json:"code" validate:"required,len=6,numeric"`
}

// SignUpDto binds the body of POST /v1/auth/sign-up.
type SignUpDto struct {
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name" validate:"required,max=100"`
	Email     string `json:"email" validate:"required,email,max=254"`
	Password  string `json:"password" validate:"required,min=8,max=256"`
}

// EmailDto binds the body of the endpoints that take an address alone.
type EmailDto struct {
	Email string `json:"email" validate:"required,email,max=254"`
}

// ResetPasswordDto binds the body of POST /v1/auth/reset-password. The
// token comes from the link in the reset email.
type ResetPasswordDto struct {
	Token    string `json:"token" validate:"required,max=512"`
	Password string `json:"password" validate:"required,min=8,max=256"`
}
