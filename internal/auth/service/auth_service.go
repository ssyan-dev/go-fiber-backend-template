package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/infra/mailer"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	sessionService "github.com/ssyan-dev/go-fiber-backend-template/internal/sessions/service"
	userService "github.com/ssyan-dev/go-fiber-backend-template/internal/user/service"
	vcService "github.com/ssyan-dev/go-fiber-backend-template/internal/verification_codes/service"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists    = userService.ErrUserAlreadyExists
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrInvalidLoginType     = errors.New("your account has no password. use oauth")
	ErrInvalidToken         = errors.New("invalid token")
	ErrEmailMustBeVerified  = errors.New("please verify your email")
	ErrUserNotFound         = errors.New("user not found")
	ErrEmailAlreadyVerified = errors.New("email is already verified")
	ErrInvalidCode          = vcService.ErrInvalidCode
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*models.User, error)
	Login(ctx context.Context, email, password, ip, userAgent string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken, ip, userAgent string) (string, string, error)
	VerifyEmail(ctx context.Context, code string) error
	ResendEmailVerification(ctx context.Context, email string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, code, newPassword string) error
	GetRefreshTokenTTL() time.Duration
}

type authSvc struct {
	userSvc    userService.UserService
	sessionSvc sessionService.SessionService
	vcSvc      vcService.VerificationCodeService
	mailerSvc  mailer.MailerService
	cfg        *config.JWTConfig
	authCfg    *config.AuthConfig
	baseURL    string
	l          *zap.Logger
}

func NewAuthService(
	userSvc userService.UserService,
	sessionSvc sessionService.SessionService,
	vcSvc vcService.VerificationCodeService,
	mailerSvc mailer.MailerService,
	cfg *config.JWTConfig,
	authCfg *config.AuthConfig,
	baseURL string,
	l *zap.Logger,
) AuthService {
	return &authSvc{
		userSvc:    userSvc,
		sessionSvc: sessionSvc,
		vcSvc:      vcSvc,
		mailerSvc:  mailerSvc,
		cfg:        cfg,
		authCfg:    authCfg,
		baseURL:    baseURL,
		l:          l,
	}
}

func (s *authSvc) Register(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.userSvc.Create(ctx, email, password)
	if err != nil {
		if !errors.Is(err, userService.ErrUserAlreadyExists) {
			s.l.Error("failed to create user", zap.Error(err))
		}
		return nil, err
	}

	if err := s.sendEmailVerificationCode(ctx, user.ID.String(), user.Email); err != nil {
		s.l.Error("failed to send verification email", zap.Error(err))
	}

	return user, nil
}

func (s *authSvc) sendEmailVerificationCode(ctx context.Context, userID, email string) error {
	verificationCode, err := s.vcSvc.Create(ctx, userID, models.VerificationTypeEmail, 10*time.Minute)
	if err != nil {
		return err
	}

	return s.mailerSvc.SendTemplate(ctx, email, "Verify your email", "email-verification.html", map[string]string{
		"Email": email,
		"URL":   s.baseURL + "/auth/verify-email",
		"Code":  verificationCode,
	})
}

func (s *authSvc) VerifyEmail(ctx context.Context, code string) error {
	vc, err := s.vcSvc.Verify(ctx, code, models.VerificationTypeEmail)
	if err != nil {
		return err
	}

	if err := s.userSvc.SetEmailVerified(ctx, vc.UserID.String()); err != nil {
		s.l.Error("failed to set user email verified", zap.Error(err))
		return err
	}

	return nil
}

func (s *authSvc) ResendEmailVerification(ctx context.Context, email string) error {
	user, err := s.userSvc.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	if user.IsEmailVerified {
		return ErrEmailAlreadyVerified
	}

	return s.sendEmailVerificationCode(ctx, user.ID.String(), user.Email)
}

func (s *authSvc) Login(ctx context.Context, email, password, ip, userAgent string) (string, string, error) {
	user, err := s.userSvc.GetByEmail(ctx, email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if user.PasswordHash == nil {
		return "", "", ErrInvalidLoginType
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	if s.authCfg.VerifyEmail && !user.IsEmailVerified {
		return "", "", ErrEmailMustBeVerified
	}

	accessToken, err := s.generateJWTToken(user, s.cfg.AccessTokenTTL)
	if err != nil {
		s.l.Error("failed to create access token", zap.Error(err))
		return "", "", err
	}

	refreshToken, err := s.generateJWTToken(user, s.cfg.RefreshTokenTTL)
	if err != nil {
		s.l.Error("failed to create refresh token", zap.Error(err))
		return "", "", err
	}

	session := &models.Session{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		IP:           ip,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.sessionSvc.Create(ctx, session); err != nil {
		s.l.Error("failed to create session", zap.Error(err))
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *authSvc) Logout(ctx context.Context, refreshToken string) error {
	return s.sessionSvc.DeleteByRefreshToken(ctx, refreshToken)
}

func (s *authSvc) Refresh(ctx context.Context, refreshToken, ip, userAgent string) (string, string, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.SecretKey), nil
	})

	if err != nil || !token.Valid {
		return "", "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", "", ErrInvalidToken
	}

	session, err := s.sessionSvc.GetByRefreshToken(ctx, refreshToken)
	if err != nil || session.IsBlocked || session.ExpiresAt.Before(time.Now()) {
		return "", "", ErrInvalidToken
	}

	user, err := s.userSvc.GetByID(ctx, userID)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	newAccessToken, err := s.generateJWTToken(user, s.cfg.AccessTokenTTL)
	if err != nil {
		return "", "", err
	}

	newRefreshToken, err := s.generateJWTToken(user, s.cfg.RefreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	_ = s.sessionSvc.DeleteByRefreshToken(ctx, refreshToken)

	newSession := &models.Session{
		UserID:       user.ID,
		RefreshToken: newRefreshToken,
		AccessToken:  newAccessToken,
		IP:           ip,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.sessionSvc.Create(ctx, newSession); err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *authSvc) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userSvc.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	verificationCode, err := s.vcSvc.Create(ctx, user.ID.String(), models.VerificationTypePasswordReset, 15*time.Minute)
	if err != nil {
		return err
	}

	return s.mailerSvc.SendTemplate(ctx, email, "Reset your password", "password-reset.html", map[string]string{
		"Email": email,
		"URL":   s.baseURL + "/auth/reset-password",
		"Code":  verificationCode,
	})
}

func (s *authSvc) ResetPassword(ctx context.Context, code, newPassword string) error {
	vc, err := s.vcSvc.Verify(ctx, code, models.VerificationTypePasswordReset)
	if err != nil {
		return err
	}

	if err := s.userSvc.ResetPassword(ctx, vc.UserID.String(), newPassword); err != nil {
		s.l.Error("failed to reset password", zap.Error(err))
		return err
	}

	if err := s.sessionSvc.DeleteAllByUserID(ctx, vc.UserID.String()); err != nil {
		s.l.Error("failed to delete user sessions on password reset", zap.Error(err))
	}

	return nil
}

func (s *authSvc) GetRefreshTokenTTL() time.Duration {
	return s.cfg.RefreshTokenTTL
}

func (s *authSvc) generateJWTToken(user *models.User, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"role": user.Role,
		"exp":  time.Now().Add(ttl).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.SecretKey))
}
