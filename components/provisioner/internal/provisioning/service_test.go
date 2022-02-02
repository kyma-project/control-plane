package provisioning

import (
	"testing"
	"time"

	installationSDK "github.com/kyma-incubator/hydroform/install/installation"
	installationMocks "github.com/kyma-project/control-plane/components/provisioner/internal/installation/mocks"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/mocks"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	uuidMocks "github.com/kyma-project/control-plane/components/provisioner/internal/uuid/mocks"

	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"

	mocks2 "github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/mocks"
	sessionMocks "github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"

	releaseMocks "github.com/kyma-project/control-plane/components/provisioner/internal/installation/release/mocks"

	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid"

	directormock "github.com/kyma-project/control-plane/components/provisioner/internal/director/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	runtimeID   = "184ccdf2-59e4-44b7-b553-6cb296af5ea0"
	operationID = "223949ed-e6b6-4ab2-ab3e-8e19cd456dd40"
	runtimeName = "test runtime"

	tenant        = "tenant"
	subAccountId  = "sub-account"
	administrator = "test@test.pl"

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

var (
	kymaRelease = model.Release{
		Id:            "releaseId",
		Version:       kymaVersion,
		TillerYAML:    "tiller yaml",
		InstallerYAML: "installer yaml",
	}
)

func TestService_ProvisionRuntime(t *testing.T) {
	releaseProvider := &releaseMocks.Provider{}
	releaseProvider.On("GetReleaseByVersion", kymaVersion).Return(kymaRelease, nil)

	inputConverter := NewInputConverter(uuid.NewUUIDGenerator(), releaseProvider, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, forceAllowPrivilegedContainers)
	graphQLConverter := NewGraphQLConverter()
	uuidGenerator := uuid.NewUUIDGenerator()

	clusterConfig := &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			KubernetesVersion: "1.16",
			ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
				GcpConfig: &gqlschema.GCPProviderConfigInput{},
			},
			OidcConfig: oidcInput(),
			DNSConfig:  dnsInput(),
		},
	}

	expectedCluster := model.Cluster{
		ID:         runtimeID,
		KymaConfig: fixKymaConfig(nil),
	}
	expectedOperation := model.Operation{
		ClusterID: runtimeID,
		State:     model.InProgress,
		Type:      model.Provision,
		Stage:     model.WaitingForClusterDomain,
	}

	runtimeInput := &gqlschema.RuntimeInput{
		Name:        runtimeName,
		Description: new(string),
		Labels:      &gqlschema.Labels{},
	}

	provisionRuntimeInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput:  runtimeInput,
		ClusterConfig: clusterConfig,
		KymaConfig:    fixKymaGraphQLConfigInput(nil),
	}

	provisionRuntimeInputNoKymaConfig := gqlschema.ProvisionRuntimeInput{
		RuntimeInput:  runtimeInput,
		ClusterConfig: clusterConfig,
	}

	clusterMatcher := getClusterMatcher(expectedCluster)
	operationMatcher := getOperationMatcher(expectedOperation)

	t.Run("Should start runtime provisioning of Gardener cluster and return operation ID with Kyma config", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
		directorServiceMock := &directormock.DirectorClient{}
		provisioner := &mocks2.Provisioner{}

		provisioningQueue := &mocks.OperationQueue{}

		directorServiceMock.On("CreateRuntime", mock.Anything, tenant).Return(runtimeID, nil)
		sessionFactoryMock.On("NewSessionWithinTransaction").Return(writeSessionWithinTransactionMock, nil)
		writeSessionWithinTransactionMock.On("InsertCluster", mock.MatchedBy(clusterMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("InsertGardenerConfig", mock.AnythingOfType("model.GardenerConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertKymaConfig", mock.AnythingOfType("model.KymaConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("Commit").Return(nil)
		writeSessionWithinTransactionMock.On("RollbackUnlessCommitted").Return()
		provisioner.On("ProvisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(nil)

		provisioningQueue.On("Add", mock.AnythingOfType("string")).Return(nil)

		service := NewProvisioningService(inputConverter, graphQLConverter, directorServiceMock, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, provisioningQueue, nil, nil, nil, nil, nil, nil)

		//when
		operationStatus, err := service.ProvisionRuntime(provisionRuntimeInput, tenant, subAccountId)
		require.NoError(t, err)

		//then
		assert.Equal(t, runtimeID, *operationStatus.RuntimeID)
		assert.Equal(t, gqlschema.OperationTypeProvision, operationStatus.Operation)
		assert.NotEmpty(t, operationStatus.ID)
		sessionFactoryMock.AssertExpectations(t)
		writeSessionWithinTransactionMock.AssertExpectations(t)
		directorServiceMock.AssertExpectations(t)
		provisioner.AssertExpectations(t)
		releaseProvider.AssertExpectations(t)
	})

	t.Run("Should start runtime provisioning of Gardener cluster and return operation ID without Kyma Config", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
		directorServiceMock := &directormock.DirectorClient{}
		provisioner := &mocks2.Provisioner{}

		provisioningNoInstallQueue := &mocks.OperationQueue{}

		expectedNoInstallOperation := model.Operation{
			ClusterID: runtimeID,
			State:     model.InProgress,
			Type:      model.ProvisionNoInstall,
			Stage:     model.WaitingForClusterDomain,
		}

		noInstallOperationMatcher := getOperationMatcher(expectedNoInstallOperation)

		directorServiceMock.On("CreateRuntime", mock.Anything, tenant).Return(runtimeID, nil)
		sessionFactoryMock.On("NewSessionWithinTransaction").Return(writeSessionWithinTransactionMock, nil)
		writeSessionWithinTransactionMock.On("InsertCluster", mock.MatchedBy(clusterMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("InsertGardenerConfig", mock.AnythingOfType("model.GardenerConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertOperation", mock.MatchedBy(noInstallOperationMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("Commit").Return(nil)
		writeSessionWithinTransactionMock.On("RollbackUnlessCommitted").Return()
		provisioner.On("ProvisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(nil)

		provisioningNoInstallQueue.On("Add", mock.AnythingOfType("string")).Return(nil)

		service := NewProvisioningService(inputConverter, graphQLConverter, directorServiceMock, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, nil, provisioningNoInstallQueue, nil, nil, nil, nil, nil)

		//when
		operationStatus, err := service.ProvisionRuntime(provisionRuntimeInputNoKymaConfig, tenant, subAccountId)
		require.NoError(t, err)

		//then
		assert.Equal(t, runtimeID, *operationStatus.RuntimeID)
		assert.Equal(t, gqlschema.OperationTypeProvisionNoInstall, operationStatus.Operation)
		assert.NotEmpty(t, operationStatus.ID)
		sessionFactoryMock.AssertExpectations(t)
		writeSessionWithinTransactionMock.AssertExpectations(t)
		directorServiceMock.AssertExpectations(t)
		provisioner.AssertExpectations(t)
		releaseProvider.AssertExpectations(t)
	})

	t.Run("Should return error and unregister Runtime when failed to commit transaction", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
		directorServiceMock := &directormock.DirectorClient{}
		provisioner := &mocks2.Provisioner{}

		directorServiceMock.On("CreateRuntime", mock.Anything, tenant).Return(runtimeID, nil)
		sessionFactoryMock.On("NewSessionWithinTransaction").Return(writeSessionWithinTransactionMock, nil)
		writeSessionWithinTransactionMock.On("InsertCluster", mock.MatchedBy(clusterMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("InsertGardenerConfig", mock.AnythingOfType("model.GardenerConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertKymaConfig", mock.AnythingOfType("model.KymaConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("Commit").Return(dberrors.Internal("error"))
		writeSessionWithinTransactionMock.On("RollbackUnlessCommitted").Return()
		provisioner.On("ProvisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(nil)
		directorServiceMock.On("DeleteRuntime", runtimeID, tenant).Return(nil)

		service := NewProvisioningService(inputConverter, graphQLConverter, directorServiceMock, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := service.ProvisionRuntime(provisionRuntimeInput, tenant, subAccountId)
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "Failed to commit transaction")
		sessionFactoryMock.AssertExpectations(t)
		writeSessionWithinTransactionMock.AssertExpectations(t)
		directorServiceMock.AssertExpectations(t)
		provisioner.AssertExpectations(t)
		releaseProvider.AssertExpectations(t)
	})

	t.Run("Should return error and unregister Runtime when failed to start provisioning", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
		directorServiceMock := &directormock.DirectorClient{}
		provisioner := &mocks2.Provisioner{}

		directorServiceMock.On("CreateRuntime", mock.Anything, tenant).Return(runtimeID, nil)
		sessionFactoryMock.On("NewSessionWithinTransaction").Return(writeSessionWithinTransactionMock, nil)
		writeSessionWithinTransactionMock.On("InsertCluster", mock.MatchedBy(clusterMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("InsertGardenerConfig", mock.AnythingOfType("model.GardenerConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertKymaConfig", mock.AnythingOfType("model.KymaConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("RollbackUnlessCommitted").Return()
		provisioner.On("ProvisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(apperrors.Internal("error"))
		directorServiceMock.On("DeleteRuntime", runtimeID, tenant).Return(nil)

		service := NewProvisioningService(inputConverter, graphQLConverter, directorServiceMock, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := service.ProvisionRuntime(provisionRuntimeInput, tenant, subAccountId)
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)

		//then
		assert.Contains(t, err.Error(), "Failed to start provisioning")
		sessionFactoryMock.AssertExpectations(t)
		writeSessionWithinTransactionMock.AssertExpectations(t)
		directorServiceMock.AssertExpectations(t)
		provisioner.AssertExpectations(t)
		releaseProvider.AssertExpectations(t)
	})

	t.Run("Should return error when failed to register Runtime", func(t *testing.T) {
		//given
		directorServiceMock := &directormock.DirectorClient{}

		directorServiceMock.On("CreateRuntime", mock.Anything, tenant).Return("", apperrors.Internal("registering error"))

		service := NewProvisioningService(inputConverter, graphQLConverter, directorServiceMock, nil, nil, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := service.ProvisionRuntime(provisionRuntimeInput, tenant, subAccountId)
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)

		//then
		assert.Contains(t, err.Error(), "Failed to register Runtime")
		directorServiceMock.AssertExpectations(t)
	})

	t.Run("Should retry when failed to register Runtime and start runtime provisioning of Gardener cluster", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
		directorServiceMock := &directormock.DirectorClient{}
		provisioner := &mocks2.Provisioner{}

		provisioningQueue := &mocks.OperationQueue{}

		directorServiceMock.On("CreateRuntime", mock.Anything, tenant).Once().Return("", apperrors.Internal("registering error"))
		directorServiceMock.On("CreateRuntime", mock.Anything, tenant).Once().Return(runtimeID, nil)
		sessionFactoryMock.On("NewSessionWithinTransaction").Return(writeSessionWithinTransactionMock, nil)
		writeSessionWithinTransactionMock.On("InsertCluster", mock.MatchedBy(clusterMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("InsertGardenerConfig", mock.AnythingOfType("model.GardenerConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertKymaConfig", mock.AnythingOfType("model.KymaConfig")).Return(nil)
		writeSessionWithinTransactionMock.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
		writeSessionWithinTransactionMock.On("Commit").Return(nil)
		writeSessionWithinTransactionMock.On("RollbackUnlessCommitted").Return()
		provisioner.On("ProvisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(nil)

		provisioningQueue.On("Add", mock.AnythingOfType("string")).Return(nil)

		service := NewProvisioningService(inputConverter, graphQLConverter, directorServiceMock, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, provisioningQueue, nil, nil, nil, nil, nil, nil)

		//when
		operationStatus, err := service.ProvisionRuntime(provisionRuntimeInput, tenant, subAccountId)
		require.NoError(t, err)

		//then
		assert.Equal(t, runtimeID, *operationStatus.RuntimeID)
		assert.NotEmpty(t, operationStatus.ID)
		sessionFactoryMock.AssertExpectations(t)
		writeSessionWithinTransactionMock.AssertExpectations(t)
		directorServiceMock.AssertExpectations(t)
		provisioner.AssertExpectations(t)
		releaseProvider.AssertExpectations(t)
	})

}

func TestService_DeprovisionRuntime(t *testing.T) {

	inputConverter := NewInputConverter(uuid.NewUUIDGenerator(), nil, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, forceAllowPrivilegedContainers)
	graphQLConverter := NewGraphQLConverter()
	lastOperation := model.Operation{State: model.Succeeded}
	mockedKubeconfig := kubeconfig

	operation := model.Operation{
		ID:             operationID,
		Type:           model.Deprovision,
		State:          model.InProgress,
		StartTimestamp: time.Now(),
		Message:        "Deprovisioning started",
		ClusterID:      runtimeID,
	}

	cluster := model.Cluster{
		ID: runtimeID,
		KymaConfig: &model.KymaConfig{
			ID: "id",
		},
		ActiveKymaConfigId: util.StringPtr("activekymaconfigid"),
		Kubeconfig:         &mockedKubeconfig,
	}

	installedState := installationSDK.InstallationState{
		State:       string(v1alpha1.StateInstalled),
		Description: "Kyma 1.x is installed on the cluster",
	}

	notInstalledState := installationSDK.InstallationState{
		State:       installationSDK.NoInstallationState,
		Description: "Kyma Installation CR not found on the cluster",
	}

	clusterMatcher := getClusterMatcher(cluster)
	operationMatcher := getOperationMatcher(operation)

	t.Run("Should start Runtime deprovisioning and return operation ID when activeKymaConfigID exists AND Kyma cluster is in installed state", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readWriteSession := &sessionMocks.ReadWriteSession{}
		provisioner := &mocks2.Provisioner{}
		installationClient := &installationMocks.Service{}

		deprovisioningQueue := &mocks.OperationQueue{}

		deprovisioningQueue.On("Add", mock.AnythingOfType("string")).Return(nil)

		sessionFactoryMock.On("NewReadWriteSession").Return(readWriteSession)
		readWriteSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
		readWriteSession.On("GetCluster", runtimeID).Return(cluster, nil)
		provisioner.On("DeprovisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(operation, nil)
		readWriteSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
		installationClient.On("CheckInstallationState", mock.Anything).Return(installedState, nil)

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisioner, uuid.NewUUIDGenerator(), nil, installationClient, nil, nil, deprovisioningQueue, nil, nil, nil, nil)

		//when
		opID, err := resolver.DeprovisionRuntime(runtimeID)
		require.NoError(t, err)

		//then
		assert.Equal(t, operationID, opID)
		sessionFactoryMock.AssertExpectations(t)
		readWriteSession.AssertExpectations(t)
		provisioner.AssertExpectations(t)
		installationClient.AssertExpectations(t)
		deprovisioningQueue.AssertExpectations(t)
	})

	t.Run("Should start Runtime deprovisioning without installation and return operation ID when activeKymaConfigID exists after upgrade from 1.x to 2.x BUT Kyma cluster is not reported as installed", func(t *testing.T) {
		//given
		operation := model.Operation{
			ID:             operationID,
			Type:           model.DeprovisionNoInstall,
			State:          model.InProgress,
			StartTimestamp: time.Now(),
			Message:        "Deprovisioning without installation started",
			ClusterID:      runtimeID,
		}
		operationMatcher := getOperationMatcher(operation)

		sessionFactoryMock := &sessionMocks.Factory{}
		readWriteSession := &sessionMocks.ReadWriteSession{}
		provisioner := &mocks2.Provisioner{}
		installationClient := &installationMocks.Service{}

		deprovisioningNoInstallQueue := &mocks.OperationQueue{}

		deprovisioningNoInstallQueue.On("Add", mock.AnythingOfType("string")).Return(nil)

		sessionFactoryMock.On("NewReadWriteSession").Return(readWriteSession)
		readWriteSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
		readWriteSession.On("GetCluster", runtimeID).Return(cluster, nil)
		provisioner.On("DeprovisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(operation, nil)
		readWriteSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)

		installationClient.On("CheckInstallationState", mock.Anything).Return(notInstalledState, nil)

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisioner, uuid.NewUUIDGenerator(), nil, installationClient, nil, nil, nil, deprovisioningNoInstallQueue, nil, nil, nil)

		//when
		opID, err := resolver.DeprovisionRuntime(runtimeID)
		require.NoError(t, err)

		//then
		assert.Equal(t, operationID, opID)
		sessionFactoryMock.AssertExpectations(t)
		readWriteSession.AssertExpectations(t)
		provisioner.AssertExpectations(t)
		installationClient.AssertExpectations(t)
		deprovisioningNoInstallQueue.AssertExpectations(t)
	})

	t.Run("Should start Runtime deprovisioning without installation and return operation ID when activeKymaConfigID is missing AND Kyma cluster is in nonInstalled state ", func(t *testing.T) {
		//given
		operation := model.Operation{
			ID:             operationID,
			Type:           model.DeprovisionNoInstall,
			State:          model.InProgress,
			StartTimestamp: time.Now(),
			Message:        "Deprovisioning without installation started",
			ClusterID:      runtimeID,
		}

		cluster := model.Cluster{
			ID:         runtimeID,
			Kubeconfig: &mockedKubeconfig,
		}
		clusterMatcher := getClusterMatcher(cluster)
		operationMatcher := getOperationMatcher(operation)

		sessionFactoryMock := &sessionMocks.Factory{}
		readWriteSession := &sessionMocks.ReadWriteSession{}
		provisioner := &mocks2.Provisioner{}
		installationClient := &installationMocks.Service{}

		deprovisioningNoInstallQueue := &mocks.OperationQueue{}

		deprovisioningNoInstallQueue.On("Add", mock.AnythingOfType("string")).Return(nil)

		sessionFactoryMock.On("NewReadWriteSession").Return(readWriteSession)
		readWriteSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
		readWriteSession.On("GetCluster", runtimeID).Return(cluster, nil)
		provisioner.On("DeprovisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(operation, nil)
		readWriteSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)

		installationClient.On("CheckInstallationState", mock.Anything).Return(notInstalledState, nil)

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisioner, uuid.NewUUIDGenerator(), nil, installationClient, nil, nil, nil, deprovisioningNoInstallQueue, nil, nil, nil)

		//when
		opID, err := resolver.DeprovisionRuntime(runtimeID)
		require.NoError(t, err)

		//then
		assert.Equal(t, operationID, opID)
		sessionFactoryMock.AssertExpectations(t)
		readWriteSession.AssertExpectations(t)
		provisioner.AssertExpectations(t)
		installationClient.AssertExpectations(t)
		deprovisioningNoInstallQueue.AssertExpectations(t)
	})

	t.Run("Should return error when failed to start deprovisioning", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readWriteSession := &sessionMocks.ReadWriteSession{}
		provisioner := &mocks2.Provisioner{}

		sessionFactoryMock.On("NewReadWriteSession").Return(readWriteSession)
		readWriteSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
		readWriteSession.On("GetCluster", runtimeID).Return(cluster, nil)
		provisioner.On("DeprovisionCluster", mock.MatchedBy(clusterMatcher), mock.MatchedBy(notEmptyUUIDMatcher)).Return(model.Operation{}, apperrors.Internal("error"))

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisioner, uuid.NewUUIDGenerator(), nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := resolver.DeprovisionRuntime(runtimeID)
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeInternal)

		//then
		assert.Contains(t, err.Error(), "Failed to start deprovisioning")
		sessionFactoryMock.AssertExpectations(t)
		readWriteSession.AssertExpectations(t)
		provisioner.AssertExpectations(t)
	})

	t.Run("Should return error while deprovisioning when failed to get cluster ", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readWriteSession := &sessionMocks.ReadWriteSession{}

		sessionFactoryMock.On("NewReadWriteSession").Return(readWriteSession)
		readWriteSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
		readWriteSession.On("GetCluster", runtimeID).Return(model.Cluster{}, dberrors.Internal("error"))

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, nil, uuid.NewUUIDGenerator(), nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := resolver.DeprovisionRuntime(runtimeID)
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "Failed to get cluster")
		sessionFactoryMock.AssertExpectations(t)
		readWriteSession.AssertExpectations(t)
	})

	t.Run("Should return error while deprovisioning when last operation in progress", func(t *testing.T) {
		//given
		operation := model.Operation{State: model.InProgress}

		sessionFactoryMock := &sessionMocks.Factory{}
		readWriteSession := &sessionMocks.ReadWriteSession{}

		sessionFactoryMock.On("NewReadWriteSession").Return(readWriteSession)
		readWriteSession.On("GetLastOperation", runtimeID).Return(operation, nil)

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, nil, uuid.NewUUIDGenerator(), nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := resolver.DeprovisionRuntime(runtimeID)
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "previous one is in progress")
		sessionFactoryMock.AssertExpectations(t)
		readWriteSession.AssertExpectations(t)
	})

	t.Run("Should return error when failed to get last operation", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readWriteSession := &sessionMocks.ReadWriteSession{}

		sessionFactoryMock.On("NewReadWriteSession").Return(readWriteSession)
		readWriteSession.On("GetLastOperation", runtimeID).Return(model.Operation{}, dberrors.Internal("error"))

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, nil, uuid.NewUUIDGenerator(), nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := resolver.DeprovisionRuntime(runtimeID)
		require.Error(t, err)

		//then
		assert.Contains(t, err.Error(), "failed to get last operation")
		sessionFactoryMock.AssertExpectations(t)
		readWriteSession.AssertExpectations(t)
	})
}

func TestService_RuntimeOperationStatus(t *testing.T) {
	uuidGenerator := &uuidMocks.UUIDGenerator{}
	inputConverter := NewInputConverter(uuidGenerator, nil, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, forceAllowPrivilegedContainers)
	graphQLConverter := NewGraphQLConverter()

	operation := model.Operation{
		ID:        operationID,
		Type:      model.Provision,
		State:     model.InProgress,
		Message:   "Message",
		ClusterID: runtimeID,
	}

	t.Run("Should return operation status", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readSession := &sessionMocks.ReadSession{}

		sessionFactoryMock.On("NewReadSession").Return(readSession)
		readSession.On("GetOperation", operationID).Return(operation, nil)

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, nil, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		status, err := resolver.RuntimeOperationStatus(operationID)
		//then
		require.NoError(t, err)
		assert.Equal(t, gqlschema.OperationTypeProvision, status.Operation)
		assert.Equal(t, gqlschema.OperationStateInProgress, status.State)
		assert.Equal(t, operation.ClusterID, *status.RuntimeID)
		assert.Equal(t, operation.ID, *status.ID)
		assert.Equal(t, operation.Message, *status.Message)
		sessionFactoryMock.AssertExpectations(t)
		readSession.AssertExpectations(t)
	})

	t.Run("Should return error when failed to get operation status", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readSession := &sessionMocks.ReadSession{}

		sessionFactoryMock.On("NewReadSession").Return(readSession)
		readSession.On("GetOperation", operationID).Return(model.Operation{}, dberrors.Internal("error"))

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, nil, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := resolver.RuntimeOperationStatus(operationID)

		//then
		require.Error(t, err)
		sessionFactoryMock.AssertExpectations(t)
		readSession.AssertExpectations(t)
	})
}

func TestService_RuntimeStatus(t *testing.T) {
	uuidGenerator := &uuidMocks.UUIDGenerator{}
	inputConverter := NewInputConverter(uuidGenerator, nil, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, forceAllowPrivilegedContainers)
	graphQLConverter := NewGraphQLConverter()

	operation := model.Operation{
		ID:        operationID,
		Type:      model.Provision,
		State:     model.Succeeded,
		Message:   "Message",
		ClusterID: runtimeID,
	}

	cluster := model.Cluster{
		ID:         runtimeID,
		Kubeconfig: util.StringPtr("kubeconfig"),
	}

	t.Run("Should return runtime status", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readSession := &sessionMocks.ReadSession{}

		sessionFactoryMock.On("NewReadSession").Return(readSession)
		readSession.On("GetLastOperation", operationID).Return(operation, nil)
		readSession.On("GetCluster", operationID).Return(cluster, nil)

		provisioner := &mocks2.Provisioner{}

		provisioner.On("GetHibernationStatus", mock.AnythingOfType("string"), cluster.ClusterConfig).Return(model.HibernationStatus{
			HibernationPossible: true,
			Hibernated:          true,
		}, nil)

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		status, err := resolver.RuntimeStatus(operationID)

		//then
		require.NoError(t, err)
		assert.Equal(t, cluster.ID, *status.LastOperationStatus.RuntimeID)
		assert.Equal(t, cluster.Kubeconfig, status.RuntimeConfiguration.Kubeconfig)
		sessionFactoryMock.AssertExpectations(t)
		readSession.AssertExpectations(t)
	})

	t.Run("Should return error when failed to get cluster", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readSession := &sessionMocks.ReadSession{}

		sessionFactoryMock.On("NewReadSession").Return(readSession)
		readSession.On("GetLastOperation", operationID).Return(operation, nil)
		readSession.On("GetCluster", operationID).Return(model.Cluster{}, dberrors.Internal("error"))

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, nil, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := resolver.RuntimeStatus(operationID)

		//then
		require.Error(t, err)
		sessionFactoryMock.AssertExpectations(t)
		readSession.AssertExpectations(t)
	})

	t.Run("Should return error when failed to get operation status", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readSession := &sessionMocks.ReadSession{}

		sessionFactoryMock.On("NewReadSession").Return(readSession)
		readSession.On("GetLastOperation", operationID).Return(model.Operation{}, dberrors.Internal("error"))

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, nil, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := resolver.RuntimeStatus(operationID)

		//then
		require.Error(t, err)
		sessionFactoryMock.AssertExpectations(t)
		readSession.AssertExpectations(t)
	})

	t.Run("Should return error when failed to get hibernation status", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		readSession := &sessionMocks.ReadSession{}
		provisioner := &mocks2.Provisioner{}

		sessionFactoryMock.On("NewReadSession").Return(readSession)
		readSession.On("GetLastOperation", operationID).Return(model.Operation{}, nil)
		readSession.On("GetCluster", operationID).Return(cluster, nil)
		provisioner.On("GetHibernationStatus", mock.AnythingOfType("string"), cluster.ClusterConfig).Return(model.HibernationStatus{}, apperrors.Internal("some error"))

		resolver := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		_, err := resolver.RuntimeStatus(operationID)

		//then
		require.Error(t, err)
		sessionFactoryMock.AssertExpectations(t)
		readSession.AssertExpectations(t)
	})
}

func TestService_UpgradeRuntime(t *testing.T) {
	releaseProvider := &releaseMocks.Provider{}
	releaseProvider.On("GetReleaseByVersion", kymaVersion).Return(kymaRelease, nil)
	inputConverter := NewInputConverter(uuid.NewUUIDGenerator(), releaseProvider, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, forceAllowPrivilegedContainers)
	graphQLConverter := NewGraphQLConverter()
	uuidGenerator := uuid.NewUUIDGenerator()

	lastOperation := model.Operation{State: model.Succeeded}

	oldKymaConfigId := "old-kyma-config-id"
	activeKymaConfigId := "active-kyma-config-id"

	cluster := model.Cluster{
		ID: runtimeID,
		KymaConfig: &model.KymaConfig{
			ID: oldKymaConfigId,
		},
		ActiveKymaConfigId: &activeKymaConfigId,
		ClusterConfig: model.GardenerConfig{
			KubernetesVersion: "1.19",
		},
		Tenant: tenant,
	}

	clusterWithNoKymaConfig := model.Cluster{
		ID: runtimeID,
	}

	expectedOperation := model.Operation{
		ClusterID: runtimeID,
		State:     model.InProgress,
		Type:      model.Upgrade,
		Stage:     model.StartingUpgrade,
	}

	runtimeUpgradeMatcher := func(rUp model.RuntimeUpgrade) bool {
		return rUp.OperationId != "" && rUp.PreUpgradeKymaConfigId == oldKymaConfigId && rUp.PostUpgradeKymaConfigId != oldKymaConfigId
	}

	upgradeInput := gqlschema.UpgradeRuntimeInput{
		KymaConfig: fixKymaGraphQLConfigInput(nil),
	}

	operationMatcher := getOperationMatcher(expectedOperation)

	for _, testCase := range []struct {
		description string
		mockFunc    func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider, upgradeQueue *mocks.OperationQueue)
	}{
		{
			description: "should fail to upgrade Runtime when failed to commit transaction",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider, upgradeQueue *mocks.OperationQueue) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("InsertKymaConfig", mock.AnythingOfType("model.KymaConfig")).Return(nil)
				writeSession.On("InsertRuntimeUpgrade", mock.MatchedBy(runtimeUpgradeMatcher)).Return(nil)
				writeSession.On("SetActiveKymaConfig", runtimeID, mock.AnythingOfType("string")).Return(nil)
				writeSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
				writeSession.On("UpdateKubernetesVersion", runtimeID, "1.20").Return(nil)
				writeSession.On("Commit").Return(nil)
				writeSession.On("RollbackUnlessCommitted").Return()
				upgradeQueue.On("Add", mock.AnythingOfType("string")).Return(nil)

				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.20", nil)
			},
		},
		{
			description: "should fail to upgrade Runtime without Kyma config",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider, upgradeQueue *mocks.OperationQueue) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("InsertKymaConfig", mock.AnythingOfType("model.KymaConfig")).Return(nil)
				writeSession.On("InsertRuntimeUpgrade", mock.MatchedBy(runtimeUpgradeMatcher)).Return(nil)
				writeSession.On("SetActiveKymaConfig", runtimeID, mock.AnythingOfType("string")).Return(nil)
				writeSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
				writeSession.On("Commit").Return(nil)
				writeSession.On("RollbackUnlessCommitted").Return()
				upgradeQueue.On("Add", mock.AnythingOfType("string")).Return(nil)

				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.19", nil)
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			sessionFactory := &sessionMocks.Factory{}
			writeSession := &sessionMocks.WriteSessionWithinTransaction{}
			readSession := &sessionMocks.ReadSession{}

			provisioningQueue := &mocks.OperationQueue{}
			deprovisioningQueue := &mocks.OperationQueue{}
			upgradeQueue := &mocks.OperationQueue{}
			upgradeShootQueue := &mocks.OperationQueue{}
			kubernetesVersionProvider := &mocks2.KubernetesVersionProvider{}

			testCase.mockFunc(sessionFactory, writeSession, readSession, kubernetesVersionProvider, upgradeQueue)

			service := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactory, nil, uuidGenerator, kubernetesVersionProvider, nil, provisioningQueue, nil, deprovisioningQueue, nil, upgradeQueue, upgradeShootQueue, nil)

			//when
			operationStatus, err := service.UpgradeRuntime(runtimeID, upgradeInput)
			require.NoError(t, err)

			//then
			assert.Equal(t, runtimeID, *operationStatus.RuntimeID)
			assert.NotEmpty(t, operationStatus.ID)
			sessionFactory.AssertExpectations(t)
			writeSession.AssertExpectations(t)
			readSession.AssertExpectations(t)
			upgradeQueue.AssertExpectations(t)
			releaseProvider.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description string
		mockFunc    func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider)
	}{
		{
			description: "should fail to upgrade Runtime when failed to commit transaction",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("InsertKymaConfig", mock.AnythingOfType("model.KymaConfig")).Return(nil)
				writeSession.On("InsertRuntimeUpgrade", mock.MatchedBy(runtimeUpgradeMatcher)).Return(nil)
				writeSession.On("SetActiveKymaConfig", runtimeID, mock.AnythingOfType("string")).Return(nil)
				writeSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
				writeSession.On("UpdateKubernetesVersion", runtimeID, "1.20").Return(nil)
				writeSession.On("Commit").Return(dberrors.Internal("error"))
				writeSession.On("RollbackUnlessCommitted").Return()
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.20", nil)
			},
		},
		{
			description: "should fail to upgrade Runtime without Kyma config",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(clusterWithNoKymaConfig, nil)
			},
		},
		{
			description: "should fail to upgrade Runtime when failed to insert new Kyma Config",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("InsertKymaConfig", mock.AnythingOfType("model.KymaConfig")).Return(dberrors.Internal("error"))
				writeSession.On("RollbackUnlessCommitted").Return()
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.19", nil)
			},
		},
		{
			description: "should fail to upgrade Runtime when last operation is in progress",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(model.Operation{State: model.InProgress}, nil)
			},
		},
		{
			description: "should fail to upgrade Runtime when failed to get Kubernetes version",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("", apperrors.Internal("oh, no!"))
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			sessionFactory := &sessionMocks.Factory{}
			writeSession := &sessionMocks.WriteSessionWithinTransaction{}
			readSession := &sessionMocks.ReadSession{}

			provisioningQueue := &mocks.OperationQueue{}
			deprovisioningQueue := &mocks.OperationQueue{}
			upgradeQueue := &mocks.OperationQueue{}
			upgradeShootQueue := &mocks.OperationQueue{}
			kubernetesVersionProvider := &mocks2.KubernetesVersionProvider{}

			testCase.mockFunc(sessionFactory, writeSession, readSession, kubernetesVersionProvider)

			service := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactory, nil, uuidGenerator, kubernetesVersionProvider, nil, provisioningQueue, nil, deprovisioningQueue, nil, upgradeQueue, upgradeShootQueue, nil)

			//when
			_, err := service.UpgradeRuntime(runtimeID, upgradeInput)
			require.Error(t, err)

			// then
			sessionFactory.AssertExpectations(t)
			writeSession.AssertExpectations(t)
			readSession.AssertExpectations(t)
			releaseProvider.AssertExpectations(t)
		})
	}
}

