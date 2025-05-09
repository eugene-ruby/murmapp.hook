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
message.forward_from_chat.id
message.forward_from_chat.title
message.forward_from_chat.username
message.forward_origin.sender_user.id
message.forward_origin.sender_user.username
message.forward_origin.chat.id
message.forward_origin.chat.title
message.forward_origin.chat.username
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

	if !strings.Contains(redactedStr, "3155b66fa12f59c373773dd79658f85d93baa739fb1025dd67641ce1d4042a21") {
		t.Errorf("expected sha256 123 placeholder, got: %s", redactedStr)
	}
	if !strings.Contains(redactedStr, "746038628e2b64e08546fdddef7df2631008986139b24975869b839e09322204") {
		t.Errorf("expected sha256 321 placeholder, got: %s", redactedStr)
	}
	if !strings.Contains(redactedStr, "5d2218514d5e1e0e423403493b355101913751bf5e89c56ed2763171b957f51d") {
		t.Errorf("expected sha256 789 placeholder, got: %s", redactedStr)
	}

	if strings.Contains(redactedStr, "\"id\":123") || strings.Contains(redactedStr, "\"id\":321") || strings.Contains(redactedStr, "\"id\":789") {
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

func TestFilterPayload_ChannelFieldsUnredacted(t *testing.T) {
	raw := []byte(`{
		"message": {
			"forward_from_chat": {
				"id": -1001234567890,
				"title": "MyChannel",
				"username": "my_channel",
				"type": "channel"
			},
			"forward_origin": {
				"type": "channel",
				"chat": {
					"id": -1001234567899,
					"title": "MyChannel",
					"username": "my_channel",
					"type": "channel"
				}
			}
		}
	}`)

	result, err := FilterPayload(raw, secretSalt)
	if err != nil {
		t.Errorf("expected payload to pass filter, but got error: %s", err)
	}

	redactedStr := string(result.RedactedJSON)

	if strings.Contains(redactedStr, "-1001234567899") || strings.Contains(redactedStr, "-1001234567890") {
		t.Errorf("expected redacted fields, got: %s", redactedStr)
	}
	if !strings.Contains(redactedStr, "a6891bd87bf997a036e3200b181bda9f9e70350e709065c2041194cca30c7520") {
		t.Errorf("expected sha256 -1001234567890 placeholder, got: %s", redactedStr)
	}
	if !strings.Contains(redactedStr, "ff5614c16de3d759cd09c19d6c6f51e306e3656b8c6b02ef6157dc84705aa730") {
		t.Errorf("expected sha256 -1001234567899 placeholder, got: %s", redactedStr)
	}
	if !strings.Contains(redactedStr, "MyChannel") || !strings.Contains(redactedStr, "my_channel") {
		t.Errorf("expected channel title and username to be preserved, got: %s", redactedStr)
	}
}
