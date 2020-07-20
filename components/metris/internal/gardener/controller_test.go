package gardener

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/log"

	"github.com/stretchr/testify/assert"
)

func TestController(t *testing.T) {
	defaultLogger := log.NewNoopLogger()
	clusterChannel := make(chan *Cluster, 1)

	ctrl, err := NewController(newFakeClient(t), "az", clusterChannel, defaultLogger)
	if err != nil {
		t.Errorf("NewController() error = %v", err)
	}

	stop := make(chan struct{})

	go func() {
		time.Sleep(2 * time.Second)
		close(stop)
	}()

	err = ctrl.Run(stop)

	assert.NoError(t, err)
}
