package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Setter func(server *HttpServer) error

type Option struct {
	Addr            string
	ShutdownTimeout time.Duration
}

type HttpServer struct {
	option *Option
	engine *gin.Engine
	logger *zap.Logger
	wait   chan struct{}
}

func New(setters ...Setter) (*HttpServer, error) {
	s := &HttpServer{
		wait: make(chan struct{}),
	}
	for _, setter := range setters {
		if err := setter(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func WithLogger(logger *zap.Logger) Setter {
	return func(server *HttpServer) error {
		server.logger = logger
		return nil
	}
}

func WithOption(option *Option) Setter {
	return func(server *HttpServer) error {
		server.option = option
		return nil
	}
}

func (s *HttpServer) Init() {
	go func() {
		s.wait <- struct{}{}
	}()
}

func (s *HttpServer) Run() error {
	<-s.wait
	srv := &http.Server{
		Addr:    s.option.Addr,
		Handler: s.engine,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			s.logger.Warn(fmt.Sprintf("listen and serve err"), zap.Error(err))
		}
	}()

	quit := make(chan os.Signal)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), s.option.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Fatal(fmt.Sprintf("listen and serve err"), zap.Error(err))
	}

	s.logger.Debug(fmt.Sprintf("server exiting..."))

	return nil
}
