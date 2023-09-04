package controller

import (
	"context"
	"github.com/kyma-project/control-plane/components/cluster-inventory/api/v1beta1"
	"github.com/kyma-project/control-plane/components/cluster-inventory/internal/controller/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

const (
	namespace  = "test"
	kubeconfig = "kubeconfig"
)

var _ = Describe("Cluster Inventory controller", func() {
	Context("Secret with kubeconfig doesn't exist", func() {
		kymaName := "kymaname1"
		namespace := "default"

		It("Create secret", func() {
			By("Create Cluster CR")
			// TODO: Cluster Inventory CR should have Cluster scope
			clusterCR := fixClusterInventoryCR(kymaName, kymaName, "shootName1", namespace)

			Expect(k8sClient.Create(context.Background(), &clusterCR)).To(Succeed())

			By("Wait for secret creation")
			var kubeconfigSecret corev1.Secret
			key := types.NamespacedName{Name: kymaName, Namespace: namespace}

			Eventually(func() bool {
				return k8sClient.Get(context.Background(), key, &kubeconfigSecret) == nil
			}, time.Second*30, time.Second*3).Should(BeTrue())

			err := k8sClient.Get(context.Background(), key, &kubeconfigSecret)
			Expect(err).To(BeNil())
			expectedSecret := fixSecret(kymaName, kymaName, "shootName1", namespace)
			Expect(kubeconfigSecret.Labels).To(Equal(expectedSecret.Labels))
			Expect(kubeconfigSecret.Data).To(Equal(expectedSecret.Data))
			Expect(kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]).To(Not(BeEmpty()))
		})
	})

	type AnnotationsPredicate func(secret corev1.Secret) bool

	Context("Secret with kubeconfig exists", func() {
		kymaName := "kymaname2"
		namespace := "default"

		DescribeTable("Rotate kubeconfig", func(shootName string, secret corev1.Secret, predicate AnnotationsPredicate) {
			By("Create kubeconfig secret")
			Expect(k8sClient.Create(context.Background(), &secret)).To(Succeed())

			By("Create Cluster CR")
			clusterCR := fixClusterInventoryCR(kymaName, kymaName, "shootName2", namespace)

			Expect(k8sClient.Create(context.Background(), &clusterCR)).To(Succeed())

			var kubeconfigSecret corev1.Secret
			key := types.NamespacedName{Name: kymaName, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), key, &kubeconfigSecret)
				if err != nil {
					return false
				}

				return predicate(kubeconfigSecret)
			}, time.Second*30, time.Second*3).Should(BeTrue())

			err := k8sClient.Get(context.Background(), key, &kubeconfigSecret)
			Expect(err).To(BeNil())
			expectedSecret := fixSecret(kymaName, kymaName, "shootName2", namespace)
			Expect(kubeconfigSecret.Labels).To(Equal(expectedSecret.Labels))
			Expect(kubeconfigSecret.Data).To(Equal(expectedSecret.Data))
			Expect(kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]).To(Not(BeEmpty()))
		},
			Entry("Rotate static kubeconfig", "shootName2", forceRotationAnnotation),
			Entry("Rotate dynamic kubeconfig", "shootName3", forceRotationAnnotation),
		)

		Describe("Skip rotation", func() {

		})

		Describe("Remove secret", func() {

		})
	})
})

type SecretBuilder struct {
	name        string
	namespace   string
	labels      map[string]string
	annotations map[string]string
	data        string
}

func NewSecretBuilder(name, namespace, data string) *SecretBuilder {
	return &SecretBuilder{
		name:      name,
		namespace: namespace,
		data:      data,
	}
}

func (sb *SecretBuilder) WithLabels(labels map[string]string) *SecretBuilder {
	sb.labels = labels

	return sb
}

func (sb *SecretBuilder) WithAnnotations(annotations map[string]string) *SecretBuilder {
	sb.annotations = annotations

	return sb
}

