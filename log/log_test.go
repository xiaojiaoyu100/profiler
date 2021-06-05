package log

import (
	"testing"

	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	logger, _, err := NewLogger()
	logger.Info("111", zap.Int("1", 1))
	logger.Warn("err:", zap.Error(err))
	SetLevel(WarnLevel)
	logger.Info("111", zap.Int("1", 1))
	logger.Warn("err:", zap.Error(err))
}
