package btpmgrcreds

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/sirupsen/logrus"
	apicorev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	BtpManagerSecretName      = "sap-btp-manager"
	BtpManagerSecretNamespace = "kyma-system"
)

var (
	BtpManagerLabels      = map[string]string{"app.kubernetes.io/managed-by": keb, "app.kubernetes.io/watched-by": keb}
	BtpManagerAnnotations = map[string]string{"Warning": "This secret is generated. Do not edit!"}
	KymaGvk               = schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "Kyma"}
)

const (
	keb             = "kcp-kyma-environment-broker"
	kcpNamespace    = "kcp-system"
	instanceIdLabel = "kyma-project.io/instance-id"
)

const (
	secretClientId     = "clientid"
	secretClientSecret = "clientsecret"
	secretSmUrl        = "sm_url"
	secretTokenUrl     = "tokenurl"
	secretClusterId    = "cluster_id"
)

type Manager struct {
	ctx          context.Context
	instances    storage.Instances
	kcpK8sClient client.Client
	dryRun       bool
	provisioner  provisioner.Client
	logger       *logrus.Logger
}

func NewManager(ctx context.Context, kcpK8sClient client.Client, instanceDb storage.Instances, logs *logrus.Logger, dryRun bool, provisioner provisioner.Client) *Manager {
	return &Manager{
		ctx:          ctx,
		instances:    instanceDb,
		kcpK8sClient: kcpK8sClient,
		dryRun:       dryRun,
		provisioner:  provisioner,
		logger:       logs,
	}
}

func (s *Manager) MatchInstance(kymaName string) (*internal.Instance, error) {
	kyma := &unstructured.Unstructured{}
	kyma.SetGroupVersionKind(KymaGvk)
	err := s.kcpK8sClient.Get(s.ctx, client.ObjectKey{
		Namespace: kcpNamespace,
		Name:      kymaName,
	}, kyma)
	if err != nil && errors.IsNotFound(err) {
		s.logger.Errorf("not found secret with name %s on cluster : %s", kymaName, err)
		return nil, err
	} else if err != nil {
		s.logger.Errorf("unexpected error while getting secret %s from cluster : %s", kymaName, err)
		return nil, err
	}
	s.logger.Infof("found kyma CR on kcp for kyma name: %s", kymaName)
	labels := kyma.GetLabels()
	instanceId, ok := labels[instanceIdLabel]
	if !ok {
		s.logger.Errorf("not found instance for kyma name %s : %s", kymaName, err)
		return nil, err
	}
	s.logger.Infof("found instance id %s for kyma name %s", instanceId, kymaName)
	instance, err := s.instances.GetByID(instanceId)
	if err != nil {
		s.logger.Errorf("while getting instance %s from db %s", instanceId, err)
		return nil, err
	}
	s.logger.Infof("instance %s found in db", instance.InstanceID)
	return instance, err
}

func (s *Manager) ReconcileAll() (int, int, int, int, error) {
	instances, err := s.GetReconcileCandidates()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	s.logger.Infof("processing %d instances as candidates", len(instances))

	updateDone, updateNotDoneDueError, updateNotDoneDueOkState := 0, 0, 0
	for _, instance := range instances {
		updated, err := s.ReconcileSecretForInstance(&instance)
		if err != nil {
			s.logger.Errorf("while doing update, for instance: %s, %s", instance.InstanceID, err)
			updateNotDoneDueError++
			continue
		}
		if updated {
			s.logger.Infof("update done for instance %s", instance.InstanceID)
			updateDone++
		} else {
			s.logger.Infof("no need to update instance %s", instance.InstanceID)
			updateNotDoneDueOkState++
		}
	}
	s.logger.Infof("(runtime-reconciler summary) from total %d instances: %d are OK, update was needed (and done with success) for %d instances, errors occur for %d instances",
		len(instances), updateNotDoneDueOkState, updateDone, updateNotDoneDueError)
	return len(instances), updateDone, updateNotDoneDueError, updateNotDoneDueOkState, nil
}

func (s *Manager) GetReconcileCandidates() ([]internal.Instance, error) {
	allInstances, _, _, err := s.instances.List(dbmodel.InstanceFilter{})
	if err != nil {
		return nil, fmt.Errorf("while getting all instances %s", err)
	}
	s.logger.Infof("total number of instances in db: %d", len(allInstances))

	var instancesWithinRuntime []internal.Instance
	for _, instance := range allInstances {
		if !instance.Reconcilable {
			s.logger.Infof("skipping instance %s because it is not reconilable (no runtimeId,last op was deprovisoning or op is in progress)", instance.InstanceID)
			continue
		}

		if instance.Parameters.ErsContext.SMOperatorCredentials == nil || instance.InstanceDetails.ServiceManagerClusterID == "" {
			s.logger.Warnf("skipping instance %s because there are no needed data attached to instance", instance.InstanceID)
			continue
		}

		instancesWithinRuntime = append(instancesWithinRuntime, instance)
		s.logger.Infof("adding instance %s as candidate for reconcilation", instance.InstanceID)
	}

	s.logger.Infof("from total number of instances (%d) took %d as candidates", len(allInstances), len(instancesWithinRuntime))
	return instancesWithinRuntime, nil
}

