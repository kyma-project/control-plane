package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/log"
)

const (
	serverReadTimeout  = 10 * time.Second
	serverWriteTimeout = 5 * time.Second
	serverIdleTimeout  = 15 * time.Second
	serverStopTimeout  = 5 * time.Second
)

// Server represent a HTTP endpoint.
type Server struct {
	Addr   string
	Logger log.Logger

	http.ServeMux
}

// Start starts the HTTP server and shut it down when stop channel is closed.
func (svr *Server) Start(stop <-chan struct{}) (err error) {
	defer func() {
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			svr.Logger.Errorf("HTTP server stopped with error: %s", err)
		} else {
			svr.Logger.Info("HTTP server stopped")
		}
	}()

	s := http.Server{
		Addr:         svr.Addr,
		Handler:      &svr.ServeMux,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	go func() {
		// wait for the stop signal from the workgroup
		<-stop

		// stop the server within 5 seconds or cancel
		ctx, cancel := context.WithTimeout(context.Background(), serverStopTimeout)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			// we can ignore, it is always a cancel error
			svr.Logger.With("error", err).Debug("error shutting down server")
		}
	}()

	svr.Logger.With("address", s.Addr).Info("started HTTP server")

	// error is handle with the defer function
	return s.ListenAndServe()
}
