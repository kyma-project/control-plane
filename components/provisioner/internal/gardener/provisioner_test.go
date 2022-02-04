package gardener

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/stretchr/testify/mock"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	sessionMocks "github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/testkit"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	gardenerMocks "github.com/kyma-project/control-plane/components/provisioner/internal/gardener/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gardenerNamespace = "default"
	runtimeId         = "runtimeId"
	tenant            = "tenant"
	operationId       = "operationId"
	clusterName       = "test-cluster"
	region            = "westeurope"
	purpose           = "production"

	auditLogsPolicyCMName = "audit-logs-policy"
)

func TestGardenerProvisioner_ProvisionCluster(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	gcpGardenerConfig, err := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{
		Zones: []string{"zone-1"},
	})
	require.NoError(t, err)

	maintWindowConfigPath := filepath.Join("testdata", "maintwindow.json")

	cluster := newClusterConfig("test-cluster", nil, gcpGardenerConfig, region, purpose)

	t.Run("should start provisioning", func(t *testing.T) {
		// given
		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		provisionerClient := NewProvisioner(gardenerNamespace, shootClient, nil, auditLogsPolicyCMName, maintWindowConfigPath)

		// when
		apperr := provisionerClient.ProvisionCluster(cluster, operationId)
		require.NoError(t, apperr)

		// then
		shoot, err := shootClient.Get(context.Background(), clusterName, v1.GetOptions{})
		require.NoError(t, err)
		assertAnnotation(t, shoot, operationIDAnnotation, operationId)
		assertAnnotation(t, shoot, runtimeIDAnnotation, runtimeId)
		assertAnnotation(t, shoot, legacyOperationIDAnnotation, operationId)
		assertAnnotation(t, shoot, legacyRuntimeIDAnnotation, runtimeId)
		assert.Equal(t, "", shoot.Labels[model.SubAccountLabel])

		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy.ConfigMapRef)
		require.NotNil(t, shoot.Spec.Maintenance.TimeWindow)
		assert.Equal(t, auditLogsPolicyCMName, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy.ConfigMapRef.Name)
	})
}

