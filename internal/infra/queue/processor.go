package queue

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/infra/mailer"
	"go.uber.org/zap"
)

type TaskProcessor interface {
	Start() error
	Shutdown()
}

type RedisTaskProcessor struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	mailer mailer.MailerService
	l      *zap.Logger
}

func NewRedisTaskProcessor(
	redisCfg *config.RedisConfig,
	queueCfg *config.QueueConfig,
	m mailer.MailerService,
	l *zap.Logger,
) *RedisTaskProcessor {
	server := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
			Password: redisCfg.Password,
		},
		asynq.Config{
			Concurrency: queueCfg.Concurrency,
			Queues: map[string]int{
				QueueCritical: queueCfg.CriticalWeight,
				QueueDefault:  queueCfg.DefaultWeight,
				QueueLow:      queueCfg.LowWeight,
			},
			ShutdownTimeout: queueCfg.ShutdownTimeout,
			Logger:          newAsynqLogger(l),
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				l.Error("asynq task execution failed",
					zap.String("type", task.Type()),
					zap.ByteString("payload", task.Payload()),
					zap.Error(err),
				)
			}),
		},
	)

	mux := asynq.NewServeMux()

	processor := &RedisTaskProcessor{
		server: server,
		mux:    mux,
		mailer: m,
		l:      l,
	}

	processor.registerHandlers()

	return processor
}

func (p *RedisTaskProcessor) registerHandlers() {
	p.mux.HandleFunc(TypeEmailVerification, p.ProcessEmailVerification)
	p.mux.HandleFunc(TypeEmailPasswordReset, p.ProcessEmailPasswordReset)
}

func (p *RedisTaskProcessor) Start() error {
	p.l.Info("starting asynq task processor")
	return p.server.Start(p.mux)
}

func (p *RedisTaskProcessor) Shutdown() {
	p.l.Info("shutting down asynq task processor")
	p.server.Shutdown()
}
