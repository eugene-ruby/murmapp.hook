package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)
import _ "embed"

//go:embed config/privacy_keys.conf
var EmbeddedPrivacyKeys string

var (
	privacyKeys   []string
	secretSalt    string
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
	
	return nil
}

// FilterPayload redacts sensitive data and encrypts IDs
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
				encrypted, err := EncryptWithKey(str, SecretEncryptionKey)
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
