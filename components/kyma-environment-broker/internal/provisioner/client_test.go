package provisioner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	schema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"

	"github.com/99designs/gqlgen/handler"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const (
	testAccountID    = "4346c639-32f8-4947-ae95-73bb8efad209"
	testSubAccountID = "42d45043-d0fb-4077-9de0-d7f781949bce"

	provisionRuntimeID            = "4e268c0f-d053-4ab7-b167-6dbc0a0e09a6"
	provisionRuntimeOperationID   = "c89f7862-0ef9-4d4e-bc82-afbc5ac98b8d"
	upgradeRuntimeOperationID     = "74f47e0a-9a76-4336-9974-70705500a981"
	deprovisionRuntimeOperationID = "f9f7b734-7538-419c-8ac1-37060c60531a"
)

var (
	testKubernetesVersion   = "1.17.16"
	testMachineImage        = "gardenlinux"
	testMachineImageVersion = "184.0.0"
	testAutoScalerMin       = 2
	testAutoScalerMax       = 4
	testMaxSurge            = 4
	testMaxUnavailable      = 1
)

func TestClient_ProvisionRuntime(t *testing.T) {
	t.Run("should trigger provisioning", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)

		// When
		status, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, ptr.String(provisionRuntimeOperationID), status.ID)
		assert.Equal(t, schema.OperationStateInProgress, status.State)
		assert.Equal(t, ptr.String(provisionRuntimeID), status.RuntimeID)

		assert.Equal(t, "test", tr.getRuntime().name)
	})

	t.Run("should trigger provisioning without dns config", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)

		// When
		status, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInputWithoutDnsConfig())

		// Then
		assert.NoError(t, err)
		assert.Equal(t, ptr.String(provisionRuntimeOperationID), status.ID)
		assert.Equal(t, schema.OperationStateInProgress, status.State)
		assert.Equal(t, ptr.String(provisionRuntimeID), status.RuntimeID)

		assert.Equal(t, "test", tr.getRuntime().name)
	})

	t.Run("provisioner should return error", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}, failed: true}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)

		// When
		status, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())

		// Then
		assert.Error(t, err)
		assert.Empty(t, status)

		assert.Equal(t, "", tr.getRuntime().name)
	})
}

func TestClient_DeprovisionRuntime(t *testing.T) {
	t.Run("should trigger deprovisioning", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		operation, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		// When
		operationId, err := client.DeprovisionRuntime(testAccountID, *operation.RuntimeID)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, deprovisionRuntimeOperationID, operationId)

		assert.Empty(t, tr.getRuntime().runtimeID)
	})

	t.Run("provisioner should return error", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		operation, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		tr.failed = true

		// When
		operationId, err := client.DeprovisionRuntime(testAccountID, *operation.RuntimeID)

		// Then
		assert.Error(t, err)
		assert.Equal(t, "", operationId)

		assert.Equal(t, provisionRuntimeID, tr.getRuntime().runtimeID)
	})
}

func TestClient_UpgradeRuntime(t *testing.T) {
	t.Run("should trigger upgrade", func(t *testing.T) {
		// given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		operation, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		// when
		status, err := client.UpgradeRuntime(testAccountID, *operation.RuntimeID, fixUpgradeRuntimeInput("1.14.0"))

		// then
		assert.NoError(t, err)
		assert.Equal(t, ptr.String(upgradeRuntimeOperationID), status.ID)
		assert.Equal(t, schema.OperationStateInProgress, status.State)
		assert.Equal(t, schema.OperationTypeUpgrade, status.Operation)
		assert.Equal(t, ptr.String(provisionRuntimeID), status.RuntimeID)
	})

	t.Run("provisioner should return error", func(t *testing.T) {
		// given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		operation, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		tr.failed = true

		// when
		status, err := client.UpgradeRuntime(testAccountID, *operation.RuntimeID, fixUpgradeRuntimeInput("1.14.0"))

		// Then
		assert.Error(t, err)
		assert.Empty(t, status)

		assert.Equal(t, "", tr.getRuntime().upgradeOperationID)
	})
}