func (sb SecretBuilder) Build() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:        sb.name,
			Namespace:   sb.namespace,
			Labels:      sb.labels,
			Annotations: sb.annotations,
		},
	}

}

func fixClusterInventoryCR(name, kymaName, shootName, namespace string) v1beta1.Cluster {

	labels := map[string]string{}

	labels["kyma-project.io/instance-id"] = "instanceID"
	labels["kyma-project.io/runtime-id"] = "runtimeID"
	labels["kyma-project.io/broker-plan-id"] = "planID"
	labels["kyma-project.io/broker-plan-name"] = "planName"
	labels["kyma-project.io/global-account-id"] = "globalAccountID"
	labels["kyma-project.io/subaccount-id"] = "subAccountID"
	labels["kyma-project.io/shoot-name"] = shootName
	labels["kyma-project.io/region"] = "region"
	labels["operator.kyma-project.io/kyma-name"] = kymaName

	return v1beta1.Cluster{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func fixSecretLabels(name, kymaName, shootName, namespace string) map[string]string {
	labels := map[string]string{}

	labels["kyma-project.io/instance-id"] = "instanceID"
	labels["kyma-project.io/runtime-id"] = "runtimeID"
	labels["kyma-project.io/broker-plan-id"] = "planID"
	labels["kyma-project.io/broker-plan-name"] = "planName"
	labels["kyma-project.io/global-account-id"] = "globalAccountID"
	labels["kyma-project.io/subaccount-id"] = "subAccountID"
	labels["kyma-project.io/shoot-name"] = shootName
	labels["kyma-project.io/region"] = "region"
	labels["operator.kyma-project.io/kyma-name"] = kymaName
	labels["operator.kyma-project.io/managed-by"] = "lifecycle-manager"

	return labels
}

func fixSecret(name, kymaName, shootName, namespace string) corev1.Secret {
	labels := map[string]string{}

	labels["kyma-project.io/instance-id"] = "instanceID"
	labels["kyma-project.io/runtime-id"] = "runtimeID"
	labels["kyma-project.io/broker-plan-id"] = "planID"
	labels["kyma-project.io/broker-plan-name"] = "planName"
	labels["kyma-project.io/global-account-id"] = "globalAccountID"
	labels["kyma-project.io/subaccount-id"] = "subAccountID"
	labels["kyma-project.io/shoot-name"] = shootName
	labels["kyma-project.io/region"] = "region"
	labels["operator.kyma-project.io/kyma-name"] = kymaName
	labels["operator.kyma-project.io/managed-by"] = "lifecycle-manager"

	return corev1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{"config": []byte(kubeconfig)},
	}
}

func fixSecretWithForceRotation(name, kymaName, shootName, namespace string, dynamic bool) corev1.Secret {
	labels := map[string]string{}

	labels["kyma-project.io/instance-id"] = "instanceID"
	labels["kyma-project.io/runtime-id"] = "runtimeID"
	labels["kyma-project.io/broker-plan-id"] = "planID"
	labels["kyma-project.io/broker-plan-name"] = "planName"
	labels["kyma-project.io/global-account-id"] = "globalAccountID"
	labels["kyma-project.io/subaccount-id"] = "subAccountID"
	labels["kyma-project.io/shoot-name"] = shootName
	labels["kyma-project.io/region"] = "region"
	labels["operator.kyma-project.io/kyma-name"] = kymaName
	labels["operator.kyma-project.io/managed-by"] = "lifecycle-manager"

	annotations := map[string]string{}

	if dynamic {
		annotations[lastKubeconfigSyncAnnotation] = "2013-05-01 23:00:00 +0000 UTC"
	} else {
		annotations[forceRotationAnnotation] = "true"
	}

	return corev1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		StringData: map[string]string{"config": "static kubeconfig"},
	}
}

func setupKubeconfigProviderMock(kpMock *mocks.KubeconfigProvider) {
	kpMock.On("Fetch", "shootName1").Return(kubeconfig, nil)
	kpMock.On("Fetch", "shootName2").Return(kubeconfig, nil)
}
