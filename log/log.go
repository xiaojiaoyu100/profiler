package log

import (
	"fmt"
	"go.uber.org/zap"
)

func NewLogger() (*zap.Logger, func(), error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, nil, fmt.Errorf("fail to create logger: %s", err)
	}
	f := func() {
		if err := logger.Sync(); err != nil {

		}
	}
	return logger, f, nil
}
