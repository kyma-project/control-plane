package controller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	clusterinventoryv1beta1 "github.com/kyma-project/control-plane/components/cluster-inventory/api/v1beta1"
)

const (
	forceRotationAnnotation      = "operator.kyma-project.io/force-kubeconfig-rotation"
	lastKubeconfigSyncAnnotation = "operator.kyma-project.io/last-sync"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	KubeconfigProvider KubeconfigProvider
	SecretNamespace    string
	log                logr.Logger
}

type Client interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error
}

//go:generate mockery --name=KubeconfigProvider
type KubeconfigProvider interface {
	Fetch(shootName string) (string, error)
}

func NewClusterInventoryController(mgr ctrl.Manager, kubeconfigProvider KubeconfigProvider, secretNamespace string, log logr.Logger) *ClusterReconciler {
	return &ClusterReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		KubeconfigProvider: kubeconfigProvider,
		SecretNamespace:    secretNamespace,
		log:                log,
	}
}

//+kubebuilder:rbac:groups=clusterinventory.kyma-project.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=clusterinventory.kyma-project.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=clusterinventory.kyma-project.io,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	r.log.Info("Starting reconciliation")
	var cluster clusterinventoryv1beta1.Cluster

	err := r.Client.Get(ctx, req.NamespacedName, &cluster)
	if err != nil {
		r.log.Error(err, "failed to get Cluster CR")
		return ctrl.Result{
			Requeue: true,
		}, nil
	}

	err = r.rotateOrCreateSecret(cluster)
	if err != nil {
		r.log.Error(err, "failed to rotate or create secret")
		return ctrl.Result{
			Requeue: true,
		}, nil
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) rotateOrCreateSecret(cluster clusterinventoryv1beta1.Cluster) error {
	r.log.Info("Looking for secret")
	secret, err := r.getSecret(cluster.Labels["kyma-project.io/shoot-name"])
	if err != nil {
		return err
	}

	if secret == nil {
		r.log.Info("Secret not found, and will be created")
		return r.createSecret(cluster)
	}

	r.log.Info("Secret found, and will be rotated if needed")
	return r.rotateSecret(secret.Name, secret.Annotations)
}

func (r *ClusterReconciler) getSecret(shootName string) (*corev1.Secret, error) {
	var secretList corev1.SecretList

	selector := client.MatchingLabels(map[string]string{
		"kyma-project.io/shoot-name": shootName,
	})

	err := r.Client.List(context.Background(), &secretList, selector)
	if err != nil {
		return nil, err
	}

	size := len(secretList.Items)

	if size == 0 {
		return nil, nil
	}

	if size > 1 {
		return nil, errors.New(fmt.Sprintf("unexpected numer of secrets found for shoot `%s`", shootName))
	}

	return &secretList.Items[0], nil
}

const (
	instanceIDLabel      = "kyma-project.io/instance-id"
	runtimeIDLabel       = "kyma-project.io/runtime-id"
	planIDLabel          = "kyma-project.io/broker-plan-id"
	planNameLabel        = "kyma-project.io/broker-plan-name"
	globalAccountIDLabel = "kyma-project.io/global-account-id"
	subaccountIDLabel    = "kyma-project.io/subaccount-id"
	shootNameLabel       = "kyma-project.io/shoot-name"
	regionLabel          = "kyma-project.io/region"
	kymaNameLabel        = "operator.kyma-project.io/kyma-name"
)

func (r *ClusterReconciler) createSecret(cluster clusterinventoryv1beta1.Cluster) error {
	secret, err := r.newSecret(cluster)
	if err != nil {
		return err
	}

	return r.Client.Create(context.Background(), &secret)
}

func (r *ClusterReconciler) newSecret(cluster clusterinventoryv1beta1.Cluster) (corev1.Secret, error) {
	labels := map[string]string{}

	for key, val := range cluster.Labels {
		labels[key] = val
	}
	labels["operator.kyma-project.io/managed-by"] = "lifecycle-manager"

	kubeconfig, err := r.KubeconfigProvider.Fetch(labels[shootNameLabel])
	if err != nil {
		return corev1.Secret{}, err
	}

	return corev1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:        cluster.Name,
			Namespace:   r.SecretNamespace,
			Labels:      labels,
			Annotations: map[string]string{lastKubeconfigSyncAnnotation: time.Now().UTC().String()},
		},
		StringData: map[string]string{"config": kubeconfig},
	}, nil
}

func (r *ClusterReconciler) rotateSecret(secretName string, annotations map[string]string) error {
	_, forceKubeconfigRotation := annotations[forceRotationAnnotation]

	if forceKubeconfigRotation {
		r.log.Info("Secret has operator.kyma-project.io/force-kubeconfig-rotation annotation and will be rotated")
		var secret corev1.Secret
		key := types.NamespacedName{Name: secretName, Namespace: r.SecretNamespace}

		err := r.Client.Get(context.Background(), key, &secret)
		if err != nil {
			r.log.Error(err, "failed to get secret")
			return err
		}

		r.log.Info("Fetching dynamic kubeconfig")
		kubeconfig, err := r.KubeconfigProvider.Fetch(secret.Labels[shootNameLabel])
		if err != nil {
			r.log.Error(err, "failed to fetch dynamic kubeconfig")
			return err
		}

		r.log.Info("Updating secret with new data")
		delete(secret.Annotations, forceRotationAnnotation)
		secret.Annotations[lastKubeconfigSyncAnnotation] = time.Now().UTC().String()

		secret.StringData = map[string]string{"config": kubeconfig}

		r.log.Info(fmt.Sprintf("%v", secret.Name))
		err = r.Client.Update(context.Background(), &secret)
		if err != nil {
			r.log.Error(err, "failed to update secret")
		}

		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterinventoryv1beta1.Cluster{}).
		Complete(r)
}
