package mail

import (
	"fmt"
	"time"
)

// PasswordResetMessage builds the email sent when WorkOS issues a
// password reset token for an address. resetURL is the full link the
// signer follows, expiresAt the moment the link stops working.
func PasswordResetMessage(webAppURL string, to string, resetURL string, expiresAt time.Time) Message {
	text := fmt.Sprintf(
		"Someone asked to reset the password for this SoW account.\n\n"+
			"Choose a new password here:\n%s\n\n"+
			"The link works once and expires at %s.\n\n"+
			"If you did not ask for this, you can ignore this email. Your password\nstays the same.",
		resetURL,
		expiresAt.UTC().Format(time.RFC1123),
	)

	return Message{To: to, Subject: "Reset your SoW password", Text: text}
}

// AccountExistsMessage builds the email sent when a sign-up request
// names an address that already has an account.
func AccountExistsMessage(webAppURL string, to string) Message {
	text := fmt.Sprintf(
		"Someone tried to create a SoW account with this email address, but one\nalready exists.\n\n"+
			"Sign in here:\n%s/\n\n"+
			"Forgot your password? Reset it here:\n%s/forgot-password\n\n"+
			"If this was not you, you can ignore this email.",
		webAppURL,
		webAppURL,
	)

	return Message{To: to, Subject: "You already have a SoW account", Text: text}
}
