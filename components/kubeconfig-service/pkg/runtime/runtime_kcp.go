package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/kubeconfig-service/pkg/caller"
	"github.com/kyma-project/control-plane/components/kubeconfig-service/pkg/env"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

/*
ConfigMap example:
Name:        "userID",
Namespace:   "kcp-system",
Label:       "service=kubeconfig",
Annotation:map[string]string
    "role=L2L3ROLE"
     "tenant=tenantID"
Data:map[string]string
    "runtimeid-a" : "starttime"
    "runtimeid-b" : "starttime"
*/

const KcpNamespace string = "kcp-system"

const ExpireTime time.Duration = 7 * 24 * time.Hour

type JsonPatchType struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func SetupConfigMap() error {
	configMapList, err := getConfigMapList()
	if configMapList == nil && err == nil {
		log.Info("No Timer will be setup.")
		return nil
	} else if err != nil {
		log.Errorf("Failed to setup timer.")
		return err
	}

	for _, configMap := range (*configMapList).Items {
		userID := configMap.ObjectMeta.Name
		role := configMap.ObjectMeta.Annotations["role"]
		tenantID := configMap.ObjectMeta.Annotations["tenant"]
		for runtimeID, startTimeString := range configMap.Data {
			log.Infof("Found ConfigMap for runtime %s user %s.", runtimeID, userID)
			c := caller.NewCaller(env.Config.GraphqlURL, tenantID)
			status, err := c.RuntimeStatus(runtimeID)
			fmt.Println("after call RuntimeStatus() ", err)
			if strings.Contains(fmt.Sprint(err), "not found") && strings.Contains(fmt.Sprint(err), "error getting Shoot") {
				//delete ConfigMap if shoot no longer exists
				coreClientset, err := GetK8sClient()
				if err != nil {
					log.Errorf("Failed to create core client set.")
					return err
				}
				err = cleanConfigMap(coreClientset, userID, runtimeID)
				if err != nil {
					log.Errorf("Failed to clean ConfigMap for user %s runtime %s, %s", userID, runtimeID, err.Error())
					return err
				}
				continue
			} else if err != nil {
				log.Errorf("Failed to fetch runtime status.")
				return err
			}
			rawConfig := *status.RuntimeConfiguration.Kubeconfig
			rtc, err := NewRuntimeClient([]byte(rawConfig), userID, role, tenantID)
			if err != nil {
				log.Errorf("Failed to create runtime client.")
				return err
			}
			startTime, err := time.Parse("2006-01-02 15:04:05 +0000 UTC", startTimeString)
			if err != nil {
				log.Errorf("Failed to convert start time.")
				return err
			}
			go rtc.SetupTimer(startTime, runtimeID)
		}
	}

	return nil
}

func GetK8sConfig() (*restclient.Config, error) {
	k8sConfig, err := restclient.InClusterConfig()
	if err != nil {
		log.Warnf("Failed to read in cluster config: %s", err.Error())
		log.Info("Trying to initialize with local config")
		home := homedir.HomeDir()
		k8sConfPath := filepath.Join(home, ".kube", "config")
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", k8sConfPath)
		if err != nil {
			log.Errorf("Failed to read k8s in-cluster configuration, %s", err.Error())
			return nil, err
		}
	}
	return k8sConfig, nil
}

func GetK8sClient() (kubernetes.Interface, error) {
	k8sconfig, err := GetK8sConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		log.Errorf("Failed to create k8s core client, %s", err.Error())
		return nil, err
	}
	return clientset, err
}

func (rtc *RuntimeClient) SetupTimer(startTime time.Time, runtimeID string) {
	userID := rtc.User.ServiceAccountName
	endTime := startTime.Add(ExpireTime)
	duration := time.Until(endTime)
	if duration >= 0 {
		timer := time.NewTimer(duration)
		<-timer.C
		defer timer.Stop()
	}

	//After timer, check start time, if changed, clean up SA and ConfigMap
	timeBefore := strings.Split(startTime.String(), " m=")[0]
	cm, err := rtc.KcpK8s.CoreV1().ConfigMaps(KcpNamespace).Get(context.Background(), userID, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Failed to get ConfigMap for user %s runtime %s, %s", userID, runtimeID, err.Error())
		return
	}
	timeAfter := cm.Data[runtimeID]
	if timeBefore != timeAfter {
		log.Infof("StartTime changed for runtime %s for user %s, skip clean up.", runtimeID, userID)
		return
	}

	log.Infof("Start to clean everything for runtime %s for user %s.", runtimeID, userID)
	rtc.RollbackE.Data = append(rtc.RollbackE.Data, SA)
	rtc.RollbackE.Data = append(rtc.RollbackE.Data, ClusterRole)
	rtc.RollbackE.Data = append(rtc.RollbackE.Data, ClusterRoleBinding)
	err = rtc.Cleaner()
	if err != nil {
		log.Errorf("Failed to clean runtime %s for user %s.", runtimeID, userID)
		return
	}

	err = rtc.UpdateConfigMap(runtimeID)
	if err != nil {
		log.Errorf("Failed to clean ConfigMap for runtime %s user %s", runtimeID, userID)
		return
	}
}

