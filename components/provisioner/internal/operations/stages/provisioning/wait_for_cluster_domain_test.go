package provisioning

import (
	"context"
	"errors"
	directormock "github.com/kyma-project/control-plane/components/provisioner/internal/director/mocks"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/stretchr/testify/mock"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-incubator/compass/components/director/pkg/graphql"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	gardenerMocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kubeconfig = `apiVersion: v1
clusters:
- cluster:
    server: https://192.168.64.4:8443
  name: minikube
contexts:
- context:
    cluster: minikube
    user: minikube
  name: minikube
current-context: minikube
kind: Config
preferences: {}
users:
- name: minikube
  user:
    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURBRENDQWVpZ0F3SUJBZ0lCQWpBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwdGFXNXAKYTNWaVpVTkJNQjRYRFRFNU1URXhOekE0TXpBek1sb1hEVEl3TVRFeE56QTRNekF6TWxvd01URVhNQlVHQTFVRQpDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGakFVQmdOVkJBTVREVzFwYm1scmRXSmxMWFZ6WlhJd2dnRWlNQTBHCkNTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFDNmY2SjZneElvL2cyMHArNWhybklUaUd5SDh0VW0KWGl1OElaK09UKyt0amd1OXRneXFnbnNsL0dDT1Q3TFo4ejdOVCttTEdKL2RLRFdBV3dvbE5WTDhxMzJIQlpyNwpDaU5hK3BBcWtYR0MzNlQ2NEQyRjl4TEtpVVpuQUVNaFhWOW1oeWVCempscTh1NnBjT1NrY3lJWHRtdU9UQUVXCmErWlp5UlhOY3BoYjJ0NXFUcWZoSDhDNUVDNUIrSm4rS0tXQ2Y1Nm5KZGJQaWduRXh4SFlaMm9TUEc1aXpkbkcKZDRad2d0dTA3NGttaFNtNXQzbjgyNmovK29tL25VeWdBQ24yNmR1K21aZzRPcWdjbUMrdnBYdUEyRm52bk5LLwo5NWErNEI3cGtNTER1bHlmUTMxcjlFcStwdHBkNUR1WWpldVpjS1Bxd3ZVcFUzWVFTRUxVUzBrUkFnTUJBQUdqClB6QTlNQTRHQTFVZER3RUIvd1FFQXdJRm9EQWRCZ05WSFNVRUZqQVVCZ2dyQmdFRkJRY0RBUVlJS3dZQkJRVUgKQXdJd0RBWURWUjBUQVFIL0JBSXdBREFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBQ3JnbExWemhmemZ2aFNvUgowdWNpNndBZDF6LzA3bW52MDRUNmQyTkpjRG80Uzgwa0o4VUJtRzdmZE5qMlJEaWRFbHRKRU1kdDZGa1E1TklOCk84L1hJdENiU0ZWYzRWQ1NNSUdPcnNFOXJDajVwb24vN3JxV3dCbllqYStlbUVYOVpJelEvekJGU3JhcWhud3AKTkc1SmN6bUg5ODRWQUhGZEMvZWU0Z2szTnVoV25rMTZZLzNDTTFsRkxlVC9Cbmk2K1M1UFZoQ0x3VEdmdEpTZgorMERzbzVXVnFud2NPd3A3THl2K3h0VGtnVmdSRU5RdTByU2lWL1F2UkNPMy9DWXdwRTVIRFpjalM5N0I4MW0yCmVScVBENnVoRjVsV3h4NXAyeEd1V2JRSkY0WnJzaktLTW1CMnJrUnR5UDVYV2xWZU1mR1VjbFdjc1gxOW91clMKaWpKSTFnPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBdW4raWVvTVNLUDROdEtmdVlhNXlFNGhzaC9MVkpsNHJ2Q0dmamsvdnJZNEx2YllNCnFvSjdKZnhnamsreTJmTSt6VS9waXhpZjNTZzFnRnNLSlRWUy9LdDlod1dhK3dvald2cVFLcEZ4Z3Qrayt1QTkKaGZjU3lvbEdad0JESVYxZlpvY25nYzQ1YXZMdXFYRGtwSE1pRjdacmprd0JGbXZtV2NrVnpYS1lXOXJlYWs2bgo0Ui9BdVJBdVFmaVovaWlsZ24rZXB5WFd6NG9KeE1jUjJHZHFFanh1WXMzWnhuZUdjSUxidE8rSkpvVXB1YmQ1Ci9OdW8vL3FKdjUxTW9BQXA5dW5idnBtWU9EcW9ISmd2cjZWN2dOaFo3NXpTdi9lV3Z1QWU2WkRDdzdwY24wTjkKYS9SS3ZxYmFYZVE3bUkzcm1YQ2o2c0wxS1ZOMkVFaEMxRXRKRVFJREFRQUJBb0lCQVFDTEVFa3pXVERkYURNSQpGb0JtVGhHNkJ1d0dvMGZWQ0R0TVdUWUVoQTZRTjI4QjB4RzJ3dnpZNGt1TlVsaG10RDZNRVo1dm5iajJ5OWk1CkVTbUxmU3VZUkxlaFNzaTVrR0cwb1VtR3RGVVQ1WGU3cWlHMkZ2bm9GRnh1eVg5RkRiN3BVTFpnMEVsNE9oVkUKTzI0Q1FlZVdEdXc4ZXVnRXRBaGJ3dG1ERElRWFdPSjcxUEcwTnZKRHIwWGpkcW1aeExwQnEzcTJkZTU2YmNjawpPYzV6dmtJNldrb0o1TXN0WkZpU3pVRDYzN3lIbjh2NGd3cXh0bHFoNWhGLzEwV296VmZqVGdWSG0rc01ZaU9SCmNIZ0dMNUVSbDZtVlBsTTQzNUltYnFnU1R2NFFVVGpzQjRvbVBsTlV5Yksvb3pPSWx3RjNPTkJjVVV6eDQ1cGwKSHVJQlQwZ1JBb0dCQU9SR2lYaVBQejdsay9Bc29tNHkxdzFRK2hWb3Yvd3ovWFZaOVVkdmR6eVJ1d3gwZkQ0QgpZVzlacU1hK0JodnB4TXpsbWxYRHJBMklYTjU3UEM3ZUo3enhHMEVpZFJwN3NjN2VmQUN0eDN4N0d0V2pRWGF2ClJ4R2xDeUZxVG9LY3NEUjBhQ0M0Um15VmhZRTdEY0huLy9oNnNzKys3U2tvRVMzNjhpS1RiYzZQQW9HQkFORW0KTHRtUmZieHIrOE5HczhvdnN2Z3hxTUlxclNnb2NmcjZoUlZnYlU2Z3NFd2pMQUs2ZHdQV0xWQmVuSWJ6bzhodApocmJHU1piRnF0bzhwS1Q1d2NxZlpKSlREQnQxYmhjUGNjWlRmSnFmc0VISXc0QW5JMVdRMlVzdzVPcnZQZWhsCmh0ek95cXdBSGZvWjBUTDlseTRJUHRqbXArdk1DQ2NPTHkwanF6NWZBb0dCQUlNNGpRT3hqSkN5VmdWRkV5WTMKc1dsbE9DMGdadVFxV3JPZnY2Q04wY1FPbmJCK01ZRlBOOXhUZFBLeC96OENkVyszT0syK2FtUHBGRUdNSTc5cApVdnlJdUxzTGZMZDVqVysyY3gvTXhaU29DM2Z0ZmM4azJMeXEzQ2djUFA5VjVQQnlUZjBwRU1xUWRRc2hrRG44CkRDZWhHTExWTk8xb3E5OTdscjhMY3A2L0FvR0FYNE5KZC9CNmRGYjRCYWkvS0lGNkFPQmt5aTlGSG9iQjdyVUQKbTh5S2ZwTGhrQk9yNEo4WkJQYUZnU09ENWhsVDNZOHZLejhJa2tNNUVDc0xvWSt4a1lBVEpNT3FUc3ZrOThFRQoyMlo3Qy80TE55K2hJR0EvUWE5Qm5KWDZwTk9XK1ErTWRFQTN6QzdOZ2M3U2U2L1ZuNThDWEhtUmpCeUVTSm13CnI3T1BXNDhDZ1lBVUVoYzV2VnlERXJxVDBjN3lIaXBQbU1wMmljS1hscXNhdC94YWtobENqUjZPZ2I5aGQvNHIKZm1wUHJmd3hjRmJrV2tDRUhJN01EdDJrZXNEZUhRWkFxN2xEdjVFT2k4ZG1uM0ZPNEJWczhCOWYzdm52MytmZwpyV2E3ZGtyWnFudU12cHhpSWlqOWZEak9XbzdxK3hTSFcxWWdSNGV2Q1p2NGxJU0FZRlViemc9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
`
)

