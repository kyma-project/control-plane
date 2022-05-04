package gardener

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/mitchellh/mapstructure"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v12 "k8s.io/api/core/v1"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/kyma-project/control-plane/components/provisioner/internal/director"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
)

//go:generate mockery -name=Client
type Client interface {
	Create(ctx context.Context, shoot *v1beta1.Shoot, opts v1.CreateOptions) (*v1beta1.Shoot, error)
	Update(ctx context.Context, shoot *v1beta1.Shoot, opts v1.UpdateOptions) (*v1beta1.Shoot, error)
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta1.Shoot, error)
}

func NewProvisioner(
	namespace string,
	shootClient Client,
	factory dbsession.Factory,
	policyConfigMapName string, maintenanceWindowConfigPath string) *GardenerProvisioner {
	return &GardenerProvisioner{
		namespace:                   namespace,
		shootClient:                 shootClient,
		dbSessionFactory:            factory,
		policyConfigMapName:         policyConfigMapName,
		maintenanceWindowConfigPath: maintenanceWindowConfigPath,
	}
}

type GardenerProvisioner struct {
	namespace                   string
	shootClient                 Client
	dbSessionFactory            dbsession.Factory
	directorService             director.DirectorClient
	policyConfigMapName         string
	maintenanceWindowConfigPath string
}

func (g *GardenerProvisioner) ProvisionCluster(cluster model.Cluster, operationId string) apperrors.AppError {
	shootTemplate, err := cluster.ClusterConfig.ToShootTemplate(g.namespace, cluster.Tenant, util.UnwrapStr(cluster.SubAccountId), cluster.ClusterConfig.OIDCConfig, cluster.ClusterConfig.DNSConfig)
	if err != nil {
		return err.Append("failed to convert cluster config to Shoot template")
	}

	region := cluster.ClusterConfig.Region
	purpose := ""
	if cluster.ClusterConfig.Purpose != nil {
		purpose = *cluster.ClusterConfig.Purpose
	}

	if g.shouldSetMaintenanceWindow(purpose) {
		err := g.setMaintenanceWindow(shootTemplate, region)

		if err != nil {
			return err.Append("error setting maintenance window for %s cluster", cluster.ID)
		}
	}

	annotate(shootTemplate, runtimeIDAnnotation, cluster.ID)
	annotate(shootTemplate, operationIDAnnotation, operationId)
	annotate(shootTemplate, legacyRuntimeIDAnnotation, cluster.ID)
	annotate(shootTemplate, legacyOperationIDAnnotation, operationId)

	if g.policyConfigMapName != "" {
		g.applyAuditConfig(shootTemplate)
	}

	_, k8serr := g.shootClient.Create(context.Background(), shootTemplate, v1.CreateOptions{})
	if k8serr != nil {
		appError := util.K8SErrorToAppError(k8serr).SetComponent(apperrors.ErrGardenerClient)
		return appError.Append("error creating Shoot for %s cluster: %s", cluster.ID)
	}

	return nil
}

func (g *GardenerProvisioner) UpgradeCluster(clusterID string, upgradeConfig model.GardenerConfig) apperrors.AppError {

	shoot, err := g.shootClient.Get(context.Background(), upgradeConfig.Name, v1.GetOptions{})
	if err != nil {
		appErr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
		return appErr.Append("error getting Shoot for cluster ID %s and name %s", clusterID, upgradeConfig.Name)
	}

	appErr := upgradeConfig.GardenerProviderConfig.EditShootConfig(upgradeConfig, shoot)

	if appErr != nil {
		return appErr.Append("error while updating Gardener shoot configuration")
	}

	err = retry.Do(func() error {
		_, err := g.shootClient.Update(context.Background(), shoot, v1.UpdateOptions{})
		return err
	}, retry.Attempts(5))
	if err != nil {
		apperr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
		return apperr.Append("error executing update shoot configuration")
	}

	return nil
}

