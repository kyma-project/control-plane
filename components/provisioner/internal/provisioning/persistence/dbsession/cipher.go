package dbsession

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type encryptFunc func([]byte) ([]byte, error)
type decryptFunc func([]byte) ([]byte, error)

func newEncryptFunc(key []byte) encryptFunc {
	return func(obj []byte) ([]byte, error) { return encrypt(key, obj) }
}

func newDecryptFunc(key []byte) decryptFunc {
	return func(obj []byte) ([]byte, error) { return decrypt(key, obj) }
}

func newEmptyFunc(_ []byte) func([]byte) ([]byte, error) {
	return func(bytes []byte) ([]byte, error) { return bytes, nil }
}

func encrypt(key, obj []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
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

func decrypt(key, obj []byte) ([]byte, error) {
	obj, err := base64.StdEncoding.DecodeString(string(obj))
	if err != nil {
		return nil, fmt.Errorf("while decoding object: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(obj) < aes.BlockSize {
		return nil, fmt.Errorf("cipher text is too short")
	}
	iv := obj[:aes.BlockSize]
	obj = obj[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(obj, obj)
	data, err := base64.StdEncoding.DecodeString(string(obj))
	if err != nil {
		return nil, fmt.Errorf("while decoding object: %w", err)
	}
	return data, nil
}
