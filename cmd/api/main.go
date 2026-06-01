package main

import (
	"context"
	"time"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/database"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/logger"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.InitConfig()
	if err != nil {
		panic(err.Error())
	}

	l := logger.NewLogger(cfg.App.Env)
	defer l.Sync()

	l.Info("starting backend",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	pg, err := database.NewPostgres(ctx, &cfg.Postgres)
	if err != nil {
		l.Fatal(err.Error())
	}
	defer pg.Close()
	l.Info("postgres connected!")

	rdb, err := database.NewRedis(ctx, &cfg.Redis)
	if err != nil {
		l.Fatal(err.Error())
	}
	defer rdb.Close()
	l.Info("redis connected!")
}
