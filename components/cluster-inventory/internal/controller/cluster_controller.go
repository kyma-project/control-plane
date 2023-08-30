/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	clusterinventoryv1beta1 "github.com/kyma-project/control-plane/components/cluster-inventory/api/v1beta1"
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
	var secret corev1.Secret
	key := types.NamespacedName{Name: cluster.Name, Namespace: r.SecretNamespace}

	if err := r.Client.Get(context.Background(), key, &secret); err != nil {
		if k8serrors.IsNotFound(err) {
			secret, err := r.createSecret(cluster)
			if err != nil {
				return err
			}

			return r.Client.Create(context.Background(), &secret)
		}

		return err
	}

	return r.rotateSecret(&secret)
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

func (r *ClusterReconciler) createSecret(cluster clusterinventoryv1beta1.Cluster) (corev1.Secret, error) {
	clusterInventoryLabels := cluster.Labels

	labels := map[string]string{}

	labels[instanceIDLabel] = clusterInventoryLabels[instanceIDLabel]
	labels[runtimeIDLabel] = clusterInventoryLabels[runtimeIDLabel]
	labels[planIDLabel] = clusterInventoryLabels[planIDLabel]
	labels[planNameLabel] = clusterInventoryLabels[planNameLabel]
	labels[globalAccountIDLabel] = clusterInventoryLabels[globalAccountIDLabel]
	labels[subaccountIDLabel] = clusterInventoryLabels[subaccountIDLabel]
	labels[shootNameLabel] = clusterInventoryLabels[shootNameLabel]
	labels[regionLabel] = clusterInventoryLabels[regionLabel]
	labels[kymaNameLabel] = clusterInventoryLabels[kymaNameLabel]
	labels["operator.kyma-project.io/managed-by"] = "lifecycle-manager"

	kubeconfig, err := r.KubeconfigProvider.Fetch(labels[shootNameLabel])
	if err != nil {
		return corev1.Secret{}, err
	}

	return corev1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      cluster.Name,
			Namespace: r.SecretNamespace,
			Labels:    labels,
		},
		StringData: map[string]string{"config": kubeconfig},
	}, nil
}

func (r *ClusterReconciler) rotateSecret(secret *corev1.Secret) error {
	_, forceKubeconfigRotation := secret.Annotations["operator.kyma-project.io/force-kubeconfig-rotation"]

	if forceKubeconfigRotation {
		kubeconfig, err := r.KubeconfigProvider.Fetch(secret.Labels[shootNameLabel])
		if err != nil {
			return err
		}
		delete(secret.Annotations, "operator.kyma-project.io/force-kubeconfig-rotation")

		secret.StringData = map[string]string{"config": kubeconfig}

		return r.Client.Update(context.Background(), secret)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterinventoryv1beta1.Cluster{}).
		Complete(r)
}