func (g *GardenerProvisioner) HibernateCluster(clusterID string, gardenerConfig model.GardenerConfig) apperrors.AppError {
	shoot, err := g.shootClient.Get(context.Background(), gardenerConfig.Name, v1.GetOptions{})
	if err != nil {
		appErr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
		return appErr.Append("error getting Shoot for cluster ID %s and name %s", clusterID, gardenerConfig.Name)
	}

	condition := gardencorev1beta1helper.GetOrInitCondition(shoot.Status.Constraints, v1beta1.ShootHibernationPossible)
	if condition.Status == v1beta1.ConditionFalse {
		return apperrors.BadRequest(fmt.Sprintf("cannot hibernate cluster: %s", condition.Message))
	}

	enabled := true
	if shoot.Spec.Hibernation != nil {
		shoot.Spec.Hibernation.Enabled = &enabled
	} else {
		shoot.Spec.Hibernation = &v1beta1.Hibernation{
			Enabled: &enabled,
		}
	}

	err = retry.Do(func() error {
		_, err := g.shootClient.Update(context.Background(), shoot, v1.UpdateOptions{})
		return err
	}, retry.Attempts(5))

	if err != nil {
		apperr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
		return apperr.Append("error executing update shoot configuration")
	}

	return nil
}

func (g *GardenerProvisioner) DeprovisionCluster(cluster model.Cluster, withoutUninstall bool, operationId string) (model.Operation, apperrors.AppError) {
	shoot, err := g.shootClient.Get(context.Background(), cluster.ClusterConfig.Name, v1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			message := fmt.Sprintf("Cluster %s already deleted. Proceeding to DeprovisionCluster stage.", cluster.ID)

			// Shoot was deleted. In order to make sure if all clean up actions were performed we need to proceed to WaitForClusterDeletion state
			if withoutUninstall {
				return newDeprovisionOperationNoInstall(operationId, cluster.ID, message, model.InProgress, model.DeleteCluster, time.Now()), nil
			}
			return newDeprovisionOperation(operationId, cluster.ID, message, model.InProgress, model.WaitForClusterDeletion, time.Now()), nil
		}
	}

	if shoot.DeletionTimestamp != nil {
		annotate(shoot, operationIDAnnotation, operationId)
		annotate(shoot, legacyOperationIDAnnotation, operationId)
		message := fmt.Sprintf("Cluster %s with id %s already scheduled for deletion.", cluster.ClusterConfig.Name, cluster.ID)
		if withoutUninstall {
			return newDeprovisionOperationNoInstall(operationId, cluster.ID, message, model.InProgress, model.DeleteCluster, shoot.DeletionTimestamp.Time), nil
		}
		return newDeprovisionOperation(operationId, cluster.ID, message, model.InProgress, model.WaitForClusterDeletion, shoot.DeletionTimestamp.Time), nil
	}

	deletionTime := time.Now()

	annotate(shoot, operationIDAnnotation, operationId)
	annotate(shoot, legacyOperationIDAnnotation, operationId)

	annotateWithConfirmDeletion(shoot)

	_, err = g.shootClient.Update(context.Background(), shoot, v1.UpdateOptions{})
	if err != nil {
		appError := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
		return model.Operation{}, appError.Append("error updating Shoot")
	}

	message := fmt.Sprintf("Deprovisioning started")

	if withoutUninstall {
		return newDeprovisionOperationNoInstall(operationId, cluster.ID, message, model.InProgress, model.DeleteCluster, deletionTime), nil
	}
	return newDeprovisionOperation(operationId, cluster.ID, message, model.InProgress, model.CleanupCluster, deletionTime), nil
}

