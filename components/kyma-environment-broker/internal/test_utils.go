package internal

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// KEB tests can run in parallel resulting in concurrent access to scheme maps
// if the global scheme from client-go is used. For this reason, KEB tests each have
// their own scheme.
func NewSchemeForTests() *runtime.Scheme {
	sch := runtime.NewScheme()
	corev1.AddToScheme(sch)
	apiextensionsv1.AddToScheme(sch)
	return sch
}
