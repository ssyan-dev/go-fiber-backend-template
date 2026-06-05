package repository

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
)

const (
	refreshTokenPath = "refresh_token:"
)

type AuthRedisRepository interface {
	SetRefreshToken(ctx context.Context, userID, refreshToken string) error
	GetRefreshToken(ctx context.Context, userID string) (string, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
}

type authRedisRepo struct {
	db  *redis.Client
	cfg *config.JWTConfig
}

func NewAuthRedisRepository(db *redis.Client, cfg *config.JWTConfig) AuthRedisRepository {
	return &authRedisRepo{
		db:  db,
		cfg: cfg,
	}
}

func (r *authRedisRepo) SetRefreshToken(ctx context.Context, userID, refreshToken string) error {
	return r.db.Set(ctx, refreshTokenPath+userID, refreshToken, r.cfg.RefreshTokenTTL).Err()
}

func (r *authRedisRepo) GetRefreshToken(ctx context.Context, userID string) (string, error) {
	return r.db.Get(ctx, refreshTokenPath+userID).Result()
}

func (r *authRedisRepo) DeleteRefreshToken(ctx context.Context, userID string) error {
	return r.db.Del(ctx, refreshTokenPath+userID).Err()
}