func (g *GardenerProvisioner) GetHibernationStatus(clusterID string, gardenerConfig model.GardenerConfig) (model.HibernationStatus, apperrors.AppError) {
	shoot, err := g.shootClient.Get(context.Background(), gardenerConfig.Name, v1.GetOptions{})
	if err != nil {
		appErr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
		return model.HibernationStatus{}, appErr.Append("error getting Shoot for cluster ID %s and name %s", clusterID, gardenerConfig.Name)
	}

	condition := gardencorev1beta1helper.GetOrInitCondition(shoot.Status.Constraints, v1beta1.ShootHibernationPossible)

	return model.HibernationStatus{
		Hibernated:          shoot.Status.IsHibernated,
		HibernationPossible: condition.Status == v1beta1.ConditionTrue,
	}, nil
}

func annotateWithConfirmDeletion(shoot *gardener_types.Shoot) {
	if shoot.Annotations == nil {
		shoot.Annotations = map[string]string{}
	}

	shoot.Annotations["confirmation.gardener.cloud/deletion"] = "true"
}

func (g *GardenerProvisioner) shouldSetMaintenanceWindow(purpose string) bool {
	return g.maintenanceWindowConfigPath != "" && purpose == "production"
}

func newDeprovisionOperation(id, runtimeId, message string, state model.OperationState, stage model.OperationStage, startTime time.Time) model.Operation {
	return model.Operation{
		ID:             id,
		Type:           model.Deprovision,
		StartTimestamp: startTime,
		State:          state,
		Stage:          stage,
		Message:        message,
		ClusterID:      runtimeId,
	}
}

func newDeprovisionOperationNoInstall(id, runtimeId, message string, state model.OperationState, stage model.OperationStage, startTime time.Time) model.Operation {
	return model.Operation{
		ID:             id,
		Type:           model.DeprovisionNoInstall,
		StartTimestamp: startTime,
		State:          state,
		Stage:          stage,
		Message:        message,
		ClusterID:      runtimeId,
	}
}

func (g *GardenerProvisioner) applyAuditConfig(template *gardener_types.Shoot) {
	if template.Spec.Kubernetes.KubeAPIServer == nil {
		template.Spec.Kubernetes.KubeAPIServer = &gardener_types.KubeAPIServerConfig{}
	}

	template.Spec.Kubernetes.KubeAPIServer.AuditConfig = &gardener_types.AuditConfig{
		AuditPolicy: &gardener_types.AuditPolicy{
			ConfigMapRef: &v12.ObjectReference{Name: g.policyConfigMapName},
		},
	}
}

func (g *GardenerProvisioner) setMaintenanceWindow(template *gardener_types.Shoot, region string) apperrors.AppError {
	window, err := g.getWindowByRegion(region)

	if err != nil {
		return err
	}

	if !window.isEmpty() {
		setMaintenanceWindow(window, template)
	} else {
		logrus.Warnf("Cannot set maintenance window. Config for region %s is empty", region)
	}
	return nil
}

func setMaintenanceWindow(window TimeWindow, template *gardener_types.Shoot) {
	template.Spec.Maintenance.TimeWindow = &gardener_types.MaintenanceTimeWindow{Begin: window.Begin, End: window.End}
}

func (g *GardenerProvisioner) getWindowByRegion(region string) (TimeWindow, apperrors.AppError) {
	data, err := getDataFromFile(g.maintenanceWindowConfigPath, region)

	if err != nil {
		return TimeWindow{}, err
	}

	var window TimeWindow

	mapErr := mapstructure.Decode(data, &window)

	if mapErr != nil {
		return TimeWindow{}, apperrors.Internal("failed to parse map to struct: %s", mapErr.Error())
	}

	return window, nil
}

type TimeWindow struct {
	Begin string
	End   string
}

func (tw TimeWindow) isEmpty() bool {
	return tw.Begin == "" || tw.End == ""
}

func getDataFromFile(filepath, region string) (interface{}, apperrors.AppError) {
	file, err := os.Open(filepath)

	if err != nil {
		return "", apperrors.Internal("failed to open file: %s", err.Error())
	}

	defer file.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return "", apperrors.Internal("failed to decode json: %s", err.Error())
	}
	return data[region], nil
}