func TestClient_UpgradeShoot(t *testing.T) {
	t.Run("should trigger shoot upgrade", func(t *testing.T) {
		// given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		operation, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		// when
		status, err := client.UpgradeShoot(testAccountID, *operation.RuntimeID, fixUpgradeShootInput())

		// then
		assert.NoError(t, err)
		assert.Equal(t, ptr.String(upgradeRuntimeOperationID), status.ID)
		assert.Equal(t, schema.OperationStateInProgress, status.State)
		assert.Equal(t, schema.OperationTypeUpgradeShoot, status.Operation)
		assert.Equal(t, ptr.String(provisionRuntimeID), status.RuntimeID)
	})

	t.Run("provisioner should return error", func(t *testing.T) {
		// given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		operation, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		tr.failed = true

		// when
		status, err := client.UpgradeShoot(testAccountID, *operation.RuntimeID, fixUpgradeShootInput())

		// Then
		assert.Error(t, err)
		assert.Empty(t, status)

		assert.Equal(t, "", tr.getRuntime().upgradeOperationID)
	})
}

func TestClient_ReconnectRuntimeAgent(t *testing.T) {
	t.Run("should reconnect runtime agent", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		operation, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		// When
		operationId, err := client.ReconnectRuntimeAgent(testAccountID, *operation.RuntimeID)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, provisionRuntimeOperationID, operationId)
	})

	t.Run("provisioner should return error", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		operation, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		tr.failed = true

		// When
		operationId, err := client.ReconnectRuntimeAgent(testAccountID, *operation.RuntimeID)

		// Then
		assert.Error(t, err)
		assert.Equal(t, "", operationId)
	})

	t.Run("provisioner returns bad request code error", func(t *testing.T) {
		server := fixHTTPMockServer(`{
			  "errors": [
				{
				  "message": "tenant header is empty",
				  "path": [
					"runtimeStatus"
				  ],
				  "extensions": {
					"error_code": 400,
					"error_reason": "Object not found",
					"error_component": "compass director"
				  }
				}
			  ],
			  "data": {
				"runtimeStatus": null
			  }
			}`)
		defer server.Close()

		client := NewProvisionerClient(server.URL, false)

		// when
		_, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		lastErr := kebError.ReasonForError(err)

		// Then
		assert.Error(t, err)
		assert.False(t, kebError.IsTemporaryError(err))
		assert.Equal(t, kebError.ErrReason("Object not found"), lastErr.Reason())
		assert.Equal(t, kebError.ErrComponent("compass director"), lastErr.Component())
	})

	t.Run("provisioner returns temporary code error", func(t *testing.T) {
		server := fixHTTPMockServer(`{
			  "errors": [
				{
				  "message": "tenant header is empty",
				  "path": [
					"runtimeStatus"
				  ],
				  "extensions": {
					"error_code": 500
				  }
				}
			  ],
			  "data": {
				"runtimeStatus": null
			  }
			}`)
		defer server.Close()

		client := NewProvisionerClient(server.URL, false)

		// when
		_, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())

		// Then
		assert.Error(t, err)
		assert.True(t, kebError.IsTemporaryError(err))
	})

	t.Run("network error", func(t *testing.T) {
		client := NewProvisionerClient("http://not-existing", false)

		// when
		_, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())

		// Then
		assert.Error(t, err)
		assert.True(t, kebError.IsTemporaryError(err))
	})
}

func TestClient_RuntimeOperationStatus(t *testing.T) {
	t.Run("should return runtime operation status", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		_, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		// When
		status, err := client.RuntimeOperationStatus(testAccountID, provisionRuntimeID)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, ptr.String(provisionRuntimeID), status.RuntimeID)
		assert.Equal(t, ptr.String(provisionRuntimeOperationID), status.ID)
		assert.Equal(t, schema.OperationStateInProgress, status.State)
		assert.Equal(t, schema.OperationTypeProvision, status.Operation)
	})

	t.Run("provisioner should return error", func(t *testing.T) {
		// Given
		tr := &testResolver{t: t, runtime: &testRuntime{}}
		testServer := fixHTTPServer(tr)
		defer testServer.Close()

		client := NewProvisionerClient(testServer.URL, false)
		_, err := client.ProvisionRuntime(testAccountID, testSubAccountID, fixProvisionRuntimeInput())
		assert.NoError(t, err)

		tr.failed = true

		// When
		status, err := client.RuntimeOperationStatus(testAccountID, provisionRuntimeID)

		// Then
		assert.Error(t, err)
		assert.Empty(t, status)
	})
}

type testRuntime struct {
	tenant                 string
	clientID               string
	name                   string
	runtimeID              string
	provisionOperationID   string
	upgradeOperationID     string
	deprovisionOperationID string
}

type testResolver struct {
	t       *testing.T
	runtime *testRuntime
	failed  bool
}

type httpHandler struct {
	r string
}

func (h httpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte(h.r))
}