func TestWaitForClusterDomain_Run(t *testing.T) {

	clusterName := "name"
	runtimeID := "runtimeID"
	tenant := "tenant"
	domain := "cluster.kymaa.com"

	cluster := model.Cluster{
		ID:     runtimeID,
		Tenant: tenant,
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
		Kubeconfig: util.PtrTo(kubeconfig),
	}

	for _, testCase := range []struct {
		description   string
		mockFunc      func(gardenerClient *gardenerMocks.GardenerClient)
		expectedStage model.OperationStage
		expectedDelay time.Duration
	}{
		{
			description: "should continue waiting if domain name is not set",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(&gardener_types.Shoot{}, nil)
			},
			expectedStage: model.WaitingForClusterDomain,
			expectedDelay: 5 * time.Second,
		},
		{
			description: "should go to the next stage if domain name is available",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootWithDomainSet(clusterName, domain), nil)

				runtime := fixRuntime(runtimeID, clusterName, map[string]interface{}{
					"label": "value",
				})
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
		{
			description: "should retry on failed GetRuntime call and go to the next stage if domain name is available",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootWithDomainSet(clusterName, domain), nil)

				runtime := fixRuntime(runtimeID, clusterName, map[string]interface{}{
					"label": "value",
				})
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
		{
			description: "should retry on failed UpdateRuntime call and go to the next stage if domain name is available",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootWithDomainSet(clusterName, domain), nil)

				runtime := fixRuntime(runtimeID, clusterName, map[string]interface{}{
					"label": "value",
				})
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardenerMocks.GardenerClient{}
			directorClient := &directormock.DirectorClient{}

			testCase.mockFunc(gardenerClient)

			waitForClusterDomainStep := NewWaitForClusterDomainStep(gardenerClient, nextStageName, 10*time.Minute)

			// when
			result, err := waitForClusterDomainStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			gardenerClient.AssertExpectations(t)
			directorClient.AssertExpectations(t)
		})
	}

	// tests for disabled Director integration
	for _, testCase := range []struct {
		description   string
		mockFunc      func(gardenerClient *gardenerMocks.GardenerClient)
		expectedStage model.OperationStage
		expectedDelay time.Duration
	}{
		{
			description: "should continue waiting if domain name is not set when Director integration is disabled",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(&gardener_types.Shoot{}, nil)
			},
			expectedStage: model.WaitingForClusterDomain,
			expectedDelay: 5 * time.Second,
		},
		{
			description: "should go to the next stage if domain name is available and Director integration is disabled",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootWithDomainSet(clusterName, domain), nil)
			},
			expectedStage: nextStageName,
			expectedDelay: 0,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardenerMocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForClusterDomainStep := NewWaitForClusterDomainStep(gardenerClient, nextStageName, 10*time.Minute)

			// when
			result, err := waitForClusterDomainStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			gardenerClient.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description        string
		mockFunc           func(gardenerClient *gardenerMocks.GardenerClient, directorClient *directormock.DirectorClient)
		cluster            model.Cluster
		unrecoverableError bool
	}{
		{
			description: "should return error if failed to read Shoot",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient, _ *directormock.DirectorClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, apperrors.Internal("some error"))
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return error if failed to get Runtime from Director",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient, directorClient *directormock.DirectorClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootWithDomainSet(clusterName, domain), nil)
				directorClient.On("GetRuntime", runtimeID, tenant).Return(graphql.RuntimeExt{}, apperrors.Internal("some error"))

			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return error if failed to update Runtime in Director",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient, directorClient *directormock.DirectorClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(fixShootWithDomainSet(clusterName, domain), nil)

				runtime := fixRuntime(runtimeID, clusterName, map[string]interface{}{
					"label": "value",
				})
				directorClient.On("GetRuntime", runtimeID, tenant).Return(runtime, nil)
				directorClient.On("UpdateRuntime", runtimeID, mock.Anything, tenant).Return(apperrors.Internal("some error"))
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardenerMocks.GardenerClient{}
			directorClient := &directormock.DirectorClient{}

			testCase.mockFunc(gardenerClient, directorClient)

			waitForClusterDomainStep := NewWaitForClusterDomainStep(gardenerClient, nextStageName, 10*time.Minute)

			// when
			_, err := waitForClusterDomainStep.Run(testCase.cluster, model.Operation{}, logrus.New())

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))

			gardenerClient.AssertExpectations(t)
			directorClient.AssertExpectations(t)
		})
	}

	// tests for disabled Director integration
	for _, testCase := range []struct {
		description        string
		mockFunc           func(gardenerClient *gardenerMocks.GardenerClient)
		cluster            model.Cluster
		unrecoverableError bool
	}{
		{
			description: "should return error if failed to read Shoot",
			mockFunc: func(gardenerClient *gardenerMocks.GardenerClient) {
				gardenerClient.On("Get", context.Background(), clusterName, mock.Anything).Return(nil, apperrors.Internal("some error"))
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardenerMocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForClusterDomainStep := NewWaitForClusterDomainStep(gardenerClient, nextStageName, 10*time.Minute)

			// when
			_, err := waitForClusterDomainStep.Run(testCase.cluster, model.Operation{}, logrus.New())

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))

			gardenerClient.AssertExpectations(t)
		})
	}
}

func fixShootWithDomainSet(name, domain string) *gardener_types.Shoot {
	return &gardener_types.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: gardener_types.ShootSpec{
			DNS: &gardener_types.DNS{
				Domain: &domain,
			},
		},
	}
}

func fixRuntime(runtimeId, name string, labels map[string]interface{}) graphql.RuntimeExt {
	return graphql.RuntimeExt{
		Runtime: graphql.Runtime{
			ID:   runtimeId,
			Name: name,
		},
		Labels: labels,
	}
}