func (rtc *RuntimeClient) UpdateConfigMap(runtimeID string) error {
	log.Infof("Trying to remove expired information for runtime %s.", runtimeID)
	userID := rtc.User.ServiceAccountName

	//checking configmap existance
	cm, err := rtc.KcpK8s.CoreV1().ConfigMaps(KcpNamespace).Get(context.Background(), userID, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Failed to get ConfigMap for user %s runtime %s, %s", userID, runtimeID, err.Error())
		return err
	}
	if len(cm.Data[runtimeID]) == 0 {
		log.Infof("Configmap of runtime %s already deleted.", runtimeID)
		return nil
	}

	err = cleanConfigMap(rtc.KcpK8s, userID, runtimeID)
	if err != nil {
		log.Errorf("Failed to clean ConfigMap for user %s runtime %s, %s", userID, runtimeID, err.Error())
		return err
	}

	return nil
}

func (rtc *RuntimeClient) DeployConfigMap(runtimeID string, L2L3OperatorRole string, startTime time.Time) error {
	userID := rtc.User.ServiceAccountName
	tenantID := rtc.User.TenantID
	startTimeString := strings.Split(startTime.String(), " m=")[0]

	log.Info("Checking if the user exists")
	cm, err := rtc.KcpK8s.CoreV1().ConfigMaps(KcpNamespace).Get(context.Background(), userID, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		log.Info("User doens't exist. Trying to create configmap.")
		configmap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        userID,
				Namespace:   KcpNamespace,
				Labels:      map[string]string{"service": "kubeconfig"},
				Annotations: map[string]string{"role": L2L3OperatorRole, "tenant": tenantID},
			},
			Data: map[string]string{runtimeID: startTimeString},
		}
		_, err = rtc.KcpK8s.CoreV1().ConfigMaps(KcpNamespace).Create(context.Background(), configmap, metav1.CreateOptions{})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Errorf("Failed to create ConfigMap for user %s runtime %s, %s", userID, runtimeID, err.Error())
			return err
		}
		log.Infof("Configmap created for runtime %s user %s.", runtimeID, userID)
	} else if err != nil {
		log.Errorf("Failed to get ConfigMap for user %s runtime %s, %s", userID, runtimeID, err.Error())
		return err
	} else {
		log.Info("User already exist. Trying to update ConfigMap.")
		var patches []*JsonPatchType
		var patch *JsonPatchType
		path := "/data/" + runtimeID
		if len(cm.Data[runtimeID]) != 0 {
			log.Info("Runtime already exist. Trying to update expire time.")
			patch = &JsonPatchType{
				Op:    "replace",
				Path:  path,
				Value: startTimeString,
			}
		} else {
			log.Info("Runtime not exist. Trying to create new entry.")
			patch = &JsonPatchType{
				Op:    "add",
				Path:  path,
				Value: startTimeString,
			}
		}
		patches = append(patches, patch)
		payload, err := json.Marshal(patches)
		if err != nil {
			log.Errorf("Failed to marshal patch for user %s runtime %s, %s", userID, runtimeID, err.Error())
			return err
		}
		_, err = rtc.KcpK8s.CoreV1().ConfigMaps(KcpNamespace).Patch(context.Background(), userID, types.JSONPatchType, payload, metav1.PatchOptions{})
		if err != nil {
			log.Errorf("Failed to update ConfigMap for user %s runtime %s, %s", userID, runtimeID, err.Error())
			return err
		}
		log.Infof("Configmap updated for runtime %s user %s.", runtimeID, userID)
	}

	return nil
}

func getConfigMapList() (*v1.ConfigMapList, error) {
	coreClientset, err := GetK8sClient()
	if err != nil {
		log.Errorf("Failed to get kcp k8s client.")
		return nil, err
	}
	cmlist, err := coreClientset.CoreV1().ConfigMaps(KcpNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "service=kubeconfig"})
	if k8serrors.IsNotFound(err) {
		log.Info("All ConfigMap cleaned up.")
		return nil, nil
	} else if err != nil {
		log.Errorf("Failed to get ConfigMap list: %s", err.Error())
		return nil, err
	}
	return cmlist, nil
}

func cleanConfigMap(coreClientset kubernetes.Interface, userID string, runtimeID string) error {
	var patches []*JsonPatchType
	path := "/data/" + runtimeID
	patch := &JsonPatchType{
		Op:   "remove",
		Path: path,
	}
	patches = append(patches, patch)
	payload, err := json.Marshal(patches)
	if err != nil {
		log.Errorf("Failed to marshal patch, %s", err.Error())
		return err
	}
	_, err = coreClientset.CoreV1().ConfigMaps(KcpNamespace).Patch(context.Background(), userID, types.JSONPatchType, payload, metav1.PatchOptions{})
	if err != nil {
		log.Errorf("Failed to update ConfigMap, %s", err.Error())
		return err
	}
	log.Infof("Succeeded in cleaning up everything for runtime %s user %s", runtimeID, userID)

	//remove user ConfigMap if no runtime left
	cm, err := coreClientset.CoreV1().ConfigMaps(KcpNamespace).Get(context.Background(), userID, metav1.GetOptions{})
	if err != nil {
		log.Errorf("Failed to get ConfigMap for user %s runtime %s, %s", userID, runtimeID, err.Error())
		return err
	}
	if len(cm.Data) == 0 {
		log.Infof("No runtime left for user %s, start to remove ConfigMap.", userID)
		err = coreClientset.CoreV1().ConfigMaps(KcpNamespace).Delete(context.Background(), userID, metav1.DeleteOptions{})
		if err != nil {
			log.Errorf("Failed to delete ConfigMap for user %s", userID)
			return err
		}
		log.Infof("Succeeded in removing ConfigMap for user %s.", userID)
	}
	return nil
}
