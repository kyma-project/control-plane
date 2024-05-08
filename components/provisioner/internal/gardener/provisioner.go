package gardener

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	log "github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	"gopkg.in/yaml.v3"
)

//go:generate mockery --name=Client
type Client interface {
	Create(ctx context.Context, shoot *v1beta1.Shoot, opts v1.CreateOptions) (*v1beta1.Shoot, error)
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta1.Shoot, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.Shoot, err error)
}

func NewProvisioner(
	namespace string,
	shootClient Client,
	factory dbsession.Factory,
	policyConfigMapName string,
	maintenanceWindowConfigPath string,
	enableDumpShootSpec bool) *GardenerProvisioner {
	return &GardenerProvisioner{
		namespace:                   namespace,
		shootClient:                 shootClient,
		dbSessionFactory:            factory,
		policyConfigMapName:         policyConfigMapName,
		maintenanceWindowConfigPath: maintenanceWindowConfigPath,
		enableDumpShootSpec:         enableDumpShootSpec,
	}
}

type GardenerProvisioner struct {
	namespace                   string
	shootClient                 Client
	dbSessionFactory            dbsession.Factory
	policyConfigMapName         string
	maintenanceWindowConfigPath string
	enableDumpShootSpec         bool
}

func (g *GardenerProvisioner) ProvisionCluster(cluster model.Cluster, operationId string) apperrors.AppError {
	shootTemplate, err := cluster.ClusterConfig.ToShootTemplate(g.namespace, cluster.Tenant, util.UnwrapOrZero(cluster.SubAccountId), cluster.ClusterConfig.OIDCConfig, cluster.ClusterConfig.DNSConfig)
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

	if g.enableDumpShootSpec {
		log.Infof("Shoot Spec Dump Start ===============================")

		shootTemplateBytes, e := yaml.Marshal(&shootTemplate)

		if e != nil {
			log.Errorf("Error marshaling Shoot spec: %s", e.Error())
		} else {
			log.Info(string(shootTemplateBytes))
		}

		log.Infof("Shoot Spec Dump End =================================")
	}

	_, k8serr := g.shootClient.Create(context.Background(), shootTemplate, v1.CreateOptions{})
	if k8serr != nil {
		appError := util.K8SErrorToAppError(k8serr).SetComponent(apperrors.ErrGardenerClient)
		return appError.Append("error creating Shoot for %s cluster: %s", cluster.ID)
	}

	return nil
}

func (g *GardenerProvisioner) UpgradeCluster(clusterID string, upgradeConfig model.GardenerConfig) apperrors.AppError {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		shoot, err := g.shootClient.Get(context.Background(), upgradeConfig.Name, v1.GetOptions{})
		if err != nil {
			appErr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
			return appErr.Append("error getting Shoot for cluster ID %s and name %s", clusterID, upgradeConfig.Name)
		}

		appErr := upgradeConfig.GardenerProviderConfig.EditShootConfig(upgradeConfig, shoot)

		if appErr != nil {
			return appErr.Append("error while updating Gardener shoot configuration")
		}

		setObjectFields(shoot)

		shootData, err := json.Marshal(shoot)
		if err != nil {
			apperr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrProvisioner)
			return apperr.Append("error during marshaling Shoot data")
		}

		_, err = g.shootClient.Patch(context.Background(), shoot.Name, types.ApplyPatchType, shootData, v1.PatchOptions{FieldManager: "provisioner", Force: util.PtrTo(true)})
		return err
	})
	if err != nil {
		apperr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
		return apperr.Append("error executing update shoot configuration")
	}

	return nil
}

func (g *GardenerProvisioner) DeprovisionCluster(cluster model.Cluster, operationId string) (model.Operation, apperrors.AppError) {
	shoot, err := g.shootClient.Get(context.Background(), cluster.ClusterConfig.Name, v1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			message := fmt.Sprintf("Cluster %s already deleted. Proceeding to DeprovisionCluster stage.", cluster.ID)

			return newDeprovisionOperation(operationId, cluster.ID, message, model.InProgress, model.WaitForClusterDeletion, time.Now()), nil
		}
	}

	if shoot.DeletionTimestamp != nil {
		annotate(shoot, operationIDAnnotation, operationId)
		annotate(shoot, legacyOperationIDAnnotation, operationId)
		message := fmt.Sprintf("Cluster %s with id %s already scheduled for deletion.", cluster.ClusterConfig.Name, cluster.ID)

		return newDeprovisionOperation(operationId, cluster.ID, message, model.InProgress, model.DeleteCluster, shoot.DeletionTimestamp.Time), nil
	}

	deletionTime := time.Now()

	annotate(shoot, operationIDAnnotation, operationId)
	annotate(shoot, legacyOperationIDAnnotation, operationId)

	annotateWithConfirmDeletion(shoot)

	setObjectFields(shoot)

	shootData, err := json.Marshal(shoot)
	if err != nil {
		apperr := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrProvisioner)
		return model.Operation{}, apperr.Append("error during marshaling Shoot data")
	}
	_, err = g.shootClient.Patch(context.Background(), shoot.Name, types.ApplyPatchType, shootData, v1.PatchOptions{FieldManager: "provisioner", Force: util.PtrTo(true)})

	if err != nil {
		appError := util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
		return model.Operation{}, appError.Append("error updating Shoot")
	}

	message := "Deprovisioning started"
	return newDeprovisionOperation(operationId, cluster.ID, message, model.InProgress, model.DeleteCluster, deletionTime), nil
}

func annotateWithConfirmDeletion(shoot *v1beta1.Shoot) {
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
		Type:           model.DeprovisionNoInstall,
		StartTimestamp: startTime,
		State:          state,
		Stage:          stage,
		Message:        message,
		ClusterID:      runtimeId,
	}
}

func (g *GardenerProvisioner) applyAuditConfig(template *v1beta1.Shoot) {
	if template.Spec.Kubernetes.KubeAPIServer == nil {
		template.Spec.Kubernetes.KubeAPIServer = &v1beta1.KubeAPIServerConfig{}
	}

	template.Spec.Kubernetes.KubeAPIServer.AuditConfig = &v1beta1.AuditConfig{
		AuditPolicy: &v1beta1.AuditPolicy{
			ConfigMapRef: &v12.ObjectReference{Name: g.policyConfigMapName},
		},
	}
}

func (g *GardenerProvisioner) setMaintenanceWindow(template *v1beta1.Shoot, region string) apperrors.AppError {
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

// workaround
func setObjectFields(shoot *v1beta1.Shoot) {
	shoot.Kind = "Shoot"
	shoot.APIVersion = "core.gardener.cloud/v1beta1"
	shoot.ManagedFields = nil
}

func setMaintenanceWindow(window TimeWindow, template *v1beta1.Shoot) {
	template.Spec.Maintenance.TimeWindow = &v1beta1.MaintenanceTimeWindow{Begin: window.Begin, End: window.End}
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
