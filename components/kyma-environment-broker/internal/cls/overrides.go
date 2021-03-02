package cls

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls/templates"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
)

func EncryptOverrides(secretKey string, overrides *OverrideParams) (string, error) {
	ovrs, err := json.Marshal(*overrides)
	if err != nil {
		return "", errors.Wrap(err, "while marshalling cls overrides")
	}
	encrypter := storage.NewEncrypter(secretKey)
	encryptedOverrides, err := encrypter.Encrypt(ovrs)
	if err != nil {
		return "", errors.Wrap(err, "while encrypting cls overrides")
	}
	return string(encryptedOverrides), nil
}

func DecryptOverrides(secretKey string, encryptedOverrides string) (*OverrideParams, error) {
	encrypter := storage.NewEncrypter(secretKey)
	decryptedOverrides, err := encrypter.Decrypt([]byte(encryptedOverrides))
	if err != nil {
		return nil, errors.Wrap(err, "while decrypting eventing overrides")
	}
	var overrideParams OverrideParams
	if err := json.Unmarshal(decryptedOverrides, &overrideParams); err != nil {
		return nil, errors.Wrap(err, "while unmarshalling eventing overrides")
	}
	return &overrideParams, nil
}

func GetExtraConfTemplate() (*template.Template, error) {
	tpl, err := template.New("fluent-bit-cls-extra-conf").Parse(templates.FluentBitExtraConf)
	if err != nil {
		return nil, err
	}
	return tpl, nil
}

func RenderOverrides(data interface{}, tmp *template.Template) (string, error) {
	var flOutputs bytes.Buffer
	err := tmp.Execute(&flOutputs, data)
	if err != nil {
		return "", errors.Wrapf(err, "Template error while injecting cls overrides: %v", err)
	}
	return flOutputs.String(), nil
}
