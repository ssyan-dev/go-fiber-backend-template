package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/models"
)

const (
	userCachePath = "users:"
	userCacheTTL  = 1 * time.Hour
)

type UserRedisRepository interface {
	SetUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, id string) (*models.User, error)
	DeleteUser(ctx context.Context, id string) error
}

type userRedisRepo struct {
	db *redis.Client
}

func NewUserRedisRepository(db *redis.Client) UserRedisRepository {
	return &userRedisRepo{db: db}
}

func (r *userRedisRepo) SetUser(ctx context.Context, user *models.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.db.Set(ctx, userCachePath+user.ID.String(), data, userCacheTTL).Err()
}

func (r *userRedisRepo) GetUser(ctx context.Context, id string) (*models.User, error) {
	data, err := r.db.Get(ctx, userCachePath+id).Bytes()
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRedisRepo) DeleteUser(ctx context.Context, id string) error {
	return r.db.Del(ctx, userCachePath+id).Err()
}
