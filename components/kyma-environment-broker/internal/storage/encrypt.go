package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
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
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

func (e *Encrypter) Decrypt(obj []byte) ([]byte, error) {
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
		return nil, err
	}
	return data, nil
}
