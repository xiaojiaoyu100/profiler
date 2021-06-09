package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	option  *Option
	server  *http.Server
	running bool
	engine  *gin.Engine
	logger  *zap.Logger
	stop    chan struct{}
	done    chan struct{}
}

func New(setters ...Setter) (*HttpServer, error) {
	s := &HttpServer{}
	for _, setter := range setters {
		if err := setter(s); err != nil {
			return nil, err
		}
	}
	if s.option.Addr == "" {
		return nil, errors.New("no addr provided")
	}
	if s.engine == nil {
		return nil, errors.New("no engine provided")
	}
	srv := &http.Server{
		Addr:    s.option.Addr,
		Handler: s.engine,
	}
	s.server = srv
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

func WithEngine(engine *gin.Engine) Setter {
	return func(server *HttpServer) error {
		server.engine = engine
		return nil
	}
}

func (s *HttpServer) Running() bool {
	if s == nil {
		return false
	}
	return s.running
}

func (s *HttpServer) Close() {
	if s == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.option.ShutdownTimeout)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		s.logger.Fatal(fmt.Sprintf("listen and serve err"), zap.Error(err))
	}
	s.running = false
}

func (s *HttpServer) Run() {
	go func() {
		err := s.server.ListenAndServe()
		if err == nil {
			s.running = true
		}
	}()
}
