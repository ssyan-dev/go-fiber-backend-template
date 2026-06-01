package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(isProduction bool) *zap.Logger {
	var config zap.Config

	if isProduction {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	l, err := config.Build()
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}

	return l
}