func fixHandler(resp string) http.Handler {
	return httpHandler{
		r: resp,
	}
}

func fixHTTPMockServer(resp string) *httptest.Server {
	return httptest.NewServer(fixHandler(resp))
}

func fixHTTPServer(tr *testResolver) *httptest.Server {
	r := mux.NewRouter()

	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accountID := r.Header.Get(accountIDKey)
			subAccountID := r.Header.Get(subAccountIDKey)
			if accountID != testAccountID {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			tr.runtime.tenant = accountID
			tr.runtime.clientID = subAccountID

			h.ServeHTTP(w, r)
		})
	})
	r.HandleFunc("/", handler.GraphQL(schema.NewExecutableSchema(schema.Config{Resolvers: tr})))

	return httptest.NewServer(r)
}

func (tr testResolver) Mutation() schema.MutationResolver {
	tr.t.Log("Mutation TestResolver")
	return &testMutationResolver{t: tr.t, runtime: tr.runtime, failed: tr.failed}
}

func (tr testResolver) Query() schema.QueryResolver {
	tr.t.Log("Query TestResolver")
	return &testQueryResolver{t: tr.t, runtime: tr.runtime, failed: tr.failed}
}

func (tr testResolver) getRuntime() *testRuntime {
	return tr.runtime
}

type testMutationResolver struct {
	t       *testing.T
	runtime *testRuntime
	failed  bool
}

func (tmr *testMutationResolver) ProvisionRuntime(_ context.Context, config schema.ProvisionRuntimeInput) (*schema.OperationStatus, error) {
	tmr.t.Log("ProvisionRuntime testMutationResolver")

	if tmr.failed {
		return nil, fmt.Errorf("provision runtime failed for %s", config.RuntimeInput.Name)
	}

	tmr.runtime.name = config.RuntimeInput.Name
	tmr.runtime.runtimeID = provisionRuntimeID
	tmr.runtime.provisionOperationID = provisionRuntimeOperationID

	return &schema.OperationStatus{
		ID:        ptr.String(tmr.runtime.provisionOperationID),
		State:     schema.OperationStateInProgress,
		RuntimeID: ptr.String(tmr.runtime.runtimeID),
	}, nil
}

func (tmr testMutationResolver) UpgradeRuntime(_ context.Context, id string, config schema.UpgradeRuntimeInput) (*schema.OperationStatus, error) {
	tmr.t.Log("UpgradeTuntime testMutationResolver")

	if tmr.failed {
		return nil, fmt.Errorf("upgrade runtime failed for version %s", id)
	}

	if tmr.runtime.runtimeID == id {
		tmr.runtime.upgradeOperationID = upgradeRuntimeOperationID
	}

	return &schema.OperationStatus{
		ID:        ptr.String(tmr.runtime.upgradeOperationID),
		State:     schema.OperationStateInProgress,
		Operation: schema.OperationTypeUpgrade,
		RuntimeID: ptr.String(tmr.runtime.runtimeID),
	}, nil
}

func (tmr testMutationResolver) DeprovisionRuntime(_ context.Context, id string) (string, error) {
	tmr.t.Log("DeprovisionRuntime testMutationResolver")

	if tmr.failed {
		return "", fmt.Errorf("deprovision failed for %s", id)
	}

	if tmr.runtime.runtimeID == id {
		tmr.runtime.runtimeID = ""
		tmr.runtime.name = ""
		tmr.runtime.deprovisionOperationID = deprovisionRuntimeOperationID
	}

	return tmr.runtime.deprovisionOperationID, nil
}

func (tmr testMutationResolver) HibernateRuntime(ctx context.Context, id string) (*schema.OperationStatus, error) {
	return nil, errors.New("not implemented")
}

func (tmr testMutationResolver) RollBackUpgradeOperation(_ context.Context, id string) (*schema.RuntimeStatus, error) {
	return nil, nil
}

func (tmr testMutationResolver) ReconnectRuntimeAgent(_ context.Context, id string) (string, error) {
	tmr.t.Log("ReconnectRuntimeAgent testMutationResolver")

	if tmr.failed {
		return "", fmt.Errorf("reconnect runtime agent failed for %s", id)
	}

	if tmr.runtime.runtimeID == id {
		return tmr.runtime.provisionOperationID, nil
	}

	return "", nil
}

