package btpmgrcreds

import (
	"time"

	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

type Job struct {
	btpOperatorManager *Manager
	logs               *logrus.Logger
}

func NewJob(manager *Manager, logs *logrus.Logger) *Job {
	return &Job{
		btpOperatorManager: manager,
		logs:               logs,
	}
}

func (s *Job) Start(autoReconcileInterval int, jobReconciliationDelay time.Duration) {
	scheduler := gocron.NewScheduler(time.UTC)
	_, schedulerErr := scheduler.Every(autoReconcileInterval).Minutes().Do(func() {
		s.logs.Infof("runtime-reconciler: scheduled call starter at %s", time.Now())
		_, _, _, _, reconcileErr := s.btpOperatorManager.ReconcileAll(jobReconciliationDelay)
		if reconcileErr != nil {
			s.logs.Errorf("runtime-reconciler: scheduled call finished with error: %s", reconcileErr)
		} else {
			s.logs.Infof("runtime-reconciler: scheduled call finished with success at %s", time.Now().String())
		}
	})

	if schedulerErr != nil {
		s.logs.Errorf("runtime-reconciler: scheduler failure: %s", schedulerErr)
	}

	s.logs.Info("runtime-listener: start scheduler")
	scheduler.StartAsync()
}
