package server

import (
	"context"
	"errors"
	"log"
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
	option Option
	engine *gin.Engine
	logger *zap.Logger
}

func Setters(logger *zap.Logger) []Setter {
	return []Setter{
		WithLogger(logger),
	}
}

func New(setters ...Setter) (*HttpServer, error) {
	s := &HttpServer{}
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

func (s *HttpServer) Run() error {
	srv := &http.Server{
		Addr:    s.option.Addr,
		Handler: s.engine,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Printf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), s.option.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
	return nil
}
