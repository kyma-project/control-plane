package controller

import (
	"context"
	"github.com/kyma-project/control-plane/components/cluster-inventory/api/v1beta1"
	"github.com/kyma-project/control-plane/components/cluster-inventory/internal/controller/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Cluster Inventory controller", func() {
	Context("Secret with kubeconfig doesn't exist", func() {
		kymaName := "kymaname1"
		secretName := "secret-name1"
		shootName := "shootName1"
		namespace := "default"

		It("Create, and remove secret", func() {
			By("Create Cluster CR")
			// TODO: Cluster Inventory CR should have Cluster scope
			clusterCR := fixClusterInventoryCR(kymaName, namespace, kymaName, shootName, secretName)
			Expect(k8sClient.Create(context.Background(), &clusterCR)).To(Succeed())

			By("Wait for secret creation")
			var kubeconfigSecret corev1.Secret
			key := types.NamespacedName{Name: secretName, Namespace: namespace}

			Eventually(func() bool {
				return k8sClient.Get(context.Background(), key, &kubeconfigSecret) == nil
			}, time.Second*30, time.Second*3).Should(BeTrue())

			err := k8sClient.Get(context.Background(), key, &kubeconfigSecret)
			Expect(err).To(BeNil())
			expectedSecret := fixNewSecret(secretName, namespace, kymaName, shootName, "kubeconfig1")
			Expect(kubeconfigSecret.Labels).To(Equal(expectedSecret.Labels))
			Expect(kubeconfigSecret.Data).To(Equal(expectedSecret.Data))
			Expect(kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]).To(Not(BeEmpty()))

			By("Delete Cluster CR")
			Expect(k8sClient.Delete(context.Background(), &clusterCR)).To(Succeed())

			By("Wait for secret deletion")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), key, &kubeconfigSecret)

				return err != nil && k8serrors.IsNotFound(err)

			}, time.Second*30, time.Second*3).Should(BeTrue())
		})
	})

	Context("Secret with kubeconfig exists", func() {
		namespace := "default"

		DescribeTable("Create secret when needed", func(clusterCR v1beta1.Cluster, secret corev1.Secret, previousTimestamp, expectedKubeconfig string) {
			By("Create kubeconfig secret")
			Expect(k8sClient.Create(context.Background(), &secret)).To(Succeed())

			By("Create Cluster CR")
			Expect(k8sClient.Create(context.Background(), &clusterCR)).To(Succeed())

			var kubeconfigSecret corev1.Secret
			key := types.NamespacedName{Name: secret.Name, Namespace: namespace}

			Eventually(func() bool {

				err := k8sClient.Get(context.Background(), key, &kubeconfigSecret)
				if err != nil {
					return false
				}

				_, forceAnnotationFound := kubeconfigSecret.Annotations[forceRotationAnnotation]
				timestampAnnotation := kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]

				return !forceAnnotationFound && timestampAnnotation != previousTimestamp
			}, time.Second*30, time.Second*3).Should(BeTrue())

			err := k8sClient.Get(context.Background(), key, &kubeconfigSecret)
			Expect(err).To(BeNil())
			Expect(string(kubeconfigSecret.Data["config"])).To(Equal(expectedKubeconfig))
		},
			Entry("Rotate static kubeconfig",
				fixClusterInventoryCR("cluster2", "default", "kymaName2", "shootName2", "static-kubeconfig-secret"),
				fixSecretWithForceRotation("static-kubeconfig-secret", namespace, "kymaName2", "shootName2", "kubeconfig2"),
				"",
				"kubeconfig2"),
			Entry("Rotate dynamic kubeconfig",
				fixClusterInventoryCR("cluster3", "default", "kymaName3", "shootName3", "dynamic-kubeconfig-secret"),
				fixSecretWithDynamicKubeconfig("dynamic-kubeconfig-secret", namespace, "kymaName3", "shootName3", "kubeconfig3", "2006-01-02T15:04:05Z07:00"),
				"2022-11-10 23:00:00 +0000",
				"kubeconfig3"),
		)

		Describe("Skip rotation", func() {

		})
	})
})

