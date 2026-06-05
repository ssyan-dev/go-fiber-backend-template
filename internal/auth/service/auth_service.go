package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/auth/repository"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*models.User, error)
	Login(ctx context.Context, email, password string) (string, string, error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
	GetRefreshTokenTTL() time.Duration
}

type authSvc struct {
	repo      repository.AuthRepository
	redisRepo repository.AuthRedisRepository
	cfg       *config.JWTConfig
	l         *zap.Logger
}

func NewAuthService(repo repository.AuthRepository, redisRepo repository.AuthRedisRepository, cfg *config.JWTConfig, l *zap.Logger) AuthService {
	return &authSvc{
		repo:      repo,
		redisRepo: redisRepo,
		cfg:       cfg,
		l:         l,
	}
}

func (s *authSvc) Register(ctx context.Context, email, password string) (*models.User, error) {
	existing, _ := s.repo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	hashStr := string(hash)
	user := &models.User{
		Email:        email,
		PasswordHash: &hashStr,
		Role:         models.RoleDefault,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.l.Error("failed to create user", zap.Error(err))
		return nil, err
	}

	return user, nil
}

func (s *authSvc) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
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

	err = s.redisRepo.SetRefreshToken(ctx, user.ID.String(), refreshToken)
	if err != nil {
		s.l.Error("failed to create set refresh token to redis", zap.Error(err))
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *authSvc) Logout(ctx context.Context, refreshToken string) error {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.SecretKey), nil
	})

	if err != nil || !token.Valid {
		return nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil
	}

	return s.redisRepo.DeleteRefreshToken(ctx, userID)
}

func (s *authSvc) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
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

	storedToken, err := s.redisRepo.GetRefreshToken(ctx, userID)
	if err != nil || storedToken != refreshToken {
		return "", "", ErrInvalidToken
	}

	user, err := s.repo.GetByID(ctx, userID)
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

	err = s.redisRepo.SetRefreshToken(ctx, userID, newRefreshToken)
	if err != nil {
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
