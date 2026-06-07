package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	sessionPrefix = "session:"
)

type SessionRedisRepository interface {
	SetSession(ctx context.Context, token string, userID string, ttl time.Duration) error
	GetSession(ctx context.Context, token string) (string, error)
	DeleteSession(ctx context.Context, token string) error
}

type sessionRedisRepo struct {
	db *redis.Client
}

func NewSessionRedisRepository(db *redis.Client) SessionRedisRepository {
	return &sessionRedisRepo{
		db: db,
	}
}

func (r *sessionRedisRepo) SetSession(ctx context.Context, token string, userID string, ttl time.Duration) error {
	return r.db.Set(ctx, sessionPrefix+token, userID, ttl).Err()
}

func (r *sessionRedisRepo) GetSession(ctx context.Context, token string) (string, error) {
	return r.db.Get(ctx, sessionPrefix+token).Result()
}

func (r *sessionRedisRepo) DeleteSession(ctx context.Context, token string) error {
	return r.db.Del(ctx, sessionPrefix+token).Err()
}
