package runtime

import (
	"context"
	"fmt"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	rbacv1helpers "k8s.io/kubernetes/pkg/apis/rbac/v1"
)

type retryConfig struct {
	RetryAttempts int
	RetrySleep    int
}

type SAInfo struct {
	ServiceAccountName     string
	ClusterRoleName        string
	ClusterRoleBindingName string
	Namespace              string
}

const Namespace = "kyma-system"
const RUNTIME_ADMIN = "runtimeAdmin"
const RUNTIME_OPERATOR = "runtimeOperator"
const SA = "SA"
const ClusterRole = "ClusterRole"

var L2L3OperatorPolicyRule = map[string][]rbacv1.PolicyRule{
	RUNTIME_ADMIN: []rbacv1.PolicyRule{
		rbacv1helpers.NewRule("*").Groups("*").Resources("*").RuleOrDie(),
		rbacv1helpers.NewRule("*").URLs("*").RuleOrDie(),
	},
	RUNTIME_OPERATOR: []rbacv1.PolicyRule{
		rbacv1helpers.NewRule("get", "list", "watch").Groups("*").Resources("*").RuleOrDie(),
		rbacv1helpers.NewRule("get", "list", "watch").URLs("*").RuleOrDie(),
	},
}

//kubeconfig login skr, create sa, clusterrole and clusterrolebinding according to userID and roleType
func GenerateSAToken(kubeConfig []byte, userID string, roleType string) (string, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return "", err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	user := SAInfo{
		ServiceAccountName:     userID,
		ClusterRoleName:        userID,
		ClusterRoleBindingName: userID,
		Namespace:              Namespace,
	}

	rmObjects := map[string]bool{SA: false, ClusterRole: false}
	defer removeObject(clientset, rmObjects, user)

	err = createServiceAccount(clientset, user)
	if err != nil {
		return "", err
	}

	err = createClusterRole(clientset, user.ClusterRoleName, roleType)
	if err != nil {
		rmObjects[SA] = true
		return "", err
	}

	saToken, err := getSecretToken(clientset, user)
	if err != nil {
		rmObjects[SA] = true
		rmObjects[ClusterRole] = true
		return "", err
	}

	crbErr := createClusterRoleBinding(clientset, user)
	if crbErr != nil {
		rmObjects[SA] = true
		rmObjects[ClusterRole] = true
		return "", crbErr
	}
	return string(saToken), nil
}

func removeObject(c kubernetes.Interface, rmObj map[string]bool, user SAInfo) {
	config := retryConfig{3, 1}
	if rmObj[SA] {
		_, err := retry(config.RetryAttempts, time.Duration(config.RetrySleep) * time.Millisecond, func() (bool, error) { return deleteServiceAccount(c, user) })
		if err != nil {
			log.Infof("Delete service account %s failed", user.ServiceAccountName)
		}
	}
	if rmObj[ClusterRole] {
		_, err := retry(config.RetryAttempts, time.Duration(config.RetrySleep) * time.Millisecond, func() (bool, error) { return deleteClusterRole(c, user.ClusterRoleName) })
		if err != nil {
			log.Infof("Delete clusterrole %s failed", user.ServiceAccountName)
		}
	}
}

func createClusterRoleBinding(c kubernetes.Interface, user SAInfo) error {
	objectMeta, roleRef, subjects := prepareClusterRoleBindingE(user)

	existed, err := verifyClusterRoleBinding(c, user, roleRef, subjects)
	if err != nil {
		return err
	}
	if existed {
		return nil
	}
	crbinding := prepareClusterRoleBinding(objectMeta, roleRef, subjects)
	_, err = c.RbacV1().ClusterRoleBindings().Create(context.TODO(), crbinding, metav1.CreateOptions{})
	return err
}

func createClusterRole(c kubernetes.Interface, clusterRoleName string, roleType string) error {
	if roleType != RUNTIME_ADMIN && roleType != RUNTIME_OPERATOR {
		return fmt.Errorf("role %s not in [%s,%s]", roleType, RUNTIME_ADMIN, RUNTIME_OPERATOR)
	}

	crExist, err := verifyClusterRole(c, clusterRoleName, roleType)
	if err != nil {
		return err
	}
	if crExist {
		return nil
	}

	clusterrole := prepareClusterRole(clusterRoleName, roleType)
	_, err = c.RbacV1().ClusterRoles().Create(context.TODO(), clusterrole, metav1.CreateOptions{})
	return err
}

func createServiceAccount(c kubernetes.Interface, user SAInfo) error {
	saExisted, err := verifyServiceAccount(c, user)
	if err != nil {
		return err
	}
	if saExisted {
		return nil
	}

	serviceAccount := prepareServiceAccount(user)
	_, err = c.CoreV1().ServiceAccounts(user.Namespace).Create(context.TODO(), serviceAccount, metav1.CreateOptions{})
	return err
}

