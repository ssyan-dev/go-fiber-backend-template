package queue

import (
	"go.uber.org/zap"
)

type asynqLogger struct {
	sugar *zap.SugaredLogger
}

func newAsynqLogger(l *zap.Logger) *asynqLogger {
	return &asynqLogger{
		sugar: l.Named("asynq").Sugar(),
	}
}

func (l *asynqLogger) Debug(args ...any) {
	l.sugar.Debug(args...)
}

func (l *asynqLogger) Info(args ...any) {
	l.sugar.Info(args...)
}

func (l *asynqLogger) Warn(args ...any) {
	l.sugar.Warn(args...)
}

func (l *asynqLogger) Error(args ...any) {
	l.sugar.Error(args...)
}

func (l *asynqLogger) Fatal(args ...any) {
	l.sugar.Fatal(args...)
}