type TestSecret struct {
	secret corev1.Secret
}

func newTestSecret(name, namespace string) *TestSecret {
	return &TestSecret{
		secret: corev1.Secret{
			ObjectMeta: v12.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
}

func (sb *TestSecret) WithLabels(labels map[string]string) *TestSecret {
	sb.secret.Labels = labels

	return sb
}

func (sb *TestSecret) WithAnnotations(annotations map[string]string) *TestSecret {
	sb.secret.Annotations = annotations

	return sb
}

func (sb *TestSecret) WithData(data string) *TestSecret {
	sb.secret.Data = (map[string][]byte{"config": []byte(data)})

	return sb
}

func (sb *TestSecret) ToSecret() corev1.Secret {
	return sb.secret
}

type TestClusterInventoryCR struct {
	cluster v1beta1.Cluster
}

func newTestClusterInventoryCR(name, namespace, shootName, secretName string) *TestClusterInventoryCR {
	return &TestClusterInventoryCR{
		cluster: v1beta1.Cluster{
			ObjectMeta: v12.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1beta1.ClusterSpec{
				Core: v1beta1.Core{
					ShootName: shootName,
				},
				AdminKubeconfig: v1beta1.AdminKubeconfig{
					SecretName: secretName,
				},
			},
		},
	}
}

func (sb *TestClusterInventoryCR) WithLabels(labels map[string]string) *TestClusterInventoryCR {
	sb.cluster.Labels = labels

	return sb
}

func (sb *TestClusterInventoryCR) ToCluster() v1beta1.Cluster {
	return sb.cluster
}

func fixClusterInventoryCR(name, namespace, kymaName, shootName, secretName string) v1beta1.Cluster {
	return newTestClusterInventoryCR(name, namespace, shootName, secretName).
		WithLabels(fixClusterInventoryLabels(kymaName, shootName)).ToCluster()
}

func fixClusterInventoryLabels(kymaName, shootName string) map[string]string {

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

	return labels
}

func fixSecretLabels(kymaName, shootName string) map[string]string {
	labels := fixClusterInventoryLabels(kymaName, shootName)
	labels["operator.kyma-project.io/managed-by"] = "lifecycle-manager"
	labels["operator.kyma-project.io/cluster-name"] = kymaName
	return labels
}

func fixDynamicKubeconfigAnnotations(date string) map[string]string {
	return map[string]string{lastKubeconfigSyncAnnotation: date}
}

func fixForceKubeconfigAnnotations() map[string]string {
	return map[string]string{forceRotationAnnotation: "true"}
}

func fixNewSecret(name, namespace, kymaName, shootName, data string) corev1.Secret {
	labels := fixSecretLabels(kymaName, shootName)

	builder := newTestSecret(name, namespace)
	return builder.WithLabels(labels).WithData(data).ToSecret()
}

func fixSecretWithForceRotation(name, namespace, kymaName, shootName, data string) corev1.Secret {
	labels := fixSecretLabels(kymaName, shootName)
	annotations := fixForceKubeconfigAnnotations()

	return newTestSecret(name, namespace).WithAnnotations(annotations).WithLabels(labels).WithData(data).ToSecret()
}

func fixSecretWithDynamicKubeconfig(name, namespace, kymaName, shootName, data, timestamp string) corev1.Secret {
	labels := fixSecretLabels(kymaName, shootName)
	annotations := fixDynamicKubeconfigAnnotations(timestamp)

	builder := newTestSecret(name, namespace)
	return builder.WithAnnotations(annotations).WithLabels(labels).WithData(data).ToSecret()
}

func setupKubeconfigProviderMock(kpMock *mocks.KubeconfigProvider) {
	kpMock.On("Fetch", "shootName1").Return("kubeconfig1", nil)
	kpMock.On("Fetch", "shootName2").Return("kubeconfig2", nil)
	kpMock.On("Fetch", "shootName3").Return("kubeconfig3", nil)
}
