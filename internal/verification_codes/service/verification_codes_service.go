package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	mailerService "github.com/ssyan-dev/go-fiber-backend-template/internal/infra/mailer"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/code"
	userService "github.com/ssyan-dev/go-fiber-backend-template/internal/user/service"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/verification_codes/repository"
	"go.uber.org/zap"
)

func mustParseUUID(s string) uuid.UUID {
	return uuid.MustParse(s)
}

var (
	ErrInvalidCode          = errors.New("invalid or expired verification code")
	ErrUserNotFound         = errors.New("user not found")
	ErrEmailAlreadyVerified = errors.New("email is already verified")
)

type VerificationCodeService interface {
	Create(ctx context.Context, userID string, t models.VerificationType, ttl time.Duration) (string, error)
	Verify(ctx context.Context, code string, t models.VerificationType) (*models.VerificationCode, error)
	DeleteByUserIDAndType(ctx context.Context, userID string, t models.VerificationType) error
	VerifyEmail(ctx context.Context, code string) error
	SendEmailVerificationCode(ctx context.Context, userID, email string) error
	ResendEmailVerificationCode(ctx context.Context, email string) error
}

type verificationCodeSvc struct {
	repo        repository.VerificationCodeRepository
	userSvc     userService.UserService
	mailerSvc   mailerService.MailerService
	redirectURL string
	l           *zap.Logger
}

func NewVerificationCodeService(
	repo repository.VerificationCodeRepository,
	userSvc userService.UserService,
	mailerSvc mailerService.MailerService,
	redirectURL string,
	l *zap.Logger,
) VerificationCodeService {
	return &verificationCodeSvc{
		repo:        repo,
		userSvc:     userSvc,
		mailerSvc:   mailerSvc,
		redirectURL: redirectURL,
		l:           l,
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

func (s *verificationCodeSvc) VerifyEmail(ctx context.Context, code string) error {
	vc, err := s.Verify(ctx, code, models.VerificationTypeEmail)
	if err != nil {
		return err
	}

	if err := s.userSvc.SetEmailVerified(ctx, vc.UserID.String()); err != nil {
		s.l.Error("failed to set user email verified", zap.Error(err))
		return err
	}

	return nil
}

func (s *verificationCodeSvc) SendEmailVerificationCode(ctx context.Context, userID, email string) error {
	verificationCode, err := s.Create(ctx, userID, models.VerificationTypeEmail, 10*time.Minute)
	if err != nil {
		return err
	}

	return s.mailerSvc.SendTemplate(ctx, email, "Verify your email", "email-verification.html", map[string]string{
		"Email": email,
		"URL":   s.redirectURL,
		"Code":  verificationCode,
	})
}

func (s *verificationCodeSvc) ResendEmailVerificationCode(ctx context.Context, email string) error {
	user, err := s.userSvc.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	if user.IsEmailVerified {
		return ErrEmailAlreadyVerified
	}

	return s.SendEmailVerificationCode(ctx, user.ID.String(), user.Email)
}
