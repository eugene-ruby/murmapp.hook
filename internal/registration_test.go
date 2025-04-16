package internal

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

func TestComputeWebhookID(t *testing.T) {
	token := "secret123"
	salt := "mysalt"

	expected := sha1.Sum([]byte(token + salt))
	expectedStr := hex.EncodeToString(expected[:])

	got := ComputeWebhookID(token, salt)
	if got != expectedStr {
		t.Errorf("expected %s, got %s", expectedStr, got)
	}
}

func TestGenerateSecretToken_Length(t *testing.T) {
	token := generateSecretToken()
	if len(token) != 32 {
		t.Errorf("expected token length 32, got %d", len(token))
	}
}