func TestService_UpgradeGardenerShoot(t *testing.T) {
	inputConverter := NewInputConverter(uuid.NewUUIDGenerator(), nil, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, forceAllowPrivilegedContainers)
	graphQLConverter := NewGraphQLConverter()
	uuidGenerator := uuid.NewUUIDGenerator()

	lastOperation := model.Operation{State: model.Succeeded}

	providerConfig, _ := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"europe-west1-a"}})
	cluster := model.Cluster{
		ID:     runtimeID,
		Tenant: tenant,
		ClusterConfig: model.GardenerConfig{
			ClusterID:              runtimeID,
			Purpose:                util.StringPtr("evaluation"),
			LicenceType:            util.StringPtr("license"),
			GardenerProviderConfig: providerConfig,
			OIDCConfig:             oidcConfig(),
		},
	}

	upgradeShootInput := newUpgradeShootInputAwsAzureGCP("testing")
	upgradedConfig, err := inputConverter.UpgradeShootInputToGardenerConfig(*upgradeShootInput.GardenerConfig, cluster.ClusterConfig)
	require.NoError(t, err)

	operation := model.Operation{
		ClusterID: runtimeID,
		State:     model.InProgress,
		Type:      model.UpgradeShoot,
		Stage:     model.WaitingForShootNewVersion,
	}

	operationMatcher := getOperationMatcher(operation)

	for _, testCase := range []struct {
		description string
		mockFunc    func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider, upgradeShootQueue *mocks.OperationQueue)
	}{
		{description: "should start runtime provisioning of Gardener cluster, update Kubernetes version and return operation ID",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider, upgradeShootQueue *mocks.OperationQueue) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)

				newUpgradedConfig := upgradedConfig
				newUpgradedConfig.KubernetesVersion = "1.20"

				writeSession.On("UpdateGardenerClusterConfig", newUpgradedConfig).Return(nil)
				writeSession.On("RollbackUnlessCommitted").Return()
				writeSession.On("InsertAdministrators", runtimeID, mock.Anything).Return(nil)
				provisioner.On("setOperationStarted", writeSession, runtimeID, model.UpgradeShoot, model.WaitingForShootNewVersion, nil, nil).Return(mock.MatchedBy(operationMatcher), nil)
				writeSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
				provisioner.On("UpgradeCluster", runtimeID, newUpgradedConfig).Return(nil)
				writeSession.On("Commit").Return(nil)
				upgradeShootQueue.On("Add", mock.AnythingOfType("string")).Return(nil)
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.20", nil)
			},
		},
		{description: "should start runtime provisioning of Gardener cluster and return operation ID",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider, upgradeShootQueue *mocks.OperationQueue) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("UpdateGardenerClusterConfig", upgradedConfig).Return(nil)
				writeSession.On("RollbackUnlessCommitted").Return()
				writeSession.On("InsertAdministrators", runtimeID, mock.Anything).Return(nil)
				provisioner.On("setOperationStarted", writeSession, runtimeID, model.UpgradeShoot, model.WaitingForShootNewVersion, nil, nil).Return(mock.MatchedBy(operationMatcher), nil)
				writeSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
				provisioner.On("UpgradeCluster", runtimeID, upgradedConfig).Return(nil)
				writeSession.On("Commit").Return(nil)
				upgradeShootQueue.On("Add", mock.AnythingOfType("string")).Return(nil)
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.19", nil)
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			sessionFactory := &sessionMocks.Factory{}
			writeSessionWithinTransaction := &sessionMocks.WriteSessionWithinTransaction{}
			readSession := &sessionMocks.ReadSession{}

			provisioner := &mocks2.Provisioner{}
			upgradeShootQueue := &mocks.OperationQueue{}

			kubernetesVersionProvider := &mocks2.KubernetesVersionProvider{}

			testCase.mockFunc(sessionFactory, readSession, writeSessionWithinTransaction, provisioner, kubernetesVersionProvider, upgradeShootQueue)

			service := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactory, provisioner, uuidGenerator, kubernetesVersionProvider, nil, nil, nil, nil, nil, nil, upgradeShootQueue, nil)

			//when
			operationStatus, err := service.UpgradeGardenerShoot(runtimeID, upgradeShootInput)
			require.NoError(t, err)

			//then
			assert.Equal(t, runtimeID, *operationStatus.RuntimeID)
			assert.NotEmpty(t, operationStatus.ID)
			sessionFactory.AssertExpectations(t)
			readSession.AssertExpectations(t)
			writeSessionWithinTransaction.AssertExpectations(t)
			upgradeShootQueue.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description string
		mockFunc    func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider)
	}{
		{description: "should fail to upgrade Shoot when failed to commit shoot update",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("RollbackUnlessCommitted").Return()
				writeSession.On("UpdateGardenerClusterConfig", upgradedConfig).Return(nil)
				writeSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
				writeSession.On("InsertAdministrators", runtimeID, mock.Anything).Return(nil)
				provisioner.On("setOperationStarted", writeSession, runtimeID, model.UpgradeShoot, model.WaitingForShootNewVersion, nil, nil).Return(mock.MatchedBy(operationMatcher), nil)
				provisioner.On("UpgradeCluster", runtimeID, upgradedConfig).Return(nil)
				writeSession.On("Commit").Return(dberrors.Internal("error"))
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.19", nil)
			},
		},
		{description: "should fail to upgrade Shoot when failed to upgrade cluster",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("RollbackUnlessCommitted").Return()
				writeSession.On("UpdateGardenerClusterConfig", upgradedConfig).Return(nil)
				writeSession.On("InsertOperation", mock.MatchedBy(operationMatcher)).Return(nil)
				writeSession.On("InsertAdministrators", runtimeID, mock.Anything).Return(nil)
				provisioner.On("setOperationStarted", writeSession, runtimeID, model.UpgradeShoot, model.WaitingForShootNewVersion, nil, nil).Return(mock.MatchedBy(operationMatcher), nil)
				provisioner.On("UpgradeCluster", runtimeID, upgradedConfig).Return(apperrors.Internal("error"))
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.19", nil)
			},
		},
		{description: "should fail to upgrade Shoot when failed to update gardener cluster config",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("RollbackUnlessCommitted").Return()
				writeSession.On("UpdateGardenerClusterConfig", upgradedConfig).Return(dberrors.Internal("error"))
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.19", nil)
			},
		},
		{description: "should fail to upgrade Shoot when failed to create write session",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(nil, dberrors.Internal("error"))
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("1.19", nil)
			},
		},
		{description: "should fail to upgrade Shoot when failed to get cluster",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(model.Cluster{}, dberrors.Internal("error"))
			},
		},
		{description: "should fail to upgrade Shoot when failed to get last operation",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(model.Operation{}, dberrors.Internal("error"))
			},
		},
		{description: "should fail to upgrade Shoot when last operation is in progress",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(model.Operation{State: model.InProgress}, nil)
			},
		},
		{
			description: "should fail to upgrade Shoot when failed to get Kubernetes version",
			mockFunc: func(sessionFactory *sessionMocks.Factory, readSession *sessionMocks.ReadSession, writeSession *sessionMocks.WriteSessionWithinTransaction, provisioner *mocks2.Provisioner, kubernetesVersionProvider *mocks2.KubernetesVersionProvider) {
				sessionFactory.On("NewReadSession").Return(readSession)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				kubernetesVersionProvider.On("Get", runtimeID, tenant).Return("", apperrors.Internal("oh, no!"))
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			sessionFactory := &sessionMocks.Factory{}
			writeSessionWithinTransaction := &sessionMocks.WriteSessionWithinTransaction{}
			readSession := &sessionMocks.ReadSession{}

			provisioner := &mocks2.Provisioner{}
			upgradeShootQueue := &mocks.OperationQueue{}

			kubernetesVersionProvider := &mocks2.KubernetesVersionProvider{}

			testCase.mockFunc(sessionFactory, readSession, writeSessionWithinTransaction, provisioner, kubernetesVersionProvider)

			service := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactory, provisioner, uuidGenerator, kubernetesVersionProvider, nil, nil, nil, nil, nil, nil, upgradeShootQueue, nil)

			//when
			_, err := service.UpgradeGardenerShoot(runtimeID, upgradeShootInput)
			require.Error(t, err)

			// then
			sessionFactory.AssertExpectations(t)
			writeSessionWithinTransaction.AssertExpectations(t)
			readSession.AssertExpectations(t)
		})
	}
}

