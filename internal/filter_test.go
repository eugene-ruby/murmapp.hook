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

func TestFilterPayload_MissingKey(t *testing.T) {
	raw := []byte(`{
		"message": {
			"from": {"uuid": 123},
			"chat": {"uuid": 456}
		}
	}`)

	_, ok, reason := FilterPayload(raw)
	if ok {
		t.Errorf("expected payload to be rejected due to no matched keys")
	}
	if !strings.Contains(reason, "no privacy keys matched") {
		t.Errorf("expected reason to mention unmatched keys, got: %s", reason)
	}
}

func TestFilterPayload_InvalidJSON(t *testing.T) {
	badJSON := []byte(`{ this is not valid JSON }`)
	_, ok, reason := FilterPayload(badJSON)
	if ok || !strings.Contains(reason, "invalid JSON") {
		t.Errorf("expected invalid JSON error, got: %s", reason)
	}
}
