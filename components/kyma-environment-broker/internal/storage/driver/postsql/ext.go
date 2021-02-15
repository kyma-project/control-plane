package postsql

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

type Cipher interface {
	Encrypt(text []byte) ([]byte, error)
	Decrypt(text []byte) ([]byte, error)

	// methods used to encrypt/decrypt SM credentials
	EncryptBasicAuth(pp *internal.ProvisioningParameters) error
	DecryptBasicAuth(pp *internal.ProvisioningParameters) error
}
