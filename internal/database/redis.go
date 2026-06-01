package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
)

func NewRedis(ctx context.Context, cfg *config.RedisConfig) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       0,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
