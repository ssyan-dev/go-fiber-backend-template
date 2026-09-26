package models

import (
	"time"

	"github.com/google/uuid"
)

type VerificationType string

const (
	VerificationTypeEmail         VerificationType = "email_verification"
	VerificationTypePasswordReset VerificationType = "password_reset"
)

type VerificationCode struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	UserID    uuid.UUID        `json:"user_id" db:"user_id"`
	Code      string           `json:"-" db:"code"`
	Type      VerificationType `json:"type" db:"type"`
	ExpiresAt time.Time        `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
}