func deleteServiceAccount(c kubernetes.Interface, user SAInfo) (bool, error){
	err := c.CoreV1().ServiceAccounts(user.Namespace).Delete(context.TODO(), user.ServiceAccountName, metav1.DeleteOptions{})
	if err == nil || errors.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

func deleteClusterRoleBinding(c kubernetes.Interface, user SAInfo) error {
	err := c.RbacV1().ClusterRoleBindings().Delete(context.TODO(), user.ClusterRoleBindingName, metav1.DeleteOptions{})
	if err == nil || errors.IsNotFound(err) {
		return nil
	}
	return err
}

func verifyClusterRoleBinding(c kubernetes.Interface, user SAInfo, roleRef rbacv1.RoleRef, subjects []rbacv1.Subject) (bool, error) {
	crb, err := c.RbacV1().ClusterRoleBindings().Get(context.TODO(), user.ClusterRoleBindingName, metav1.GetOptions{})
	if crb != nil && err == nil {
		if reflect.DeepEqual(crb.Subjects, subjects) && reflect.DeepEqual(crb.RoleRef, roleRef) {
			return true, nil
		} else {
			err = deleteClusterRoleBinding(c, user)
			if err == nil || errors.IsNotFound(err) {
				return false, nil
			} else {
				return false, err
			}
		}
	}

	if errors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func deleteClusterRole(c kubernetes.Interface, clusterRoleName string) (bool, error) {
	err := c.RbacV1().ClusterRoles().Delete(context.TODO(), clusterRoleName, metav1.DeleteOptions{})
	if err == nil || errors.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

func verifyClusterRole(c kubernetes.Interface, clusterRoleName string, roleType string) (bool, error) {
	cr, err := c.RbacV1().ClusterRoles().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
	if cr != nil && err == nil {
		if reflect.DeepEqual(cr.Rules, L2L3OperatorPolicyRule[roleType]) {
			return true, nil
		} else {
			_, err = deleteClusterRole(c, clusterRoleName)
			if err == nil {
				return false, nil
			} else {
				return false, err
			}
		}
	}

	if errors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func verifyServiceAccount(c kubernetes.Interface, user SAInfo) (bool, error) {
	sa, err := c.CoreV1().ServiceAccounts(user.Namespace).Get(context.TODO(), user.ServiceAccountName, metav1.GetOptions{})
	if sa != nil && err == nil {
		_, err = deleteServiceAccount(c, user)
		if err == nil || errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	if errors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func getSecretToken(c kubernetes.Interface, user SAInfo) ([]byte, error) {
	// Wait for the TokenController to provision a ServiceAccount token
	config := retryConfig{3, 10}
	secretName, err := getSASecretName(config.RetryAttempts, config.RetrySleep, c, user)
	if err != nil {
		return nil, err
	}
	secret, err := c.CoreV1().Secrets(user.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data["token"], err
}

func prepareServiceAccount(user SAInfo) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      user.ServiceAccountName,
			Namespace: user.Namespace,
		},
	}
}

func prepareClusterRole(clusterRoleName string, roleType string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
		},
		Rules: L2L3OperatorPolicyRule[roleType],
	}
}

func prepareClusterRoleBindingE(user SAInfo) (metav1.ObjectMeta, rbacv1.RoleRef, []rbacv1.Subject) {
	objectMeta := metav1.ObjectMeta{
		Name: user.ClusterRoleBindingName,
	}

	roleRef := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "ClusterRole",
		Name:     user.ClusterRoleName,
	}
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      user.ServiceAccountName,
			Namespace: user.Namespace,
		},
	}
	return objectMeta, roleRef, subjects
}

func prepareClusterRoleBinding(objectMeta metav1.ObjectMeta, roleRef rbacv1.RoleRef, subjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: objectMeta,
		RoleRef:    roleRef,
		Subjects:   subjects,
	}
}

func retry[T any](attempts int, d time.Duration, f func() (T, error)) (result T, err error) {
    for i := 0; i < attempts; i++ {
        if i > 0 {
			time.Sleep(d)
        }
        result, err = f()
        if err == nil {
            return result, nil
        }
    }
    return result, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func getSASecretName(attempts int, sleep int, c kubernetes.Interface, user SAInfo) (string, error) {
	var err error
	sa, err := retry(attempts, time.Duration(sleep) * time.Millisecond, func() (*corev1.ServiceAccount, error) { return getServiceAccount(c, user) })
    if len(sa.Secrets) != 0 {
		return sa.Secrets[0].Name, nil
	}
	return "", err
}

func getServiceAccount(c kubernetes.Interface, user SAInfo) (*corev1.ServiceAccount, error){
	sa, err := c.CoreV1().ServiceAccounts(user.Namespace).Get(context.TODO(), user.ServiceAccountName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return sa, err
}