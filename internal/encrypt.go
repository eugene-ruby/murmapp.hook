package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"os"
	"fmt"
	"io"
)

var SecretEncryptionKey []byte
var PayloadEncryptionKey []byte

func InitEncryptionKey() error {
   payloadKey := os.Getenv("ENCRYPTION_KEY")
	if payloadKey == "" || len(payloadKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY env var not set")
	}
	PayloadEncryptionKey = []byte(payloadKey)

	encKey := os.Getenv("TELEGRAM_ID_ENCRYPTION_KEY")
	if encKey == "" || len(encKey) != 32 {
		return fmt.Errorf("TELEGRAM_ID_ENCRYPTION_KEY must be 32 bytes")
	}
	SecretEncryptionKey = []byte(encKey)

	return nil
}

func EncryptWithKeyBytes(plain []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plain, nil)
	return ciphertext, nil
}


func EncryptWithKey(plain string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func DecryptWithKey(ciphertext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher init failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM init failed: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	data := ciphertext[gcm.NonceSize():]

	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plain), nil
}