func TestGardenerProvisioner_DeprovisionCluster(t *testing.T) {

	gcpGardenerConfig, err := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{})
	require.NoError(t, err)

	cluster := model.Cluster{
		ID: runtimeId,
		ClusterConfig: model.GardenerConfig{
			ID:                     "id",
			ClusterID:              runtimeId,
			Name:                   clusterName,
			ProjectName:            "project-name",
			GardenerProviderConfig: gcpGardenerConfig,
		},
		ActiveKymaConfigId: util.StringPtr("activekymaconfigid"),
	}

	t.Run("should start deprovisioning", func(t *testing.T) {
		// given
		clientset := fake.NewSimpleClientset(
			&gardener_types.Shoot{
				ObjectMeta: v1.ObjectMeta{Name: clusterName, Namespace: gardenerNamespace, Finalizers: []string{"test"}},
			})

		sessionFactoryMock := &sessionMocks.Factory{}
		session := &sessionMocks.WriteSession{}

		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		provisionerClient := NewProvisioner(gardenerNamespace, shootClient, sessionFactoryMock, auditLogsPolicyCMName, "")

		// when
		sessionFactoryMock.On("NewWriteSession").Return(session)

		operation, apperr := provisionerClient.DeprovisionCluster(cluster, false, operationId)
		require.NoError(t, apperr)

		// then
		assert.Equal(t, model.InProgress, operation.State)
		assert.Equal(t, operationId, operation.ID)
		assert.Equal(t, runtimeId, operation.ClusterID)
		assert.Equal(t, model.Deprovision, operation.Type)

		_, err := shootClient.Get(context.Background(), clusterName, v1.GetOptions{})
		assert.NoError(t, err)
	})

	t.Run("should start deprovisioning without uninstallation", func(t *testing.T) {
		// given
		clientset := fake.NewSimpleClientset(
			&gardener_types.Shoot{
				ObjectMeta: v1.ObjectMeta{Name: clusterName, Namespace: gardenerNamespace, Finalizers: []string{"test"}},
			})

		sessionFactoryMock := &sessionMocks.Factory{}
		session := &sessionMocks.WriteSession{}

		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		provisionerClient := NewProvisioner(gardenerNamespace, shootClient, sessionFactoryMock, auditLogsPolicyCMName, "")

		// when
		sessionFactoryMock.On("NewWriteSession").Return(session)

		operation, apperr := provisionerClient.DeprovisionCluster(cluster, true, operationId)
		require.NoError(t, apperr)

		// then
		assert.Equal(t, model.InProgress, operation.State)
		assert.Equal(t, operationId, operation.ID)
		assert.Equal(t, runtimeId, operation.ClusterID)
		assert.Equal(t, model.DeprovisionNoInstall, operation.Type)

		_, err := shootClient.Get(context.Background(), clusterName, v1.GetOptions{})
		assert.NoError(t, err)
	})

	t.Run("should proceed to WaitForClusterDeletion step if shoot does not exist", func(t *testing.T) {
		// given
		clientset := fake.NewSimpleClientset()

		sessionFactoryMock := &sessionMocks.Factory{}
		session := &sessionMocks.WriteSession{}

		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		provisionerClient := NewProvisioner(gardenerNamespace, shootClient, sessionFactoryMock, auditLogsPolicyCMName, "")

		// when
		sessionFactoryMock.On("NewWriteSession").Return(session)
		session.On("MarkClusterAsDeleted", cluster.ID).Return(nil)

		operation, apperr := provisionerClient.DeprovisionCluster(cluster, false, operationId)
		require.NoError(t, apperr)

		// then
		assert.Equal(t, model.InProgress, operation.State)
		assert.Equal(t, model.WaitForClusterDeletion, operation.Stage)
		assert.Equal(t, operationId, operation.ID)
		assert.Equal(t, runtimeId, operation.ClusterID)
		assert.Equal(t, model.Deprovision, operation.Type)

		_, err := shootClient.Get(context.Background(), clusterName, v1.GetOptions{})
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestGardenerProvisioner_UpgradeCluster(t *testing.T) {
	initialShoot := testkit.NewTestShoot(clusterName).
		InNamespace(gardenerNamespace).
		WithAutoUpdate(false, false).
		WithWorkers(testkit.NewTestWorker("peon").ToWorker()).
		ToShoot()

	expectedShoot := testkit.NewTestShoot(clusterName).
		InNamespace(gardenerNamespace).
		WithKubernetesVersion("1.16").
		WithAutoUpdate(false, false).
		WithPurpose(purpose).
		WithWorkers(
			testkit.NewTestWorker("peon").
				WithMachineType("n1-standard-4").
				WithVolume("standard", 50).
				WithMinMax(1, 5).
				WithMaxSurge(25).
				WithMaxUnavailable(1).
				WithZones("zone-1").
				ToWorker()).
		ToShoot()

	gcpGardenerConfig, err := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"zone-1"}})
	require.NoError(t, err)
	cluster := newClusterConfig(clusterName, nil, gcpGardenerConfig, region, purpose)

	t.Run("should upgrade shoot", func(t *testing.T) {
		// given
		clientset := fake.NewSimpleClientset(initialShoot)
		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		sessionFactory := &sessionMocks.Factory{}
		provisioner := NewProvisioner(gardenerNamespace, shootClient, sessionFactory, auditLogsPolicyCMName, "")

		// when
		apperr := provisioner.UpgradeCluster(cluster.ID, cluster.ClusterConfig)
		require.NoError(t, apperr)

		// then
		shoot, err := shootClient.Get(context.Background(), clusterName, v1.GetOptions{})
		require.NoError(t, err)

		assert.Equal(t, expectedShoot, shoot)
	})
	t.Run("should return error when failed to get shoot from Gardener", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		sessionFactory := &sessionMocks.Factory{}
		provisioner := NewProvisioner(gardenerNamespace, shootClient, sessionFactory, auditLogsPolicyCMName, "")

		// when
		apperr := provisioner.UpgradeCluster(cluster.ID, cluster.ClusterConfig)

		// then
		require.Error(t, apperr)
		assert.Equal(t, apperrors.CodeInternal, apperr.Code())
	})
}

