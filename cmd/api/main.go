package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/database"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/logger"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/pkg/response"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.InitConfig()
	if err != nil {
		panic(err.Error())
	}

	l := logger.NewLogger(cfg.App.IsProduction())
	defer l.Sync()

	l.Info("starting backend",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	pg, err := database.NewPostgres(ctx, &cfg.Postgres)
	if err != nil {
		l.Fatal("failed to make postgres connection", zap.Error(err))
	}
	defer pg.Close()
	l.Info("postgres connected!")

	rdb, err := database.NewRedis(ctx, &cfg.Redis)
	if err != nil {
		l.Fatal("failed to make redis connection", zap.Error(err))
	}
	defer rdb.Close()
	l.Info("redis connected!")

	app := fiber.New()
	api := app.Group(cfg.App.GlobalPrefix)

	api.Get("/health", func(c fiber.Ctx) error {
		return response.Success(c, fiber.StatusOK, "server is healthy!", fiber.Map{
			"env": cfg.App.Env,
		})
	})

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	if err := app.Listen(addr); err != nil {
		l.Fatal("failed to run http server: %v", zap.Error(err))
	}
}
