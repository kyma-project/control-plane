package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
)

func NewEncrypter(secretKey string) *Encrypter {
	return &Encrypter{key: []byte(secretKey)}
}

type Encrypter struct {
	key []byte
}

func (e *Encrypter) Encrypt(obj []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(obj)
	bytes := make([]byte, aes.BlockSize+len(b))
	iv := bytes[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(bytes[aes.BlockSize:], []byte(b))

	return []byte(base64.StdEncoding.EncodeToString(bytes)), nil
}

func (e *Encrypter) Decrypt(obj []byte) ([]byte, error) {
	obj, err := base64.StdEncoding.DecodeString(string(obj))
	if err != nil {
		return nil, errors.Wrap(err, "while decoding object")
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	if len(obj) < aes.BlockSize {
		return nil, errors.New("cipher text is too short")
	}
	iv := obj[:aes.BlockSize]
	obj = obj[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(obj, obj)
	data, err := base64.StdEncoding.DecodeString(string(obj))
	if err != nil {
		return nil, errors.Wrap(err, "while decoding decrypted object")
	}
	return data, nil
}

func (e *Encrypter) EncryptBasicAuth(pp *internal.ProvisioningParameters) error {
	if pp.ErsContext.ServiceManager == nil {
		return nil
	}
	creds := pp.ErsContext.ServiceManager.Credentials.BasicAuth
	if creds.Username == "" || creds.Password == "" {
		return nil
	}
	username, err := e.Encrypt([]byte(pp.ErsContext.ServiceManager.Credentials.BasicAuth.Username))
	if err != nil {
		return errors.Wrap(err, "while encrypting username")
	}
	password, err := e.Encrypt([]byte(pp.ErsContext.ServiceManager.Credentials.BasicAuth.Password))
	if err != nil {
		return errors.Wrap(err, "while encrypting password")
	}

	pp.ErsContext.ServiceManager = &internal.ServiceManagerEntryDTO{
		Credentials: internal.ServiceManagerCredentials{
			BasicAuth: internal.ServiceManagerBasicAuth{
				Username: string(username),
				Password: string(password),
			}},
		URL: pp.ErsContext.ServiceManager.URL,
	}

	return nil
}

func (e *Encrypter) DecryptBasicAuth(pp *internal.ProvisioningParameters) error {
	if pp.ErsContext.ServiceManager == nil {
		return nil
	}
	creds := pp.ErsContext.ServiceManager.Credentials.BasicAuth
	if creds.Username == "" || creds.Password == "" {
		return nil
	}
	username, err := e.Decrypt([]byte(creds.Username))
	if err != nil {
		return errors.Wrap(err, "while decrypting username")
	}
	password, err := e.Decrypt([]byte(creds.Password))
	if err != nil {
		return errors.Wrap(err, "while decrypting password")
	}

	pp.ErsContext.ServiceManager.Credentials.BasicAuth.Username = string(username)
	pp.ErsContext.ServiceManager.Credentials.BasicAuth.Password = string(password)

	return nil
}