func (s *Manager) ReconcileSecretForInstance(instance *internal.Instance) (bool, error) {
	s.logger.Infof("reconcilation of btp-manager secret started for %s", instance.InstanceID)

	futureSecret, err := PrepareSecret(instance.Parameters.ErsContext.SMOperatorCredentials, instance.InstanceDetails.ServiceManagerClusterID)
	if err != nil {
		return false, err
	}

	k8sClient, err := s.getSkrK8sClient(instance)
	if err != nil {
		return false, fmt.Errorf("while getting k8sClient for %s : %w", instance.InstanceID, err)
	}
	s.logger.Infof("connected to skr with success for instance %s", instance.InstanceID)

	currentSecret := &v1.Secret{}
	err = k8sClient.Get(context.Background(), client.ObjectKey{Name: BtpManagerSecretName, Namespace: BtpManagerSecretNamespace}, currentSecret)
	if err != nil && errors.IsNotFound(err) {
		s.logger.Infof("not found btp-manager secret on cluster for instance: %s", instance.InstanceID)
		if s.dryRun {
			s.logger.Infof("[dry-run] secret for instance %s would be created", instance.InstanceID)
		} else {
			if err := CreateOrUpdateSecret(k8sClient, futureSecret, s.logger); err != nil {
				s.logger.Errorf("while creating secret in cluster for %s", instance.InstanceID)
				return false, err
			}
			s.logger.Infof("created btp-manager secret on cluster for instance %s successfully", instance.InstanceID)
		}
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("while getting secret from cluster for instance %s : %s", instance.InstanceID, err)
	}

	notMatchingKeys, err := s.compareSecrets(currentSecret, futureSecret)
	if err != nil {
		return false, fmt.Errorf("validation of secrets failed with unexpected reason for instance: %s : %s", instance.InstanceID, err)
	} else if len(notMatchingKeys) > 0 {
		s.logger.Infof("btp-manager secret on cluster does not match for instance credentials in db : %s, incorrect values for keys: %s ", instance.InstanceID, strings.Join(notMatchingKeys, ","))
		if s.dryRun {
			s.logger.Infof("[dry-run] secret for instance %s would be updated", instance.InstanceID)
		} else {
			if err := CreateOrUpdateSecret(k8sClient, futureSecret, s.logger); err != nil {
				s.logger.Errorf("while updating secret in cluster for %s %s", instance.InstanceID, err)
				return false, err
			}
			s.logger.Infof("btp-manager secret on cluster updated for %s to match state from instances db", instance.InstanceID)
		}
		return true, nil
	} else {
		s.logger.Infof("instance %s OK: btp-manager secret on cluster match within expected data", instance.InstanceID)
	}

	return false, nil
}

func (s *Manager) getSkrK8sClient(instance *internal.Instance) (client.Client, error) {
	secretName := getKubeConfigSecretName(instance.RuntimeID)
	kubeConfigSecret := &v1.Secret{}
	err := s.kcpK8sClient.Get(s.ctx, client.ObjectKey{Name: secretName, Namespace: kcpNamespace}, kubeConfigSecret)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("while getting secret from kcp for %s : %w", instance.InstanceID, err)
	}

	var kubeConfig []byte
	if errors.IsNotFound(err) {
		s.logger.Infof("not found secret for %s, now it will be executed try to get kubeConfig from provisioner.", instance.InstanceID)
		status, err := s.provisioner.RuntimeStatus(instance.Parameters.ErsContext.GlobalAccountID, instance.RuntimeID)
		if err != nil {
			return nil, fmt.Errorf("while getting runtime status from provisioner for %s : %s", instance.InstanceID, err)
		}

		if status.RuntimeConfiguration.Kubeconfig == nil {
			return nil, fmt.Errorf("kubeconfig empty in provisioner response for %s", instance.InstanceID)
		}

		s.logger.Infof("found kubeconfig in provisioner for %s", instance.InstanceID)
		kubeConfig = []byte(*status.RuntimeConfiguration.Kubeconfig)
	} else {
		s.logger.Infof("found secret %s on kcp cluster for %s", secretName, instance.InstanceID)

		config, ok := kubeConfigSecret.Data["config"]
		if !ok {
			return nil, fmt.Errorf("while getting 'config' from secret from %s for %s", secretName, instance.InstanceID)
		}

		s.logger.Infof("found kubeconfig in secret %s for %s", secretName, instance.InstanceID)
		kubeConfig = config
	}

	if kubeConfig == nil || len(kubeConfig) == 0 {
		return nil, fmt.Errorf("not found kubeConfig as secret nor in provisioner or is empty for %s", instance.InstanceID)
	}
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("while making REST cfg from kube config string for %s : %s", instance.InstanceID, err)
	}
	k8sClient, err := client.New(restCfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("while creating k8sClient from REST config for %s : %s", instance.InstanceID, err)
	}
	return k8sClient, nil
}

