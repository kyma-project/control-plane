package postsql

type Cipher interface {
	Encrypt(text string) (string, error)
	Decrypt(text string) (string, error)
}
