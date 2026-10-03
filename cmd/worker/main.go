package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/infra/mailer"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/infra/queue"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		panic(err.Error())
	}
	config.IsProduction = cfg.App.IsProduction()

	l := logger.NewLogger(cfg.App.IsProduction())
	defer l.Sync()

	l.Info("starting queue worker",
		zap.String("env", cfg.App.Env),
		zap.Int("concurrency", cfg.Queue.Concurrency),
	)

	ms, err := mailer.NewMailerService(&cfg.SMTP, l, mailer.DefaultTemplatesFS)
	if err != nil {
		l.Fatal("failed to initialize mailer", zap.Error(err))
	}

	processor := queue.NewRedisTaskProcessor(&cfg.Redis, &cfg.Queue, ms, l)
	if err := processor.Start(); err != nil {
		l.Fatal("failed to start queue processor", zap.Error(err))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	l.Info("shutting down queue worker...")
	processor.Shutdown()
	l.Info("queue worker stopped successfully")
}
