package controller

import (
	"context"
	"github.com/kyma-project/control-plane/components/cluster-inventory/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

const namespace = "test"

var _ = Describe("Cluster Inventory controller", func() {
	Context("Secret with kubeconfig doesn't exist", func() {
		kymaName := "kymaname"
		namespace := "default"

		It("Create secret", func() {
			By("Create Cluster CR")
			// TODO: Cluster Inventory CR should have Cluster scope
			clusterCR := fixClusterInventoryCR(kymaName, kymaName, namespace)

			Expect(k8sClient.Create(context.Background(), &clusterCR)).To(Succeed())

			By("Wait for secret creation")
			var obj corev1.Secret
			key := types.NamespacedName{Name: kymaName, Namespace: namespace}

			Eventually(func() bool {
				return k8sClient.Get(context.Background(), key, &obj) == nil
			}, time.Second*60, time.Second*3).Should(BeTrue())

			err := k8sClient.Get(context.Background(), key, &obj)
			Expect(err).To(BeNil())
			Expect(obj).To(BeIdenticalTo(fixSecret(kymaName, kymaName, namespace)))
		})
	})

	Context("Secret with kubeconfig exists", func() {
		Describe("Rotate static kubeconfig", func() {

		})

		Describe("Rotate dynamic kubeconfig", func() {

		})

		Describe("Skip rotation", func() {

		})

		Describe("Remove secret", func() {

		})
	})
})

func fixClusterInventoryCR(name, kymaName, namespace string) v1beta1.Cluster {

	labels := map[string]string{}

	labels["kyma-project.io/instance-id"] = "instanceID"
	labels["kyma-project.io/runtime-id"] = "runtimeID"
	labels["kyma-project.io/broker-plan-id"] = "planID"
	labels["kyma-project.io/broker-plan-name"] = "planName"
	labels["kyma-project.io/global-account-id"] = "globalAccountID"
	labels["kyma-project.io/subaccount-id"] = "subAccountID"
	labels["kyma-project.io/shoot-name"] = "shootName"
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

func fixSecret(name, kymaName, namespace string) corev1.Secret {
	labels := map[string]string{}

	labels["kyma-project.io/instance-id"] = "instanceID"
	labels["kyma-project.io/runtime-id"] = "runtimeID"
	labels["kyma-project.io/broker-plan-id"] = "planID"
	labels["kyma-project.io/broker-plan-name"] = "planName"
	labels["kyma-project.io/global-account-id"] = "globalAccountID"
	labels["kyma-project.io/subaccount-id"] = "subAccountID"
	labels["kyma-project.io/shoot-name"] = "shootName"
	labels["kyma-project.io/region"] = "region"
	labels["operator.kyma-project.io/kyma-name"] = kymaName
	labels["operator.kyma-project.io/managed-by"] = "lifecycle-manager"

	return corev1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
