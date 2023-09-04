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

var _ = Describe("Cluster Inventory controller", func() {
	Context("Secret with kubeconfig doesn't exist", func() {
		kymaName := "kymaname1"
		namespace := "default"

		It("Create secret", func() {
			By("Create Cluster CR")
			// TODO: Cluster Inventory CR should have Cluster scope
			clusterCR := fixClusterInventoryCR(kymaName, namespace, kymaName, "shootName1")
			Expect(k8sClient.Create(context.Background(), &clusterCR)).To(Succeed())

			By("Wait for secret creation")
			var kubeconfigSecret corev1.Secret
			key := types.NamespacedName{Name: kymaName, Namespace: namespace}

			Eventually(func() bool {
				return k8sClient.Get(context.Background(), key, &kubeconfigSecret) == nil
			}, time.Second*30, time.Second*3).Should(BeTrue())

			err := k8sClient.Get(context.Background(), key, &kubeconfigSecret)
			Expect(err).To(BeNil())
			expectedSecret := fixNewSecret(kymaName, namespace, kymaName, "shootName1", "kubeconfig1")
			Expect(kubeconfigSecret.Labels).To(Equal(expectedSecret.Labels))
			Expect(kubeconfigSecret.Data).To(Equal(expectedSecret.Data))
			Expect(kubeconfigSecret.Annotations[lastKubeconfigSyncAnnotation]).To(Not(BeEmpty()))
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
				fixClusterInventoryCR("cluster2", "default", "kymaName2", "shootName2"),
				fixSecretWithForceRotation("static-kubeconfig-secret", namespace, "kymaName2", "shootName2", "kubeconfig2"),
				"",
				"kubeconfig2"),
			//Entry("Rotate dynamic kubeconfig",
			//	fixClusterInventoryCR("cluster3", "default", "kymaName3", "shootName3"),
			//	fixNewSecret("dynamic-kubeconfig-secret", namespace, "kymaName3", "shootName3", "kubeconfig3"),
			//	"oooo",
			//	"dynamic kubeconfig 2"),
		)

		Describe("Skip rotation", func() {

		})

		Describe("Remove secret", func() {

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

func newTestClusterInventoryCR(name, namespace string) *TestClusterInventoryCR {
	return &TestClusterInventoryCR{
		cluster: v1beta1.Cluster{
			ObjectMeta: v12.ObjectMeta{
				Name:      name,
				Namespace: namespace,
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

func fixClusterInventoryCR(name, namespace, kymaName, shootName string) v1beta1.Cluster {
	return newTestClusterInventoryCR(name, namespace).
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

func fixSecretWithDynamicKubeconfig(name, namespace, kymaName, shootName string) corev1.Secret {
	labels := fixSecretLabels(kymaName, shootName)
	annotations := fixDynamicKubeconfigAnnotations("2022-11-10 23:00:00 +0000")

	builder := newTestSecret(name, namespace)
	return builder.WithAnnotations(annotations).WithLabels(labels).WithData("dynamic kubeconfig").ToSecret()
}

func setupKubeconfigProviderMock(kpMock *mocks.KubeconfigProvider) {
	kpMock.On("Fetch", "shootName1").Return("kubeconfig1", nil)
	kpMock.On("Fetch", "shootName2").Return("kubeconfig2", nil)
	kpMock.On("Fetch", "shootName3").Return("kubeconfig3", nil)
}
