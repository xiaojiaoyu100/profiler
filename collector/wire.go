// +build wireinject

package collector

import (
	"github.com/google/wire"
	"github.com/xiaojiaoyu100/profiler/collector/server"
	"github.com/xiaojiaoyu100/profiler/log"
	"go.uber.org/zap"
)

func Setters(logger *zap.Logger, server *server.HttpServer) []Setter {
	return []Setter{
		WithLogger(logger),
		WithHttpServer(server),
	}
}

func InitializeApp() (*App, func(), error) {
	panic(
		wire.Build(
			log.NewLogger,
			server.Setters,
			server.New,
			Setters,
			New,
		))
}