func TestService_RollBackLastUpgrade(t *testing.T) {
	releaseProvider := &releaseMocks.Provider{}
	inputConverter := NewInputConverter(uuid.NewUUIDGenerator(), releaseProvider, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, forceAllowPrivilegedContainers)
	graphQLConverter := NewGraphQLConverter()
	uuidGenerator := uuid.NewUUIDGenerator()

	lastOperation := model.Operation{ID: operationID, State: model.Succeeded, Type: model.Upgrade}

	oldKymaConfigId := "old-kyma-config-id"

	runtimeUpgrade := model.RuntimeUpgrade{
		State:                   model.UpgradeSucceeded,
		OperationId:             operationID,
		PreUpgradeKymaConfigId:  oldKymaConfigId,
		PostUpgradeKymaConfigId: "new-id",
	}

	cluster := model.Cluster{
		ID: runtimeID,
		KymaConfig: &model.KymaConfig{
			ID: oldKymaConfigId,
		},
	}

	t.Run("Should start runtime provisioning of Gardener cluster and return operation ID", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
		readSessionMock := &sessionMocks.ReadSession{}
		provisioner := &mocks2.Provisioner{}

		sessionFactoryMock.On("NewReadSession").Return(readSessionMock, nil)
		readSessionMock.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
		readSessionMock.On("GetRuntimeUpgrade", operationID).Return(runtimeUpgrade, nil)
		readSessionMock.On("GetCluster", runtimeID).Return(cluster, nil)
		sessionFactoryMock.On("NewSessionWithinTransaction").Return(writeSessionWithinTransactionMock, nil)
		writeSessionWithinTransactionMock.On("SetActiveKymaConfig", runtimeID, oldKymaConfigId).Return(nil)
		writeSessionWithinTransactionMock.On("UpdateUpgradeState", operationID, model.UpgradeRolledBack).Return(nil)
		writeSessionWithinTransactionMock.On("Commit").Return(nil)
		writeSessionWithinTransactionMock.On("RollbackUnlessCommitted").Return()

		// TODO: consider using matcher instead of mock.Anything

		provisioner.On("GetHibernationStatus", mock.Anything, mock.Anything).Return(model.HibernationStatus{
			HibernationPossible: true,
			Hibernated:          true,
		}, nil)

		service := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		//when
		runtimeStatus, err := service.RollBackLastUpgrade(runtimeID)
		require.NoError(t, err)

		//then
		assert.NotEmpty(t, runtimeStatus)
		sessionFactoryMock.AssertExpectations(t)
		writeSessionWithinTransactionMock.AssertExpectations(t)
		readSessionMock.AssertExpectations(t)
	})

	for _, testCase := range []struct {
		description string
		mockFunc    func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession)
	}{
		{
			description: "should fail to roll back upgrade when failed to commit transaction",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetRuntimeUpgrade", operationID).Return(runtimeUpgrade, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("SetActiveKymaConfig", runtimeID, oldKymaConfigId).Return(nil)
				writeSession.On("UpdateUpgradeState", operationID, model.UpgradeRolledBack).Return(nil)
				writeSession.On("Commit").Return(dberrors.Internal("error"))
				writeSession.On("RollbackUnlessCommitted").Return()
			},
		},
		{
			description: "should fail to roll back upgrade when failed to get last operation",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(model.Operation{}, dberrors.Internal("error"))
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			sessionFactoryMock := &sessionMocks.Factory{}
			writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
			readSessionMock := &sessionMocks.ReadSession{}

			testCase.mockFunc(sessionFactoryMock, writeSessionWithinTransactionMock, readSessionMock)

			service := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, nil, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

			//when
			_, err := service.RollBackLastUpgrade(runtimeID)
			require.Error(t, err)

			//then
			sessionFactoryMock.AssertExpectations(t)
			writeSessionWithinTransactionMock.AssertExpectations(t)
			readSessionMock.AssertExpectations(t)
		})
	}
}

