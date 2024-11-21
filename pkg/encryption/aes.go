package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
)

type Encryptor struct {
	key []byte
}

func NewEncryptor(key []byte) *Encryptor {
	return &Encryptor{key: key}
}

func (e *Encryptor) Encrypt(plaintext string) (string, string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plaintext))

	return base64.RawStdEncoding.EncodeToString(ciphertext),
		base64.RawStdEncoding.EncodeToString(iv),
		nil
}

func (e *Encryptor) Decrypt(ciphertext, iv string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	decodedCiphertext, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	decodedIV, err := base64.RawStdEncoding.DecodeString(iv)
	if err != nil {
		return "", err
	}

	decodedCiphertext = decodedCiphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, decodedIV)
	stream.XORKeyStream(decodedCiphertext, decodedCiphertext)

	return string(decodedCiphertext), nil
}
