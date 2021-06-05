package log

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DebugLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

var (
	atomLevel zap.AtomicLevel
	logger    *zap.Logger
	err       error
)

func NewLogger() (*zap.Logger, func(), error) {
	atomLevel = zap.NewAtomicLevel()
	atomLevel.SetLevel(zapcore.InfoLevel)

	cfg := zap.NewProductionConfig()
	cfg.Level = atomLevel
	logger, err = cfg.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("fail to create logger: %s", err)
	}
	f := func() {
		if err := logger.Sync(); err != nil {

		}
	}
	return logger, f, nil
}

func SetLevel(level int) error {
	switch level {
	case DebugLevel:
		setLevel(zap.DebugLevel)
	case InfoLevel:
		setLevel(zap.InfoLevel)
	case WarnLevel:
		setLevel(zap.WarnLevel)
	case ErrorLevel:
		setLevel(zap.ErrorLevel)
	default:
		return errors.New("incorrect log level")
	}
	return nil
}

func setLevel(level zapcore.Level) {
	atomLevel.SetLevel(level)
}
