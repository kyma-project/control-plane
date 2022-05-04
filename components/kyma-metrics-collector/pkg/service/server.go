package service

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"context"

	"github.com/gorilla/mux"

	log "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
)

const (
	serverReadTimeout  = 10 * time.Second
	serverWriteTimeout = 5 * time.Second
	serverIdleTimeout  = 15 * time.Second
	serverStopTimeout  = 5 * time.Second
)

type Server struct {
	Addr   string
	Router *mux.Router
	Logger *zap.SugaredLogger
}

// Start starts the HTTP server and shut it down when stop channel is closed.
func (s *Server) Start() {

	server := http.Server{
		Addr:         s.Addr,
		Handler:      s.Router,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("start server")
		}
		s.namedLogger().Info("HTTP server stopped")
	}()
	s.namedLogger().Infof("started HTTP server at %s", s.Addr)

	<-done
	gracefulCtx, cancelShutdown := context.WithTimeout(context.Background(), serverStopTimeout)
	defer cancelShutdown()

	if err := server.Shutdown(gracefulCtx); err != nil {
		s.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
			Fatal("server is shutting down")
	}
	s.namedLogger().Infof("server gracefully stopped")
}

func (s *Server) namedLogger() *zap.SugaredLogger {
	return s.Logger.With("component", "kmc")
}
