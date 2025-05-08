package webhook

import (
	"strings"
	"testing"
)
var secretSalt string

func init() {
	EmbeddedPrivacyKeys = `
message.from.id
message.from.first_name
message.from.username
message.chat.id
message.forward_from.id
message.forward_origin.sender_user.id
`
	_ = LoadPrivacyKeys()
	secretSalt = "testSecretSalt"
}

func TestFilterPayload_FullMatch(t *testing.T) {
	raw := []byte(`{
		"message": {
			"from": {"id": 123, "first_name": "Eugene", "username": "anonymous"},
			"chat": {"id": 123},
			"forward_from": {"id": 789},
			"forward_origin": {"sender_user": {"id": "321"}},
			"text": "some message"
		}
	}`)

	result, err := FilterPayload(raw, secretSalt)
	if err != nil {
		t.Errorf("expected payload to pass filter, but got error: %s", err)
	}

	redactedStr := string(result.RedactedJSON)

	if strings.Contains(redactedStr, "Eugene") || strings.Contains(redactedStr, "anonymous") {
		t.Errorf("expected redacted fields, got: %s", redactedStr)
	}

	if strings.Count(redactedStr, "[redacted]") != 2 {
		t.Errorf("expected [redacted] to appear 2 times, got %d: %s",
			strings.Count(redactedStr, "[redacted]"), redactedStr)
	}

	if !strings.Contains(redactedStr, "06c8ff76e137028539e8d29fd966c301d62ab7e9") {
		t.Errorf("expected sha1 123 placeholder, got: %s", redactedStr)
	}
	if !strings.Contains(redactedStr, "7a461bd11eb9a8b78eef182ec544d99b51203535") {
		t.Errorf("expected sha1 321 placeholder, got: %s", redactedStr)
	}

	if strings.Contains(redactedStr, "\"id\":123") || strings.Contains(redactedStr, "\"id\":321") {
		t.Errorf("expected id to be encrypted, got: %s", redactedStr)
	}
}

func TestFilterPayload_MissingKey(t *testing.T) {
	raw := []byte(`{
		"message": {
			"from": {"uuid": 123},
			"chat": {"uuid": 456}
		}
	}`)

	_, err := FilterPayload(raw, secretSalt)
	if err == nil {
		t.Errorf("expected payload to be rejected due to no matched keys")
	}
	if !strings.Contains(err.Error(), "no privacy keys matched") {
		t.Errorf("expected reason to mention unmatched keys, got: %s", err)
	}
}

func TestFilterPayload_InvalidJSON(t *testing.T) {
	badJSON := []byte(`{ this is not valid JSON }`)

	_, err := FilterPayload(badJSON, secretSalt)
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected invalid JSON error, got: %v", err)
	}
}

func TestFilterPayload_TextEncryptionOnly(t *testing.T) {
	raw := []byte(`{
		"message": {
			"text": "super secret"
		},
		"channel_post": {
			"text": "channel announcement"
		}
	}`)

	_, err := FilterPayload(raw, secretSalt)
	if err == nil {
		t.Errorf("expected payload to be dropped because no ids matched")
	}
	if !strings.Contains(err.Error(), "no privacy keys matched") {
		t.Errorf("expected reason to mention no privacy match, got: %s", err)
	}
}
