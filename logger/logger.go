package logger

import (
	"errors"
	"log"

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
	l         *zap.SugaredLogger
	err       error
	//debugLevel = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
	//	return lvl >= zapcore.ErrorLevel
	//})
	//infoLevel = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
	//	return lvl >= zapcore.InfoLevel
	//})
)

func init() {
	atomLevel = zap.NewAtomicLevel()
	atomLevel.SetLevel(zapcore.InfoLevel)

	cfg := zap.NewProductionConfig()
	cfg.Level = atomLevel

	logger, err = cfg.Build()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	l = logger.Sugar()
}

func Info(format string, a ...interface{}) {
	l.Info(format, a)
}

func Warn(format string, a ...interface{}) {
	l.Warn(format, a)
}

func Error(format string, a ...interface{}) {
	l.Error(format, a)
}

func Debug(format string, a ...interface{}) {
	l.Debug(format, a)
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