func newClusterConfig(name string, subAccountID *string, providerConfig model.GardenerProviderConfig, region string, purpose string) model.Cluster {
	return model.Cluster{
		ID:           runtimeId,
		Tenant:       tenant,
		SubAccountId: subAccountID,
		ClusterConfig: model.GardenerConfig{
			ID:                     "id",
			ClusterID:              runtimeId,
			Name:                   name,
			ProjectName:            "project-name",
			KubernetesVersion:      "1.16",
			VolumeSizeGB:           util.IntPtr(50),
			DiskType:               util.StringPtr("standard"),
			MachineType:            "n1-standard-4",
			Provider:               "gcp",
			TargetSecret:           "secret",
			Region:                 region,
			Purpose:                util.StringPtr(purpose),
			WorkerCidr:             "10.10.10.10",
			AutoScalerMin:          1,
			AutoScalerMax:          5,
			MaxSurge:               25,
			MaxUnavailable:         1,
			GardenerProviderConfig: providerConfig,
		},
	}
}

func TestGardenerProvisioner_HibernateCluster(t *testing.T) {

	gcpGardenerConfig, err := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"zone-1"}})
	require.NoError(t, err)
	cluster := newClusterConfig(clusterName, nil, gcpGardenerConfig, region, purpose)

	t.Run("should return error if failed to get shoot", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		sessionFactory := &sessionMocks.Factory{}
		provisioner := NewProvisioner(gardenerNamespace, shootClient, sessionFactory, auditLogsPolicyCMName, "")

		// when
		apperr := provisioner.HibernateCluster(cluster.ID, cluster.ClusterConfig)

		// then
		require.Error(t, apperr)
		assert.Equal(t, apperrors.CodeInternal, apperr.Code())
	})

	t.Run("should return error if cluster cannot be hibernated", func(t *testing.T) {
		shoot := testkit.NewTestShoot(clusterName).
			InNamespace(gardenerNamespace).
			WithHibernationState(false, false).
			ToShoot()

		clientset := fake.NewSimpleClientset(shoot)
		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		sessionFactory := &sessionMocks.Factory{}
		provisioner := NewProvisioner(gardenerNamespace, shootClient, sessionFactory, auditLogsPolicyCMName, "")

		// when
		apperr := provisioner.HibernateCluster(cluster.ID, cluster.ClusterConfig)

		// then
		require.Error(t, apperr)
		assert.Equal(t, apperrors.CodeBadRequest, apperr.Code())

	})

	t.Run("should hibernate cluster", func(t *testing.T) {
		shoot := testkit.NewTestShoot(clusterName).
			InNamespace(gardenerNamespace).
			WithHibernationState(true, false).
			ToShoot()

		clientset := fake.NewSimpleClientset(shoot)
		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		sessionFactory := &sessionMocks.Factory{}
		provisioner := NewProvisioner(gardenerNamespace, shootClient, sessionFactory, auditLogsPolicyCMName, "")

		// when
		apperr := provisioner.HibernateCluster(cluster.ID, cluster.ClusterConfig)

		// then
		require.NoError(t, apperr)
	})

	t.Run("should return error if failed to hibernate cluster", func(t *testing.T) {
		shoot := testkit.NewTestShoot(clusterName).
			InNamespace(gardenerNamespace).
			WithHibernationState(true, false).
			ToShoot()

		shootClient := &gardenerMocks.Client{}

		shootClient.On("Get", mock.Anything, clusterName, mock.Anything).Return(shoot, nil)
		shootClient.On("Update", mock.Anything, shoot, mock.Anything).Return(nil, errors.New("some error"))

		sessionFactory := &sessionMocks.Factory{}
		provisioner := NewProvisioner(gardenerNamespace, shootClient, sessionFactory, auditLogsPolicyCMName, "")

		// when
		apperr := provisioner.HibernateCluster(cluster.ID, cluster.ClusterConfig)

		// then
		require.Error(t, apperr)
	})
}

