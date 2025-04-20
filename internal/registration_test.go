package internal

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"os"
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

func TestDecryptApiKeyInRequest(t *testing.T) {
	_ = os.Setenv("ENCRYPTION_KEY", "01234567890123456789012345678901")
	_ = os.Setenv("TELEGRAM_ID_ENCRYPTION_KEY", "12345678901234567890123456789012")
	_ = InitEncryptionKey()

	original := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	encrypted, err := EncryptWithKey(original, SecretEncryptionKey)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	ciphertext, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	decrypted, err := DecryptWithKey(ciphertext, SecretEncryptionKey)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}
	if decrypted != original {
		t.Errorf("expected decrypted API key to be '%s', got '%s'", original, decrypted)
	}
}

type mockWebhookRegistrar struct {
	calledWithToken string
	calledURL      string
}

func (m *mockWebhookRegistrar) register(apiKey, url, token string) error {
	m.calledWithToken = apiKey
	m.calledURL = url
	return nil
}

func TestRegisterWebhookRequestFlow(t *testing.T) {
	_ = os.Setenv("ENCRYPTION_KEY", "01234567890123456789012345678901")
	_ = os.Setenv("TELEGRAM_ID_ENCRYPTION_KEY", "12345678901234567890123456789012")
	_ = InitEncryptionKey()

	original := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	encryptedStr, err := EncryptWithKey(original, SecretEncryptionKey)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	ciphertext, err := base64.URLEncoding.DecodeString(encryptedStr)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}

	dummyRegistrar := &mockWebhookRegistrar{}
	url := "https://bot.example.com/hook"
	token := "abc123"

	// simulate what registrations_consumer would do:
	decrypted, err := DecryptWithKey(ciphertext, SecretEncryptionKey)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}
	_ = dummyRegistrar.register(decrypted, url, token)

	if dummyRegistrar.calledWithToken != original {
		t.Errorf("expected token to be '%s', got '%s'", original, dummyRegistrar.calledWithToken)
	}
	if dummyRegistrar.calledURL != url {
		t.Errorf("expected URL to be '%s', got '%s'", url, dummyRegistrar.calledURL)
	}
}
