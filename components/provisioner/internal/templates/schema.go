package templates

import (
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pkg/errors"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

func defaultScheme() (*runtime.Scheme, error) {
	resourcesSchema := runtime.NewScheme()

	var addToSchemes = []func(*runtime.Scheme) error{
		scheme.AddToScheme,
		apiextensionsv1beta1.AddToScheme,
		v1beta1.AddToScheme,
	}

	for _, f := range addToSchemes {
		err := f(resourcesSchema)
		if err != nil {
			return nil, errors.Wrap(err, "failed to add types to schema")
		}
	}

	return resourcesSchema, nil
}

func defaultEncoder() (runtime.Encoder, error) {
	resourceScheme, err := defaultScheme()
	if err != nil {
		return nil, err
	}
	codecs := serializer.NewCodecFactory(resourceScheme)
	info, _ := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), runtime.ContentTypeYAML)

	return info.Serializer, nil
}
