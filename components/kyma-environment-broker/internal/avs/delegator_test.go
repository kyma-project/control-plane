package avs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func newTestParams(t *testing.T) (*Client, *MockAvsServer, Config, *InternalEvalAssistant, *ExternalEvalAssistant, *logrus.Logger) {
	// Given
	server := NewMockAvsServer(t)
	mockServer := FixMockAvsServer(server)
	avsCfg := Config{
		OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
		ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
	}
	client, err := NewClient(context.TODO(), avsCfg, logrus.New())
	assert.NoError(t, err)
	iea := NewInternalEvalAssistant(avsCfg)
	eea := NewExternalEvalAssistant(avsCfg)

	log := logrus.New()

	return client, server, avsCfg, iea, eea, log
}

type testParams struct {
	t               *testing.T
	client          *Client
	avsCfg          Config
	internalMonitor *BasicEvaluationCreateResponse
	externalMonitor *BasicEvaluationCreateResponse
}

func newDelOpsParams(params testParams) (*Delegator, internal.UpgradeKymaOperation) {
	operation := internal.UpgradeKymaOperation{}

	*params.internalMonitor, *params.externalMonitor = createMonitors(params.client)

	operation.Avs = internal.AvsLifecycleData{
		AvsEvaluationInternalId: params.internalMonitor.Id,
		AVSEvaluationExternalId: params.externalMonitor.Id,
	}

	operation.Avs.AvsExternalEvaluationStatus = internal.AvsEvaluationStatus{
		Current:  StatusActive,
		Original: StatusMaintenance,
	}

	operation.Avs.AvsInternalEvaluationStatus = internal.AvsEvaluationStatus{
		Current:  StatusActive,
		Original: StatusMaintenance,
	}

	ops := storage.NewMemoryStorage().Operations()
	delegator := NewDelegator(params.client, params.avsCfg, ops)
	err := ops.InsertUpgradeKymaOperation(operation)
	assert.NoError(params.t, err)
	assert.NotEqual(params.t, params.internalMonitor.Id, params.externalMonitor.Id)

	return delegator, operation
}

func setOpAvsStatus(op *internal.UpgradeKymaOperation, current string, original string) (string, string) {
	op.Avs.AvsInternalEvaluationStatus.Current = current
	op.Avs.AvsInternalEvaluationStatus.Original = original

	return current, original
}

func createMonitors(client *Client) (BasicEvaluationCreateResponse, BasicEvaluationCreateResponse) {
	internalEval, _ := client.CreateEvaluation(&BasicEvaluationCreateRequest{
		Name: "internal-monitor",
	})

	externalEval, _ := client.CreateEvaluation(&BasicEvaluationCreateRequest{
		Name: "external-monitor",
	})

	return *internalEval, *externalEval
}

