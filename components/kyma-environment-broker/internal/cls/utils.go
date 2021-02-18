package cls

import (
	"encoding/json"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
)

func EncryptOverrides(secretKey string, overrides *ClsOverrides) (string, error) {
	ovrs, err := json.Marshal(*overrides)
	if err != nil {
		return "", errors.Wrap(err, "while encoding cls overrides")
	}
	encrypter := storage.NewEncrypter(secretKey)
	encryptedOverrides, err := encrypter.Encrypt(ovrs)
	if err != nil {
		return "", errors.Wrap(err, "while encrypting cls overrides")
	}
	return string(encryptedOverrides), nil
}

func DecryptOverrides(secretKey string, encryptedOverrides string) (*ClsOverrides, error) {
	encrypter := storage.NewEncrypter(secretKey)
	decryptedOverrides, err := encrypter.Decrypt([]byte(encryptedOverrides))
	if err != nil {
		return nil, errors.Wrap(err, "while decrypting eventing overrides")
	}
	clsOverrides := ClsOverrides{}
	if err := json.Unmarshal(decryptedOverrides, &clsOverrides); err != nil {
		return nil, errors.Wrap(err, "while unmarshall eventing overrides")
	}
	return &clsOverrides, nil
}
