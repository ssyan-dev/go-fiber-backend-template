package queue

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"

	TypeEmailVerification  = "task:email:verification"
	TypeEmailPasswordReset = "task:email:password_reset"
)

type EmailVerificationPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	URL    string `json:"url"`
	Code   string `json:"code"`
}

type EmailPasswordResetPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	URL    string `json:"url"`
	Code   string `json:"code"`
}
