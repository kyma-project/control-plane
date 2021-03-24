package utils

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/kyma-project/control-plane/components/metris/internal/log"
)

// ExitHandler returns when a signals is caught.
func ExitHandler(stop <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	select {
	case sig := <-c:
		log.Named("exit-handler").With("signal", sig).Info("shutting down")
	case <-stop:
	}

	return nil
}
