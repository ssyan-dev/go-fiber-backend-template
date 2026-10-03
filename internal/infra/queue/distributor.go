package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/config"
	"github.com/ssyan-dev/go-fiber-backend-template/internal/infra/mailer"
	"go.uber.org/zap"
)

type TaskDistributor interface {
	DistributeEmailVerification(ctx context.Context, payload *EmailVerificationPayload, opts ...asynq.Option) error
	DistributeEmailPasswordReset(ctx context.Context, payload *EmailPasswordResetPayload, opts ...asynq.Option) error
	Close() error
}

type RedisTaskDistributor struct {
	client *asynq.Client
	cfg    *config.QueueConfig
	mailer mailer.MailerService
	l      *zap.Logger
}

func NewRedisTaskDistributor(
	redisCfg *config.RedisConfig,
	queueCfg *config.QueueConfig,
	m mailer.MailerService,
	l *zap.Logger,
) *RedisTaskDistributor {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password,
	})

	return &RedisTaskDistributor{
		client: client,
		cfg:    queueCfg,
		mailer: m,
		l:      l,
	}
}

func (d *RedisTaskDistributor) defaultOptions(queueName string) []asynq.Option {
	opts := []asynq.Option{
		asynq.Queue(queueName),
		asynq.Retention(24 * time.Hour),
	}
	if d.cfg.MaxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(d.cfg.MaxRetry))
	}
	if d.cfg.TaskTimeout > 0 {
		opts = append(opts, asynq.Timeout(d.cfg.TaskTimeout))
	}
	return opts
}

func (d *RedisTaskDistributor) DistributeEmailVerification(
	ctx context.Context,
	payload *EmailVerificationPayload,
	opts ...asynq.Option,
) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("queue: marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeEmailVerification, jsonPayload)
	taskOpts := append(d.defaultOptions(QueueDefault), opts...)

	_, err = d.client.EnqueueContext(ctx, task, taskOpts...)
	if err != nil {
		if d.cfg.FallbackSyncOnFail && d.mailer != nil {
			d.l.Warn("redis queue enqueue failed, falling back to sync email sending",
				zap.String("task", TypeEmailVerification),
				zap.String("email", payload.Email),
				zap.Error(err),
			)
			return d.mailer.SendTemplate(ctx, payload.Email, "Verify your email", "email-verification.html", payload)
		}
		return fmt.Errorf("queue: enqueue verification task: %w", err)
	}

	d.l.Debug("enqueued email verification task",
		zap.String("task", TypeEmailVerification),
		zap.String("email", payload.Email),
		zap.String("queue", QueueDefault),
	)

	return nil
}

func (d *RedisTaskDistributor) DistributeEmailPasswordReset(
	ctx context.Context,
	payload *EmailPasswordResetPayload,
	opts ...asynq.Option,
) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("queue: marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeEmailPasswordReset, jsonPayload)
	taskOpts := append(d.defaultOptions(QueueCritical), opts...)

	_, err = d.client.EnqueueContext(ctx, task, taskOpts...)
	if err != nil {
		if d.cfg.FallbackSyncOnFail && d.mailer != nil {
			d.l.Warn("redis queue enqueue failed, falling back to sync email sending",
				zap.String("task", TypeEmailPasswordReset),
				zap.String("email", payload.Email),
				zap.Error(err),
			)
			return d.mailer.SendTemplate(ctx, payload.Email, "Reset your password", "password-reset.html", payload)
		}
		return fmt.Errorf("queue: enqueue password reset task: %w", err)
	}

	d.l.Debug("enqueued password reset task",
		zap.String("task", TypeEmailPasswordReset),
		zap.String("email", payload.Email),
		zap.String("queue", QueueCritical),
	)

	return nil
}

func (d *RedisTaskDistributor) Close() error {
	return d.client.Close()
}