func TestService_HibernateShoot(t *testing.T) {
	releaseProvider := &releaseMocks.Provider{}
	inputConverter := NewInputConverter(uuid.NewUUIDGenerator(), releaseProvider, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, forceAllowPrivilegedContainers)
	uuidGenerator := uuid.NewUUIDGenerator()
	graphQLConverter := NewGraphQLConverter()

	lastOperation := model.Operation{ID: operationID, State: model.Succeeded, Type: model.Upgrade}

	cluster := model.Cluster{
		ID: runtimeID,
	}

	timeNow := time.Now()
	hibernationOperation := model.Operation{
		ID:             operationID,
		Type:           model.Hibernate,
		StartTimestamp: timeNow,
		State:          model.InProgress,
		Message:        "",
		ClusterID:      runtimeID,
		Stage:          model.WaitForHibernation,
		LastTransition: &timeNow,
	}

	for _, testCase := range []struct {
		description string
		mockFunc    func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, provisioner *mocks2.Provisioner)
	}{
		{
			description: "should fail failed to get last operation",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, provisioner *mocks2.Provisioner) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(model.Operation{}, dberrors.Internal("error"))
			},
		},
		{
			description: "should fail when operation in progress",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, provisioner *mocks2.Provisioner) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(model.Operation{ID: operationID, State: model.InProgress, Type: model.Upgrade}, nil)
			},
		},
		{
			description: "should fail when failed to get cluster",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, provisioner *mocks2.Provisioner) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(model.Cluster{}, dberrors.Internal("error"))
			},
		},
		{
			description: "should fail when failed to start transaction",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, provisioner *mocks2.Provisioner) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(nil, dberrors.Internal("error"))
			},
		},
		{
			description: "should fail when failed to set operation started",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, provisioner *mocks2.Provisioner) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("InsertOperation", mock.MatchedBy(getOperationMatcher(hibernationOperation))).Return(dberrors.Internal("error"))
				writeSession.On("RollbackUnlessCommitted").Return(nil)
			},
		},
		{
			description: "should fail when failed to hibernate cluster",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, provisioner *mocks2.Provisioner) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("InsertOperation", mock.MatchedBy(getOperationMatcher(hibernationOperation))).Return(nil)
				writeSession.On("RollbackUnlessCommitted").Return(nil)
				provisioner.On("HibernateCluster", cluster.ID, cluster.ClusterConfig).Return(apperrors.Internal("some error"))
			},
		},
		{
			description: "should fail when failed to commit transaction",
			mockFunc: func(sessionFactory *sessionMocks.Factory, writeSession *sessionMocks.WriteSessionWithinTransaction, readSession *sessionMocks.ReadSession, provisioner *mocks2.Provisioner) {
				sessionFactory.On("NewReadSession").Return(readSession, nil)
				readSession.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
				readSession.On("GetCluster", runtimeID).Return(cluster, nil)
				sessionFactory.On("NewSessionWithinTransaction").Return(writeSession, nil)
				writeSession.On("InsertOperation", mock.MatchedBy(getOperationMatcher(hibernationOperation))).Return(nil)
				writeSession.On("RollbackUnlessCommitted").Return(nil)
				provisioner.On("HibernateCluster", cluster.ID, cluster.ClusterConfig).Return(nil)
				writeSession.On("Commit").Return(dberrors.Internal("error"))
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			//given
			sessionFactoryMock := &sessionMocks.Factory{}
			writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
			readSessionMock := &sessionMocks.ReadSession{}
			provisioner := &mocks2.Provisioner{}

			testCase.mockFunc(sessionFactoryMock, writeSessionWithinTransactionMock, readSessionMock, provisioner)

			service := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisioner, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, nil)

			//when
			_, err := service.HibernateCluster(runtimeID)
			require.Error(t, err)

			//then
			sessionFactoryMock.AssertExpectations(t)
			writeSessionWithinTransactionMock.AssertExpectations(t)
			readSessionMock.AssertExpectations(t)
			provisioner.AssertExpectations(t)
		})
	}

	t.Run("Should hibernate cluster and return operation ID", func(t *testing.T) {
		//given
		sessionFactoryMock := &sessionMocks.Factory{}
		writeSessionWithinTransactionMock := &sessionMocks.WriteSessionWithinTransaction{}
		readSessionMock := &sessionMocks.ReadSession{}
		provisionerMock := &mocks2.Provisioner{}
		hibernationQueue := &mocks.OperationQueue{}

		sessionFactoryMock.On("NewReadSession").Return(readSessionMock, nil)
		readSessionMock.On("GetLastOperation", runtimeID).Return(lastOperation, nil)
		readSessionMock.On("GetCluster", runtimeID).Return(cluster, nil)
		sessionFactoryMock.On("NewSessionWithinTransaction").Return(writeSessionWithinTransactionMock, nil)
		writeSessionWithinTransactionMock.On("InsertOperation", mock.MatchedBy(getOperationMatcher(hibernationOperation))).Return(nil)
		writeSessionWithinTransactionMock.On("RollbackUnlessCommitted").Return(nil)
		provisionerMock.On("HibernateCluster", cluster.ID, cluster.ClusterConfig).Return(nil)
		writeSessionWithinTransactionMock.On("Commit").Return(nil)
		hibernationQueue.On("Add", mock.AnythingOfType("string")).Return()

		service := NewProvisioningService(inputConverter, graphQLConverter, nil, sessionFactoryMock, provisionerMock, uuidGenerator, nil, nil, nil, nil, nil, nil, nil, nil, hibernationQueue)

		//when
		runtimeStatus, err := service.HibernateCluster(runtimeID)
		require.NoError(t, err)

		//then
		assert.NotEmpty(t, runtimeStatus)
		sessionFactoryMock.AssertExpectations(t)
		writeSessionWithinTransactionMock.AssertExpectations(t)
		readSessionMock.AssertExpectations(t)
		provisionerMock.AssertExpectations(t)
	})
}

func getOperationMatcher(expected model.Operation) func(model.Operation) bool {
	return func(op model.Operation) bool {
		return op.Type == expected.Type && op.ClusterID == expected.ClusterID &&
			op.State == expected.State && op.Stage == expected.Stage
	}
}

func getClusterMatcher(expected model.Cluster) func(model.Cluster) bool {
	return func(cluster model.Cluster) bool {
		return cluster.ID == expected.ID
	}
}

func notEmptyUUIDMatcher(id string) bool {
	return len(id) > 0
}
