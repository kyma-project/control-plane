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

func (e *Encrypter) EncryptSMCreds(pp *internal.ProvisioningParameters) error {
	if pp.ErsContext.ServiceManager == nil && pp.ErsContext.SMOperatorCredentials == nil {
		return nil
	}
	var err error
	encrypted := internal.ERSContext{}
	if pp.ErsContext.ServiceManager != nil {
		creds := pp.ErsContext.ServiceManager.Credentials.BasicAuth
		var username, password []byte
		if creds.Username != "" {
			username, err = e.Encrypt([]byte(creds.Username))
			if err != nil {
				return errors.Wrap(err, "while encrypting username")
			}
		}
		if creds.Password != "" {
			password, err = e.Encrypt([]byte(creds.Password))
			if err != nil {
				return errors.Wrap(err, "while encrypting password")
			}
		}
		encrypted.ServiceManager = &internal.ServiceManagerEntryDTO{
			URL: pp.ErsContext.ServiceManager.URL,
			Credentials: internal.ServiceManagerCredentials{
				BasicAuth: internal.ServiceManagerBasicAuth{
					Password: string(password),
					Username: string(username),
				},
			},
		}
	}
	if pp.ErsContext.SMOperatorCredentials != nil {
		creds := pp.ErsContext.SMOperatorCredentials
		var clientID, clientSecret []byte
		if creds.ClientID != "" {
			clientID, err = e.Encrypt([]byte(creds.ClientID))
			if err != nil {
				return errors.Wrap(err, "while encrypting ClientID")
			}
		}
		if creds.ClientSecret != "" {
			clientSecret, err = e.Encrypt([]byte(creds.ClientSecret))
			if err != nil {
				return errors.Wrap(err, "while encrypting ClientSecret")
			}
		}
		encrypted.SMOperatorCredentials = &internal.ServiceManagerOperatorCredentials{
			ClientID:          string(clientID),
			ClientSecret:      string(clientSecret),
			ServiceManagerURL: creds.ServiceManagerURL,
			URL:               creds.URL,
			XSAppName:         creds.XSAppName,
		}
	}
	pp.ErsContext.ServiceManager = encrypted.ServiceManager
	pp.ErsContext.SMOperatorCredentials = encrypted.SMOperatorCredentials
	return nil
}

func (e *Encrypter) DecryptSMCreds(pp *internal.ProvisioningParameters) error {
	if pp.ErsContext.ServiceManager == nil && pp.ErsContext.SMOperatorCredentials == nil {
		return nil
	}
	var err error
	var username, password, clientID, clientSecret []byte
	if pp.ErsContext.ServiceManager != nil {
		creds := pp.ErsContext.ServiceManager.Credentials.BasicAuth
		if creds.Username != "" {
			username, err = e.Decrypt([]byte(creds.Username))
			if err != nil {
				return errors.Wrap(err, "while decrypting username")
			}
		}
		if creds.Password != "" {
			password, err = e.Decrypt([]byte(creds.Password))
			if err != nil {
				return errors.Wrap(err, "while decrypting password")
			}
		}
	}
	if pp.ErsContext.SMOperatorCredentials != nil {
		creds := pp.ErsContext.SMOperatorCredentials
		if creds.ClientID != "" {
			clientID, err = e.Decrypt([]byte(creds.ClientID))
			if err != nil {
				return errors.Wrap(err, "while decrypting ClientID")
			}
		}
		if creds.ClientSecret != "" {
			clientSecret, err = e.Decrypt([]byte(creds.ClientSecret))
			if err != nil {
				return errors.Wrap(err, "while decrypting ClientSecret")
			}
		}
	}
	if len(password) != 0 {
		pp.ErsContext.ServiceManager.Credentials.BasicAuth.Password = string(password)
	}
	if len(username) != 0 {
		pp.ErsContext.ServiceManager.Credentials.BasicAuth.Username = string(username)
	}
	if len(clientID) != 0 {
		pp.ErsContext.SMOperatorCredentials.ClientID = string(clientID)
	}
	if len(clientSecret) != 0 {
		pp.ErsContext.SMOperatorCredentials.ClientSecret = string(clientSecret)
	}
	return nil
}
