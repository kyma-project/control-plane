package service

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"context"

	"github.com/gorilla/mux"

	"github.com/sirupsen/logrus"
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
	Logger *logrus.Logger
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
			s.Logger.Fatalf("failed to start server, listen: %s\n", err)
		}
		s.Logger.Info("HTTP server stopped")
	}()
	s.Logger.Infof("started HTTP server at %s", s.Addr)

	<-done
	gracefulCtx, cancelShutdown := context.WithTimeout(context.Background(), serverStopTimeout)
	defer cancelShutdown()

	if err := server.Shutdown(gracefulCtx); err != nil {
		s.Logger.Fatalf("shutdown error: %v\n", err)
	}
	s.Logger.Infof("gracefully stopped\n")
}
