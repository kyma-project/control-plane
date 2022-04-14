package installation

import (
	"context"
	"strings"
	"time"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	sc "github.com/kubernetes-sigs/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/service-catalog/pkg/util"
	utilErrors "github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apiServBeta "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

const SystemBrokerCRDName = "clusterservicebrokers.servicecatalog.k8s.io"
const SystemCatalogCRDName = "clusterserviceclasses.servicecatalog.k8s.io"
const SystemInstanceCRD = "serviceinstances.servicecatalog.k8s.io"

const (
	ClusterServiceBrokerNameLabel   = "servicecatalog.k8s.io/spec.clusterServiceBrokerName"
	ClusterServiceClassRefNameLabel = "servicecatalog.k8s.io/spec.clusterServiceClassRef.name"
)

type CleanupClient interface {
	PerformCleanup(resourceSelector string) error
}

type CrdsManager interface {
	List(ctx context.Context, opts metav1.ListOptions) (*apiServBeta.CustomResourceDefinitionList, error)
}

func NewServiceCatalogCleanupClient(kubeconfig *rest.Config) (CleanupClient, error) {
	scCli, err := sc.NewForConfig(kubeconfig)
	if err != nil {
		return &serviceCatalogClient{}, err
	}

	apiExtensionsClientSet, err := apiextensionsclient.NewForConfig(kubeconfig)

	if err != nil {
		return &serviceCatalogClient{}, err
	}

	crdsInterface := apiExtensionsClientSet.ApiextensionsV1beta1().CustomResourceDefinitions()

	return &serviceCatalogClient{
		client:      scCli,
		crdsManager: crdsInterface,
	}, nil
}

type serviceCatalogClient struct {
	client      sc.Interface
	crdsManager CrdsManager
}

func (s *serviceCatalogClient) PerformCleanup(resourceSelector string) error {
	exist, err := s.ensureCRDsExist()
	if err != nil {
		return errors.Wrapf(err, "while checking CustomResourceDefinitions")
	}

	if !exist {
		logrus.Info("Service Catalog not installed properly. Cleanup skipped")
		return nil
	}

	clusterServiceBrokers, err := s.listClusterServiceBroker(metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(utilErrors.K8SErrorToAppError(err), "while listing ClusterServiceBrokers")
	}

	logrus.Debugf("Filtering ClusterServiceBrokers with url prefix %s", resourceSelector)
	brokersWithUrlPrefix := s.filterCsbWithUrlPrefix(clusterServiceBrokers, resourceSelector)

	cscWithMatchingLabel, err := s.getClusterServiceClassesForBrokers(brokersWithUrlPrefix)
	if err != nil {
		return errors.Wrapf(err, "while getting ClusterServiceClasses")
	}

	serviceInstances, err := s.getServiceInstancesForClusterServiceClasses(cscWithMatchingLabel)
	if err != nil {
		return errors.Wrapf(err, "while getting ServiceInstances")
	}

	s.deleteServiceInstances(serviceInstances)
	s.removeClusterBrokersFinalizers(clusterServiceBrokers.Items)

	return nil
}

func (s *serviceCatalogClient) ensureCRDsExist() (bool, error) {
	list, err := s.crdsManager.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, utilErrors.K8SErrorToAppError(err)
	}
	for _, crd := range []string{SystemBrokerCRDName, SystemCatalogCRDName, SystemInstanceCRD} {
		exists := s.ensureCRDExists(crd, list)
		if !exists {
			logrus.Debugf("Custom Resource Definition: %s not found", crd)
			return false, nil
		}
	}
	return true, nil
}

func (s *serviceCatalogClient) ensureCRDExists(crdName string, list *apiServBeta.CustomResourceDefinitionList) bool {
	for _, item := range list.Items {
		if item.Name == crdName {
			return true
		}
	}
	return false
}

func (s *serviceCatalogClient) listClusterServiceBroker(options metav1.ListOptions) (*v1beta1.ClusterServiceBrokerList, error) {
	result := &v1beta1.ClusterServiceBrokerList{}
	err := wait.PollImmediate(10*time.Second, 2*time.Minute, func() (done bool, err error) {
		csbList, err := s.client.ServicecatalogV1beta1().ClusterServiceBrokers().List(context.Background(), options)
		if err != nil {
			logrus.Errorf("while listing ClusterServiceBrokers: %s", err.Error())
			return false, nil
		}
		result = csbList
		return true, nil
	})
	return result, err
}

func (s *serviceCatalogClient) listClusterServiceClass(options metav1.ListOptions) (*v1beta1.ClusterServiceClassList, error) {
	result := &v1beta1.ClusterServiceClassList{}
	err := wait.PollImmediate(10*time.Second, 2*time.Minute, func() (done bool, err error) {
		cscList, err := s.client.ServicecatalogV1beta1().ClusterServiceClasses().List(context.Background(), options)
		if err != nil {
			logrus.Errorf("while listing ClusterServiceClasses: %s", err.Error())
			return false, nil
		}
		result = cscList
		return true, nil
	})
	return result, err
}

