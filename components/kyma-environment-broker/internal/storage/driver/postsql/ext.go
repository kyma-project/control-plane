package postsql

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

type Cipher interface {
	Encrypt(text []byte) ([]byte, error)
	Decrypt(text []byte) ([]byte, error)

	// methods used to encrypt/decrypt SM credentials
	EncryptSMCreds(pp *internal.ProvisioningParameters) error
	DecryptSMCreds(pp *internal.ProvisioningParameters) error

	// methods used to encrypt/decrypt kubeconfig
	EncryptKubeconfig(pp *internal.ProvisioningParameters) error
	DecryptKubeconfig(pp *internal.ProvisioningParameters) error
}