func TestGardenerProvisioner_GetHibernationStatus(t *testing.T) {
	gcpGardenerConfig, err := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"zone-1"}})
	require.NoError(t, err)
	cluster := newClusterConfig(clusterName, nil, gcpGardenerConfig, region, purpose)

	t.Run("should  fail when failed to get cluster", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

		sessionFactory := &sessionMocks.Factory{}
		provisioner := NewProvisioner(gardenerNamespace, shootClient, sessionFactory, auditLogsPolicyCMName, "")

		// when
		_, apperr := provisioner.GetHibernationStatus(cluster.ID, cluster.ClusterConfig)

		// then
		require.Error(t, apperr)
		assert.Equal(t, apperrors.CodeInternal, apperr.Code())
	})

	for _, testcase := range []struct {
		description         string
		shoot               *gardener_types.Shoot
		hibernationPossible bool
		hibernated          bool
	}{
		{
			description:         "should get status when hibernation impossible",
			hibernationPossible: false,
			hibernated:          false,
		},
		{
			description:         "should get status when hibernation possible",
			hibernationPossible: true,
			hibernated:          true,
		},
	} {
		t.Run(testcase.description, func(t *testing.T) {
			// given
			shoot := testkit.NewTestShoot(clusterName).
				InNamespace(gardenerNamespace).
				WithHibernationState(testcase.hibernationPossible, testcase.hibernated).
				ToShoot()

			clientset := fake.NewSimpleClientset(shoot)
			shootClient := clientset.CoreV1beta1().Shoots(gardenerNamespace)

			sessionFactory := &sessionMocks.Factory{}
			provisioner := NewProvisioner(gardenerNamespace, shootClient, sessionFactory, auditLogsPolicyCMName, "")

			// when
			status, apperr := provisioner.GetHibernationStatus(cluster.ID, cluster.ClusterConfig)

			// then
			require.NoError(t, apperr)
			require.Equal(t, testcase.hibernationPossible, status.HibernationPossible)
			require.Equal(t, testcase.hibernated, status.Hibernated)
		})
	}
}

func TestGardenerProvisioner_ClusterPurpose(t *testing.T) {
	clientset_A := fake.NewSimpleClientset()
	clientset_B := fake.NewSimpleClientset()

	gcpGardenerConfig, err := model.NewGCPGardenerConfig(&gqlschema.GCPProviderConfigInput{Zones: []string{"zone-1"}})
	require.NoError(t, err)
	cluster_A := newClusterConfig(clusterName, nil, gcpGardenerConfig, region, "")
	cluster_B := newClusterConfig(clusterName, nil, gcpGardenerConfig, region, purpose)

	maintWindowConfigPath := filepath.Join("testdata", "maintwindow.json")

	t.Run("should start provisioning with 2 clusters with different purpose", func(t *testing.T) {
		shootClient_A := clientset_A.CoreV1beta1().Shoots(gardenerNamespace)
		provisionerClient_A := NewProvisioner(gardenerNamespace, shootClient_A, nil, auditLogsPolicyCMName, maintWindowConfigPath)

		shootClient_B := clientset_B.CoreV1beta1().Shoots(gardenerNamespace)
		provisionerClient_B := NewProvisioner(gardenerNamespace, shootClient_B, nil, auditLogsPolicyCMName, maintWindowConfigPath)

		//when
		apperr_A := provisionerClient_A.ProvisionCluster(cluster_A, operationId)
		require.NoError(t, apperr_A)
		apperr_B := provisionerClient_B.ProvisionCluster(cluster_B, operationId)
		require.NoError(t, apperr_B)

		//then
		shoot_A, err := shootClient_A.Get(context.Background(), clusterName, v1.GetOptions{})
		require.NoError(t, err)
		shoot_B, err := shootClient_B.Get(context.Background(), clusterName, v1.GetOptions{})
		require.NoError(t, err)
		assert.NotEqual(t, shoot_A.Spec.Maintenance.TimeWindow, shoot_B.Spec.Maintenance.TimeWindow)
	})
}

func assertAnnotation(t *testing.T, shoot *gardener_types.Shoot, name, value string) {
	annotations := shoot.Annotations
	if annotations == nil {
		t.Errorf("annotations are nil, expected annotation: %s, value: %s", name, value)
		return
	}

	val, found := annotations[name]
	if !found {
		t.Errorf("annotation not found, expected annotation: %s, value: %s", name, value)
		return
	}

	assert.Equal(t, value, val, fmt.Sprintf("invalid value for %s annotation", name))
}
