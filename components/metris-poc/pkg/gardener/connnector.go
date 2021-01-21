package gardener

import gcorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

type Client struct {
	shoots []*gcorev1beta1.Shoot
}
