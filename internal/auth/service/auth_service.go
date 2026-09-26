package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	sessionService "github.com/ssyan-dev/go-fiber-backend-template/internal/sessions/service"
	userService "github.com/ssyan-dev/go-fiber-backend-template/internal/user/service"
	vcService "github.com/ssyan-dev/go-fiber-backend-template/internal/verification_codes/service"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists   = userService.ErrUserAlreadyExists
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidLoginType    = errors.New("your account has no password. use oauth")
	ErrInvalidToken        = errors.New("invalid token")
	ErrEmailMustBeVerified = errors.New("please verify your email")
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*models.User, error)
	Login(ctx context.Context, email, password, ip, userAgent string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken, ip, userAgent string) (string, string, error)
	GetRefreshTokenTTL() time.Duration
}

type authSvc struct {
	userSvc    userService.UserService
	sessionSvc sessionService.SessionService
	vcSvc      vcService.VerificationCodeService
	cfg        *config.JWTConfig
	authCfg    *config.AuthConfig
	l          *zap.Logger
}

func NewAuthService(
	userSvc userService.UserService,
	sessionSvc sessionService.SessionService,
	vcSvc vcService.VerificationCodeService,
	cfg *config.JWTConfig,
	authCfg *config.AuthConfig,
	l *zap.Logger,
) AuthService {
	return &authSvc{
		userSvc:    userSvc,
		sessionSvc: sessionSvc,
		vcSvc:      vcSvc,
		cfg:        cfg,
		authCfg:    authCfg,
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

	if err := s.vcSvc.SendEmailVerificationCode(ctx, user.ID.String(), user.Email); err != nil {
		s.l.Error("failed to send verification email", zap.Error(err))
	}

	return user, nil
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