func (s *Manager) compareSecrets(s1, s2 *v1.Secret) ([]string, error) {
	areSecretEqualByKey := func(key string) (bool, error) {
		currentValue, ok := s1.Data[key]
		if !ok {
			return false, fmt.Errorf("while getting the value for the  key %s in the first secret", key)
		}
		expectedValue, ok := s2.Data[key]
		if !ok {
			return false, fmt.Errorf("while getting the value for the key %s in the second secret", key)
		}
		return reflect.DeepEqual(currentValue, expectedValue), nil
	}

	notEqual := make([]string, 0)
	for _, key := range []string{secretClientSecret, secretClientId, secretSmUrl, secretTokenUrl, secretClusterId} {
		equal, err := areSecretEqualByKey(key)
		if err != nil {
			s.logger.Errorf("getting value for key %s", key)
			return nil, err
		}
		if !equal {
			notEqual = append(notEqual, key)
		}
	}

	return notEqual, nil
}

func getKubeConfigSecretName(runtimeId string) string {
	return fmt.Sprintf("kubeconfig-%s", runtimeId)
}

func PrepareSecret(credentials *internal.ServiceManagerOperatorCredentials, clusterID string) (*apicorev1.Secret, error) {
	if credentials == nil || clusterID == "" {
		return nil, fmt.Errorf("empty params given")
	}
	if credentials.ClientID == "" {
		return nil, fmt.Errorf("client Id not set")
	}
	if credentials.ClientSecret == "" {
		return nil, fmt.Errorf("clients ecret not set")
	}
	if credentials.ServiceManagerURL == "" {
		return nil, fmt.Errorf("service manager url not set")
	}
	if credentials.URL == "" {
		return nil, fmt.Errorf("url not set")
	}

	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        BtpManagerSecretName,
			Namespace:   BtpManagerSecretNamespace,
			Labels:      BtpManagerLabels,
			Annotations: BtpManagerAnnotations,
		},
		Data: map[string][]byte{
			secretClientId:     []byte(credentials.ClientID),
			secretClientSecret: []byte(credentials.ClientSecret),
			secretSmUrl:        []byte(credentials.ServiceManagerURL),
			secretTokenUrl:     []byte(credentials.URL),
			secretClusterId:    []byte(clusterID),
		},
		Type: apicorev1.SecretTypeOpaque,
	}, nil
}

func CreateOrUpdateSecret(k8sClient client.Client, futureSecret *apicorev1.Secret, log logrus.FieldLogger) error {
	if futureSecret == nil {
		return fmt.Errorf("empty secret data given")
	}
	currentSecret := apicorev1.Secret{}
	getErr := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: BtpManagerSecretNamespace, Name: BtpManagerSecretName}, &currentSecret)
	switch {
	case getErr != nil && !apierrors.IsNotFound(getErr):
		return fmt.Errorf("failed to get the secret for BTP Manager: %s", getErr)
	case getErr != nil && apierrors.IsNotFound(getErr):
		namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: BtpManagerSecretNamespace}}
		createErr := k8sClient.Create(context.Background(), namespace)
		if createErr != nil && !apierrors.IsAlreadyExists(createErr) {
			return fmt.Errorf("could not create %s namespace: %s", BtpManagerSecretNamespace, createErr)
		}

		createErr = k8sClient.Create(context.Background(), futureSecret)
		if createErr != nil {
			return fmt.Errorf("failed to create the secret for BTP Manager: %s", createErr)
		}

		log.Info("the secret for BTP Manager created")
		return nil
	default:
		if !reflect.DeepEqual(currentSecret.Labels, BtpManagerLabels) {
			log.Warnf("the secret %s was not created by KEB and its data will be overwritten", BtpManagerSecretName)
		}

		currentSecret.Data = futureSecret.Data
		currentSecret.ObjectMeta.Labels = futureSecret.ObjectMeta.Labels
		currentSecret.ObjectMeta.Annotations = futureSecret.ObjectMeta.Annotations
		updateErr := k8sClient.Update(context.Background(), &currentSecret)
		if updateErr != nil {
			return fmt.Errorf("failed to update the secret for BTP Manager: %s", updateErr)
		}

		log.Info("the secret for BTP Manager updated")
		return nil
	}
}
