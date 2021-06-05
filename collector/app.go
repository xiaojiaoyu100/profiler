package collector

import (
	"github.com/xiaojiaoyu100/profiler/collector/build"
	"github.com/xiaojiaoyu100/profiler/collector/server"
	"go.uber.org/zap"
)

type App struct {
	buildOption  *build.Option
	logger       *zap.Logger
	httpServer   *server.HttpServer
}

type Setter func(app *App) error

func New(setters ...Setter) (*App, error) {
	a := &App{}
	for _, setter := range setters {
		if err := setter(a); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func WithHttpServer(server *server.HttpServer) Setter {
	return func(app *App) error {
		app.httpServer = server
		return nil
	}
}

func WithLogger(logger *zap.Logger) Setter {
	return func(app *App) error {
		app.logger = logger
		return nil
	}
}

func (a *App) Run() error {
	if err := a.httpServer.Run(); err != nil {
		return err
	}
	return nil
}
