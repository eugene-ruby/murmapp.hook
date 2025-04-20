package internal

import (
	"os"
	"strings"
	"testing"
)

func init() {
	EmbeddedPrivacyKeys = `
message.from.id
message.from.first_name
message.from.username
message.chat.id
message.forward_from.id
message.forward_origin.sender_user.id
`
	os.Setenv("SECRET_SALT", "test_salt")
	os.Setenv("TELEGRAM_ID_ENCRYPTION_KEY", "01234567890123456789012345678901")
	_ = LoadPrivacyKeys()
}


func TestFilterPayload_FullMatch(t *testing.T) {
	raw := []byte(`{
		"message": {
			"from": {"id": 123, "first_name": "Eugene", "username": "anonymous"},
			"chat": {"id": 123},
			"forward_from": {"id": 789},
			"forward_origin": {"sender_user": {"id": 321}}
		}
	}`)

	redacted, ok, reason := FilterPayload(raw)
	if !ok {
		t.Errorf("expected payload to pass filter, but got rejected: %s", reason)
	}

	redactedStr := string(redacted)
	if strings.Contains(redactedStr, "Eugene") || strings.Contains(redactedStr, "anonymous") {
		t.Errorf("expected redacted fields, got: %s", redactedStr)
	}
	if !strings.Contains(redactedStr, "[redacted]") {
		t.Errorf("expected [redacted] placeholder, got: %s", redactedStr)
	}
	if strings.Contains(redactedStr, "\"id\":123") {
		t.Errorf("expected id to be encrypted, but got raw value: %s", redactedStr)
	}
}

func LoadPrivacyKeysFromAbsolutePath(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			privacyKeys = append(privacyKeys, line)
		}
	}
	return nil
}

func TestFilterPayload_MissingKey(t *testing.T) {
	os.Setenv("SECRET_SALT", "test_salt")
	os.Setenv("TELEGRAM_ID_ENCRYPTION_KEY", "01234567890123456789012345678901")
	privacyKeys = []string{"message.from.id", "message.chat.id", "message.missing_field"}

	raw := []byte(`{
		"message": {
			"from": {"id": 123},
			"chat": {"id": 456}
		}
	}`)

	_, ok, reason := FilterPayload(raw)
	if ok {
		t.Errorf("expected payload to be rejected due to missing field")
	}
	if !strings.Contains(reason, "missing") {
		t.Errorf("expected reason to mention missing key, got: %s", reason)
	}
}
