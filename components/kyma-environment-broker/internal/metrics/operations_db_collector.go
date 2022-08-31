package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// Retention is the default time and date for obtaining operations by the database query
// For performance reasons, it is not possible to query entire operations database table,
// so instead KEB queries the database for last 14 days worth of data and then for deltas
// during the ellapsed time
var Retention = 14 * 24 * time.Hour
var PollingInterval = 30 * time.Second

type operationsGetter interface {
	ListOperationsInTimeRange(from, to time.Time) ([]internal.Operation, error)
}

type opsMetricService struct {
	logger     logrus.FieldLogger
	operations *prometheus.GaugeVec
	lastUpdate time.Time
	db         operationsGetter
	cache      map[string]internal.Operation
}

// StartOpsMetricService creates service for exposing prometheus metrics for operations.
//
// This is intended as a replacement for OperationResultCollector to address shortcomings
// of the initial implementation - lack of consistency and non-aggregatable metric desing.
// The underlying data is fetched asynchronously from the KEB SQL database to provide
// consistency and the operation result state is exposed as a label instead of a value to
// enable common gauge aggregation.

// compass_keb_operation_result

func StartOpsMetricService(ctx context.Context, db operationsGetter, logger logrus.FieldLogger) {
	svc := &opsMetricService{
		db:         db,
		lastUpdate: time.Now().Add(-Retention),
		logger:     logger,
		cache:      make(map[string]internal.Operation),
		operations: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "operation_result",
			Help:      "Results of operations",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "type", "state"}),
	}
	go svc.run(ctx)
}

func (s *opsMetricService) setOperation(op internal.Operation, val float64) {
	labels := make(map[string]string)
	labels["operation_id"] = op.ID
	labels["instance_id"] = op.InstanceID
	labels["global_account_id"] = op.GlobalAccountID
	labels["plan_id"] = op.Plan
	labels["type"] = string(op.Type)
	labels["state"] = string(op.State)
	s.operations.With(labels).Set(val)
}

func (s *opsMetricService) updateOperation(op internal.Operation) {
	oldOp, found := s.cache[op.ID]
	if found {
		s.setOperation(oldOp, 0)
	}
	s.setOperation(op, 1)
	if op.State == domain.Failed || op.State == domain.Succeeded {
		delete(s.cache, op.ID)
	} else {
		s.cache[op.ID] = op
	}
}

func (s *opsMetricService) updateMetrics() (err error) {
	defer func() {
		if r := recover(); r != nil {
			// it's not desirable to panic metrics goroutine, instead it should return and log the error
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()
	now := time.Now()
	operations, err := s.db.ListOperationsInTimeRange(s.lastUpdate, now)
	if err != nil {
		return fmt.Errorf("failed to list operations: %v", err)
	}
	s.logger.Infof("updating operations metrics for: %v operations", len(operations))
	for _, op := range operations {
		s.updateOperation(op)
	}
	s.lastUpdate = now
	return nil
}

func (s *opsMetricService) run(ctx context.Context) {
	if err := s.updateMetrics(); err != nil {
		s.logger.Error("failed to update operations metrics", err)
	}
	ticker := time.NewTicker(PollingInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.updateMetrics(); err != nil {
				s.logger.Error("failed to update operations metrics", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
