package webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)
import _ "embed"

//go:embed config/privacy_keys.conf
var EmbeddedPrivacyKeys string

var privacyKeys []string

// LoadPrivacyKeys reads keys from embedded file and initializes encryption config
func LoadPrivacyKeys() error {
	privacyKeys = nil
	lines := strings.Split(EmbeddedPrivacyKeys, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			privacyKeys = append(privacyKeys, line)
		}
	}
	return nil
}

type FilterResult struct {
	RedactedJSON []byte
	Matched      int
	TelegramIDs  []TelegramID
}

type TelegramID struct {
	TelegramXId string
	OpenTelegramID string
}

// FilterPayload redacts sensitive data and encrypts IDs
func FilterPayload(raw []byte, secretSalt string) (FilterResult, error) {
	result := FilterResult{}
	var obj map[string]interface{}
	
	if err := json.Unmarshal(raw, &obj); err != nil {
		return result, fmt.Errorf("invalid JSON")
	}

	matched := 0
	uniqXID := map[string]bool{}
	for _, path := range privacyKeys {
		parts := strings.Split(path, ".")
		telegramID, res := applyPrivacyRule(obj, parts, secretSalt)
		if res {
			matched++

			if !uniqXID[telegramID.TelegramXId] {
				result.TelegramIDs = append(result.TelegramIDs, telegramID)
				uniqXID[telegramID.TelegramXId] = true
			}
		}
	}
	result.Matched = matched

	if matched == 0 {
		return FilterResult{}, fmt.Errorf("no privacy keys matched")
	}

	r, err := json.Marshal(obj)
	if err != nil {
		return FilterResult{}, fmt.Errorf("error marshaling redacted JSON")
	}
	result.RedactedJSON = r

	return result, nil
}

func applyPrivacyRule(root map[string]interface{}, path []string, secretSalt string) (TelegramID, bool) {
	telegramID := TelegramID{}

	var current interface{} = root
	for i, key := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return telegramID, false
		}
		val, exists := m[key]
		if !exists {
			return telegramID, false
		}
		if i == len(path)-1 {
			if key == "id" {
				var open_id string
				switch v := val.(type) {
				case float64:
					open_id = fmt.Sprintf("%.0f", v)
				case int:
					open_id = fmt.Sprintf("%d", v)
				case int64:
					open_id = fmt.Sprintf("%d", v)
				case string:
					open_id = v
				default:
					return telegramID, false
				}
				telegram_xid := TelegramXID(open_id, secretSalt)
				telegramID.TelegramXId = telegram_xid
				telegramID.OpenTelegramID = open_id
				m[key] = telegram_xid
			} else {
				if isChannel(m) && (key == "title" || key == "username") {
					return telegramID, false
				}
				m[key] = "[redacted]"
			}
			return telegramID, true
		}
		current = val
	}
	return telegramID, false
}

func isChannel(m map[string]interface{}) bool {
	t, ok := m["type"]
	if !ok {
		return false
	}
	return t == "channel"
}

func TelegramXID(telegram_id, secretSalt string) string {
	h := sha256.New()
	h.Write([]byte(telegram_id + secretSalt))
	return hex.EncodeToString(h.Sum(nil))
}
