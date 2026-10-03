package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
)

type LimiterOptions struct {
	KeyPrefix  string
	Max        int
	Expiration time.Duration
	Message    string
	Storage    fiber.Storage
	Next       func(fiber.Ctx) bool
}

func NewLimiter(opts LimiterOptions) fiber.Handler {
	msg := opts.Message
	if msg == "" {
		msg = "too many requests, please try again later"
	}

	prefix := opts.KeyPrefix
	if prefix == "" {
		prefix = "limiter:"
	}

	return limiter.New(limiter.Config{
		Max:        opts.Max,
		Expiration: opts.Expiration,
		Storage:    opts.Storage,
		Next:       opts.Next,
		KeyGenerator: func(c fiber.Ctx) string {
			return prefix + c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests, msg, nil)
		},
	})
}

func AuthLimiter(cfg *config.LimiterConfig, storage fiber.Storage) fiber.Handler {
	max := 10
	expiration := 1 * time.Minute
	if cfg != nil {
		if cfg.AuthMax > 0 {
			max = cfg.AuthMax
		}
		if cfg.AuthExpiration > 0 {
			expiration = cfg.AuthExpiration
		}
	}

	return NewLimiter(LimiterOptions{
		KeyPrefix:  "limiter:auth:",
		Max:        max,
		Expiration: expiration,
		Storage:    storage,
		Message:    "too many authentication attempts, please try again in a minute",
	})
}

func GlobalLimiter(cfg *config.LimiterConfig, storage fiber.Storage) fiber.Handler {
	max := 120
	expiration := 1 * time.Minute
	if cfg != nil {
		if cfg.GlobalMax > 0 {
			max = cfg.GlobalMax
		}
		if cfg.GlobalExpiration > 0 {
			expiration = cfg.GlobalExpiration
		}
	}

	return NewLimiter(LimiterOptions{
		KeyPrefix:  "limiter:global:",
		Max:        max,
		Expiration: expiration,
		Storage:    storage,
		Message:    "rate limit exceeded, please slow down",
		Next: func(c fiber.Ctx) bool {
			path := c.Path()
			return strings.HasSuffix(path, "/health") || strings.Contains(path, "/docs")
		},
	})
}