// Since logic is the same for both internal and external monitors,
// we will only focus on internal monitor (easier to track).
func TestDelegator_SetStatus(t *testing.T) {
	// Given
	client, server, avsCfg, internalEA, _, logger := newTestParams(t)
	params := testParams{
		t:               t,
		client:          client,
		avsCfg:          avsCfg,
		internalMonitor: &BasicEvaluationCreateResponse{},
		externalMonitor: &BasicEvaluationCreateResponse{},
	}

	t.Run("set for valid fields (requested != current)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		requested := StatusInactive
		current, _ := setOpAvsStatus(&op, StatusActive, StatusMaintenance)

		// When
		op, d, err := delegator.SetStatus(logger, op, internalEA, requested)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, requested, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, current, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("set for valid fields (requested == current)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		requested := StatusInactive
		current, _ := setOpAvsStatus(&op, requested, StatusMaintenance)

		// When
		op, d, err := delegator.SetStatus(logger, op, internalEA, requested)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, current, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, StatusActive, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("set for partial fields (requested != current, original empty)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		requested := StatusInactive
		current, _ := setOpAvsStatus(&op, StatusActive, "")

		// When
		op, d, err := delegator.SetStatus(logger, op, internalEA, requested)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, requested, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, current, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("set for partial fields (requested == current, original empty)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		requested := StatusInactive
		current, _ := setOpAvsStatus(&op, requested, "")

		// When
		op, d, err := delegator.SetStatus(logger, op, internalEA, requested)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, current, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, StatusActive, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("set for partial fields (requested != avs current, current empty)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		requested := StatusInactive
		setOpAvsStatus(&op, "", StatusMaintenance)

		// When
		op, d, err := delegator.SetStatus(logger, op, internalEA, requested)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, requested, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, params.internalMonitor.Status, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("set for partial fields (requested == avs current, current empty)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		requested := params.internalMonitor.Status
		_, original := setOpAvsStatus(&op, "", StatusMaintenance)

		// When
		op, d, err := delegator.SetStatus(logger, op, internalEA, requested)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, params.internalMonitor.Status, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, original, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("set for empty fields", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		requested := StatusInactive
		_, _ = setOpAvsStatus(&op, "", "")

		// When
		op, d, err := delegator.SetStatus(logger, op, internalEA, requested)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, requested, internalEA.GetEvalStatus(op.Avs))
		// even though the avs Lifecycle data was initially empty,
		// during SetStatus call it was reloaded from avs backend api
		assert.Equal(t, params.internalMonitor.Status, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("set for invalid request", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		requested := "invalid"
		_, _ = setOpAvsStatus(&op, StatusActive, StatusMaintenance)

		// When
		op, _, err := delegator.SetStatus(logger, op, internalEA, requested)

		// Then
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), requested)
	})

	t.Run("disabled monitors", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)
		op.Avs.AvsEvaluationInternalId = 0

		version := op.Version

		// When
		op, _, err := delegator.SetStatus(logger, op, internalEA, StatusActive)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, version, op.Version)
	})

	t.Run("deleted monitors", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)
		op.Avs.AVSInternalEvaluationDeleted = true

		version := op.Version

		// When
		op, _, err := delegator.SetStatus(logger, op, internalEA, StatusActive)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, version, op.Version)
	})

	t.Run("reset for valid fields (current != original)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		current, original := setOpAvsStatus(&op, StatusActive, StatusMaintenance)

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, original, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, current, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("reset for valid fields (current == original)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		current, _ := setOpAvsStatus(&op, StatusInactive, StatusInactive)

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, current, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, StatusActive, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	// Since logic is the same for both internal and external monitors,
	// we will only focus on internal monitor (easier to track).
	t.Run("reset for empty fields", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		_, original := setOpAvsStatus(&op, "", "")

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		// restores current status from default
		assert.Equal(t, StatusActive, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, original, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	// Since logic is the same for both internal and external monitors,
	// we will only focus on internal monitor (easier to track).
	t.Run("reset for partial fields (original empty)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		current, _ := setOpAvsStatus(&op, StatusMaintenance, "")

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		// restores current status from default
		assert.Equal(t, StatusActive, internalEA.GetEvalStatus(op.Avs))
		assert.Equal(t, current, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	// Since logic is the same for both internal and external monitors,
	// we will only focus on internal monitor (easier to track).
	t.Run("reset for partial fields (current empty)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		instanceStatus := StatusInactive
		server.Evaluations.BasicEvals[params.internalMonitor.Id].Status = instanceStatus

		_, original := setOpAvsStatus(&op, "", StatusMaintenance)

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, original, internalEA.GetEvalStatus(op.Avs))
		// restores original status from current, which is restored from Avs
		assert.Equal(t, instanceStatus, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("reset from invalid fields (original invalid)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		current, _ := setOpAvsStatus(&op, StatusInactive, "invalid")

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		// restores current status from default
		assert.Equal(t, StatusActive, internalEA.GetEvalStatus(op.Avs))
		// restores original status from current, which is restored from Avs
		assert.Equal(t, current, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("reset from invalid fields (current invalid)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		_, original := setOpAvsStatus(&op, "invalidField", StatusMaintenance)

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, original, internalEA.GetEvalStatus(op.Avs))
		// restores original status from current, which is restored from Avs
		assert.Equal(t, params.internalMonitor.Status, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("reset from invalid avs fields (avs != current, original invalid)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		instanceStatus := "invalidField"
		server.Evaluations.BasicEvals[params.internalMonitor.Id].Status = instanceStatus

		current, _ := setOpAvsStatus(&op, StatusInactive, "invalid")

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		// restores current status from default
		assert.Equal(t, StatusActive, internalEA.GetEvalStatus(op.Avs))
		// restores original status from current, which is restored from Avs
		assert.Equal(t, current, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("reset from invalid avs fields (avs == current, original invalid)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		instanceStatus := "invalidField"
		server.Evaluations.BasicEvals[params.internalMonitor.Id].Status = instanceStatus

		current, _ := setOpAvsStatus(&op, StatusInactive, instanceStatus)

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		// restores current status from default
		assert.Equal(t, StatusActive, internalEA.GetEvalStatus(op.Avs))
		// restores original status from current, which is restored from Avs
		assert.Equal(t, current, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("reset from invalid avs fields (avs != current, current invalid)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		instanceStatus := "invalidField"
		server.Evaluations.BasicEvals[params.internalMonitor.Id].Status = instanceStatus

		_, original := setOpAvsStatus(&op, "invalid", StatusInactive)

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, original, internalEA.GetEvalStatus(op.Avs))
		// don't push junk data into original, leave as is
		assert.Equal(t, original, internalEA.GetOriginalEvalStatus(op.Avs))
	})

	t.Run("reset from invalid avs fields (avs == current, current invalid)", func(t *testing.T) {
		delegator, op := newDelOpsParams(params)

		instanceStatus := "invalidField"
		server.Evaluations.BasicEvals[params.internalMonitor.Id].Status = instanceStatus

		_, original := setOpAvsStatus(&op, instanceStatus, StatusInactive)

		// When
		op, d, err := delegator.ResetStatus(logger, op, internalEA)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), d)
		assert.Equal(t, original, internalEA.GetEvalStatus(op.Avs))
		// don't push junk data into original, leave as is
		assert.Equal(t, original, internalEA.GetOriginalEvalStatus(op.Avs))
	})
}