func (s *serviceCatalogClient) listServiceInstance(options metav1.ListOptions) (*v1beta1.ServiceInstanceList, error) {
	result := &v1beta1.ServiceInstanceList{}
	err := wait.PollImmediate(10*time.Second, 2*time.Minute, func() (done bool, err error) {
		siList, err := s.client.ServicecatalogV1beta1().ServiceInstances(metav1.NamespaceAll).List(context.Background(), options)
		if err != nil {
			logrus.Errorf("while listing ServiceInstances: %s", err.Error())
			return false, nil
		}
		result = siList
		return true, nil
	})
	return result, err
}

func (s *serviceCatalogClient) filterCsbWithUrlPrefix(csbList *v1beta1.ClusterServiceBrokerList, urlPrefix string) []v1beta1.ClusterServiceBroker {
	var csbWithBrokerUrlPrefix []v1beta1.ClusterServiceBroker
	for _, clusterServiceBroker := range csbList.Items {
		if strings.HasPrefix(clusterServiceBroker.Spec.URL, urlPrefix) {
			csbWithBrokerUrlPrefix = append(csbWithBrokerUrlPrefix, clusterServiceBroker)
		}
	}

	return csbWithBrokerUrlPrefix
}

func (s *serviceCatalogClient) getClusterServiceClassesForBrokers(brokers []v1beta1.ClusterServiceBroker) ([]v1beta1.ClusterServiceClass, error) {
	var cscWithMatchingLabel []v1beta1.ClusterServiceClass

	for _, csb := range brokers {
		labelValue := util.GenerateSHA(csb.Name)
		csbListOptions := fixListOptionsWithLabelSelector(ClusterServiceBrokerNameLabel, labelValue)

		clusterServiceClasses, err := s.listClusterServiceClass(csbListOptions)
		if err != nil {
			return []v1beta1.ClusterServiceClass{}, errors.Wrapf(utilErrors.K8SErrorToAppError(err), "while listing ClusterServiceClasses for ClusterServiceBroker %q", csb.Name)
		}

		for _, serviceClass := range clusterServiceClasses.Items {
			logrus.Debugf("found ClusterServiceClass with label %q: %s", labelValue, serviceClass.Name)
			cscWithMatchingLabel = append(cscWithMatchingLabel, serviceClass)
		}
	}

	return cscWithMatchingLabel, nil
}

func (s *serviceCatalogClient) getServiceInstancesForClusterServiceClasses(serviceClasses []v1beta1.ClusterServiceClass) ([]v1beta1.ServiceInstance, error) {
	var serviceInstances []v1beta1.ServiceInstance

	for _, clusterServiceClass := range serviceClasses {
		labelValue := util.GenerateSHA(clusterServiceClass.Name)

		options := fixListOptionsWithLabelSelector(ClusterServiceClassRefNameLabel, labelValue)

		serviceInstancesList, err := s.listServiceInstance(options)
		if err != nil {
			return []v1beta1.ServiceInstance{}, errors.Wrapf(utilErrors.K8SErrorToAppError(err), "while listing ServiceInstances")
		}

		for _, serviceInstance := range serviceInstancesList.Items {
			logrus.Debugf("found ServiceInstance with label %q: %s", labelValue, serviceInstance.Name)
			serviceInstances = append(serviceInstances, serviceInstance)
		}
	}
	return serviceInstances, nil
}

func (s *serviceCatalogClient) deleteServiceInstances(serviceInstances []v1beta1.ServiceInstance) {
	for _, serviceInstance := range serviceInstances {
		logrus.Debugf("trying to delete ServiceInstance %q", serviceInstance.Name)

		_ = wait.PollImmediate(10*time.Second, 2*time.Minute, func() (done bool, err error) {
			if err := s.client.ServicecatalogV1beta1().ServiceInstances(serviceInstance.Namespace).Delete(context.Background(), serviceInstance.Name, metav1.DeleteOptions{}); err != nil {
				logrus.Errorf("while removing ServiceInstance %s: %s", serviceInstance.Name, err.Error())
				if apiErrors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			}
			return true, nil
		})
	}
}

func (s *serviceCatalogClient) removeClusterBrokersFinalizers(brokers []v1beta1.ClusterServiceBroker) {
	for _, broker := range brokers {
		logrus.Debugf("trying to remove finalizers from ClusterServiceBroker %q", broker.Name)

		_ = wait.PollImmediate(10*time.Second, 2*time.Minute, func() (bool, error) {
			broker.Finalizers = []string{}
			if _, err := s.client.ServicecatalogV1beta1().ClusterServiceBrokers().Update(context.Background(), &broker, metav1.UpdateOptions{}); err != nil {
				if apiErrors.IsNotFound(err) {
					return true, nil
				}
				logrus.Errorf("while updating ClusterServiceBroker %s: %s", broker.Name, err.Error())
				return false, nil
			}
			return true, nil
		})
	}
}

func fixListOptionsWithLabelSelector(labelName, labelValue string) metav1.ListOptions {
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{labelName: labelValue},
	}

	return metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
}
