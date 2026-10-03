package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/infra/mailer"
	"go.uber.org/zap"
)

func (p *RedisTaskProcessor) ProcessEmailVerification(ctx context.Context, t *asynq.Task) error {
	var payload EmailVerificationPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		p.l.Error("failed to unmarshal email verification payload",
			zap.Error(err),
			zap.ByteString("payload", t.Payload()),
		)
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	p.l.Info("processing email verification task",
		zap.String("user_id", payload.UserID),
		zap.String("email", payload.Email),
	)

	err := p.mailer.SendTemplate(ctx, payload.Email, "Verify your email", "email-verification.html", payload)
	if err != nil {
		if errors.Is(err, mailer.ErrMailerDisabled) {
			p.l.Warn("mailer is disabled, skipping task", zap.String("email", payload.Email))
			return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
		}
		p.l.Error("failed to send verification email",
			zap.String("email", payload.Email),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	p.l.Info("email verification sent successfully", zap.String("email", payload.Email))
	return nil
}

func (p *RedisTaskProcessor) ProcessEmailPasswordReset(ctx context.Context, t *asynq.Task) error {
	var payload EmailPasswordResetPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		p.l.Error("failed to unmarshal password reset payload",
			zap.Error(err),
			zap.ByteString("payload", t.Payload()),
		)
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	p.l.Info("processing password reset task",
		zap.String("user_id", payload.UserID),
		zap.String("email", payload.Email),
	)

	err := p.mailer.SendTemplate(ctx, payload.Email, "Reset your password", "password-reset.html", payload)
	if err != nil {
		if errors.Is(err, mailer.ErrMailerDisabled) {
			p.l.Warn("mailer is disabled, skipping task", zap.String("email", payload.Email))
			return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
		}
		p.l.Error("failed to send password reset email",
			zap.String("email", payload.Email),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	p.l.Info("password reset email sent successfully", zap.String("email", payload.Email))
	return nil
}
