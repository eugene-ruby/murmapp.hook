package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	_ "embed"
)

//go:embed config/privacy_keys.conf
var EmbeddedPrivacyKeys string

var (
	privacyKeys       []string
	secretSalt        string
	encryptionKey     []byte // for telegram_id
	textEncryptionKey []byte // for message text
)

// LoadPrivacyKeys reads keys from embedded file and initializes encryption config
func LoadPrivacyKeys() error {
	lines := strings.Split(EmbeddedPrivacyKeys, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			privacyKeys = append(privacyKeys, line)
		}
	}

	secretSalt = os.Getenv("SECRET_SALT")
	if secretSalt == "" {
		return fmt.Errorf("SECRET_SALT env var not set")
	}

	encKey := os.Getenv("TELEGRAM_ID_ENCRYPTION_KEY")
	if encKey == "" || len(encKey) != 32 {
		return fmt.Errorf("TELEGRAM_ID_ENCRYPTION_KEY must be 32 bytes")
	}
	encryptionKey = []byte(encKey)

	textKey := os.Getenv("ENCRYPTION_KEY")
	if textKey == "" || len(textKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be 32 bytes")
	}
	textEncryptionKey = []byte(textKey)

	return nil
}

// FilterPayload redacts sensitive data and encrypts IDs/text
func FilterPayload(raw []byte) (redacted []byte, ok bool, reason string) {
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, false, "invalid JSON"
	}

	matched := 0
	for _, path := range privacyKeys {
		parts := strings.Split(path, ".")
		if applyPrivacyRule(obj, parts) {
			matched++
		}
	}

	// encrypt all known text locations
	encryptTextAtPath(obj, []string{"message", "text"})
	encryptTextAtPath(obj, []string{"message", "reply_to_message", "text"})
	encryptTextAtPath(obj, []string{"channel_post", "text"})

	if matched == 0 {
		return raw, false, "no privacy keys matched"
	}

	result, err := json.Marshal(obj)
	if err != nil {
		return nil, false, "error marshaling redacted JSON"
	}

	return result, true, ""
}

func applyPrivacyRule(root map[string]interface{}, path []string) bool {
	var current interface{} = root
	for i, key := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return false
		}
		val, exists := m[key]
		if !exists {
			return false
		}
		if i == len(path)-1 {
			if key == "id" {
				str := fmt.Sprintf("%v", val)
				switch v := val.(type) {
				case float64:
					str = fmt.Sprintf("%.0f", v)
				case string:
					str = v
				default:
					return false
				}
				encrypted, err := encryptWithKey(str, encryptionKey)
				if err != nil {
					return false
				}
				m[key] = encrypted
			} else {
				m[key] = "[redacted]"
			}
			return true
		}
		current = val
	}
	return false
}

func encryptTextAtPath(root map[string]interface{}, path []string) {
	var current interface{} = root
	for i, key := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return
		}
		val, exists := m[key]
		if !exists {
			return
		}
		if i == len(path)-1 {
			str, ok := val.(string)
			if !ok || str == "" {
				return
			}
			encrypted, err := encryptWithKey(str, textEncryptionKey)
			if err == nil {
				m[key] = encrypted
			}
			return
		}
		current = val
	}
}

func encryptWithKey(plain string, key []byte) (string, error) {
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