func (tmr testMutationResolver) UpgradeShoot(_ context.Context, id string, config schema.UpgradeShootInput) (*schema.OperationStatus, error) {
	tmr.t.Log("UpgradeShoot testMutationResolver")

	if tmr.failed {
		return nil, fmt.Errorf("upgrade runtime failed for version %s", id)
	}

	if tmr.runtime.runtimeID == id {
		tmr.runtime.upgradeOperationID = upgradeRuntimeOperationID
	}

	return &schema.OperationStatus{
		ID:        ptr.String(tmr.runtime.upgradeOperationID),
		State:     schema.OperationStateInProgress,
		Operation: schema.OperationTypeUpgradeShoot,
		RuntimeID: ptr.String(tmr.runtime.runtimeID),
	}, nil
}

type testQueryResolver struct {
	t       *testing.T
	runtime *testRuntime
	failed  bool
}

func (tqr testQueryResolver) RuntimeStatus(_ context.Context, id string) (*schema.RuntimeStatus, error) {
	return nil, nil
}

func (tqr testQueryResolver) RuntimeOperationStatus(_ context.Context, id string) (*schema.OperationStatus, error) {
	tqr.t.Log("RuntimeOperationStatus - testQueryResolver")

	if tqr.failed {
		return nil, fmt.Errorf("query about runtime operation status failed for %s", id)
	}

	if tqr.runtime.runtimeID == id {
		return &schema.OperationStatus{
			ID:        ptr.String(tqr.runtime.provisionOperationID),
			Operation: schema.OperationTypeProvision,
			State:     schema.OperationStateInProgress,
			Message:   ptr.String("test message"),
			RuntimeID: ptr.String(tqr.runtime.runtimeID),
		}, nil
	}

	return nil, nil
}

func fixProvisionRuntimeInput() schema.ProvisionRuntimeInput {
	return schema.ProvisionRuntimeInput{
		RuntimeInput: &schema.RuntimeInput{
			Name:        "test",
			Description: nil,
			Labels:      nil,
		},
		ClusterConfig: &schema.ClusterConfigInput{
			GardenerConfig: &schema.GardenerConfigInput{
				ProviderSpecificConfig: &schema.ProviderSpecificInput{},
				Name:                   "abcd",
				VolumeSizeGb:           ptr.Integer(50),
				DNSConfig: &schema.DNSConfigInput{
					Domain: "test.devtest.kyma.ondemand.com",
					Providers: []*schema.DNSProviderInput{
						&schema.DNSProviderInput{
							DomainsInclude: []string{"devtest.kyma.ondemand.com"},
							Primary:        true,
							SecretName:     "efg",
							Type:           "route53_type_test",
						},
					},
				},
			},
		},
		KymaConfig: &schema.KymaConfigInput{
			Components: []*schema.ComponentConfigurationInput{
				{
					Component: "test",
				},
			},
		},
	}
}

func fixProvisionRuntimeInputWithoutDnsConfig() schema.ProvisionRuntimeInput {
	return schema.ProvisionRuntimeInput{
		RuntimeInput: &schema.RuntimeInput{
			Name:        "test",
			Description: nil,
			Labels:      nil,
		},
		ClusterConfig: &schema.ClusterConfigInput{
			GardenerConfig: &schema.GardenerConfigInput{
				ProviderSpecificConfig: &schema.ProviderSpecificInput{},
				Name:                   "abcd",
				VolumeSizeGb:           ptr.Integer(50),
			},
		},
		KymaConfig: &schema.KymaConfigInput{
			Components: []*schema.ComponentConfigurationInput{
				{
					Component: "test",
				},
			},
		},
	}
}

func fixUpgradeRuntimeInput(kymaVersion string) schema.UpgradeRuntimeInput {
	return schema.UpgradeRuntimeInput{KymaConfig: &schema.KymaConfigInput{
		Version: kymaVersion,
		Configuration: []*schema.ConfigEntryInput{
			{
				Key:   "a.config.key",
				Value: "a.config.value",
			},
		},
		Components: []*schema.ComponentConfigurationInput{
			{
				Component: "test-component",
				Namespace: "test-namespace",
			},
		},
	}}
}

func fixUpgradeShootInput() schema.UpgradeShootInput {
	return schema.UpgradeShootInput{
		GardenerConfig: &schema.GardenerUpgradeInput{
			KubernetesVersion:   &testKubernetesVersion,
			MachineImage:        &testMachineImage,
			MachineImageVersion: &testMachineImageVersion,
			AutoScalerMin:       &testAutoScalerMin,
			AutoScalerMax:       &testAutoScalerMax,
			MaxSurge:            &testMaxSurge,
			MaxUnavailable:      &testMaxUnavailable,
		},
	}
}
