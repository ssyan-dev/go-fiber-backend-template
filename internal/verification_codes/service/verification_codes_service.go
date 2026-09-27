package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/code"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/verification_codes/repository"
	"go.uber.org/zap"
)

func mustParseUUID(s string) uuid.UUID {
	return uuid.MustParse(s)
}

var (
	ErrInvalidCode = errors.New("invalid or expired verification code")
)

type VerificationCodeService interface {
	Create(ctx context.Context, userID string, t models.VerificationType, ttl time.Duration) (string, error)
	Verify(ctx context.Context, code string, t models.VerificationType) (*models.VerificationCode, error)
	DeleteByUserIDAndType(ctx context.Context, userID string, t models.VerificationType) error
}

type verificationCodeSvc struct {
	repo repository.VerificationCodeRepository
	l    *zap.Logger
}

func NewVerificationCodeService(
	repo repository.VerificationCodeRepository,
	l *zap.Logger,
) VerificationCodeService {
	return &verificationCodeSvc{
		repo: repo,
		l:    l,
	}
}

func (s *verificationCodeSvc) Create(ctx context.Context, userID string, t models.VerificationType, ttl time.Duration) (string, error) {
	_ = s.repo.DeleteByUserIDAndType(ctx, userID, t)

	verificationCode, err := code.GenerateVerificationCode()
	if err != nil {
		s.l.Error("failed to generate verification code", zap.Error(err))
		return "", err
	}

	vc := &models.VerificationCode{
		UserID:    mustParseUUID(userID),
		Code:      verificationCode,
		Type:      t,
		ExpiresAt: time.Now().Add(ttl),
	}

	if err := s.repo.Create(ctx, vc); err != nil {
		s.l.Error("failed to save verification code", zap.Error(err))
		return "", err
	}

	return verificationCode, nil
}

func (s *verificationCodeSvc) Verify(ctx context.Context, verificationCode string, t models.VerificationType) (*models.VerificationCode, error) {
	vc, err := s.repo.GetByCode(ctx, verificationCode)
	if err != nil {
		return nil, ErrInvalidCode
	}

	if vc.Type != t {
		return nil, ErrInvalidCode
	}

	if time.Now().After(vc.ExpiresAt) {
		_ = s.repo.DeleteByUserIDAndType(ctx, vc.UserID.String(), t)
		return nil, ErrInvalidCode
	}

	_ = s.repo.DeleteByUserIDAndType(ctx, vc.UserID.String(), t)

	return vc, nil
}

func (s *verificationCodeSvc) DeleteByUserIDAndType(ctx context.Context, userID string, t models.VerificationType) error {
	return s.repo.DeleteByUserIDAndType(ctx, userID, t)
}
