package postsql

type Cipher interface {
	Encrypt(text []byte) ([]byte, error)
	Decrypt(text []byte) ([]byte, error)
}
