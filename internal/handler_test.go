package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	hookpb "murmapp.hook/proto"
	"google.golang.org/protobuf/proto"
)

type MockPublisher struct {
	Published bool
	LastBody  []byte
	LastKey   string
}

func (m *MockPublisher) Publish(exchange, routingKey string, body []byte) error {
	m.Published = true
	m.LastBody = body
	m.LastKey = routingKey
	return nil
}

func (m *MockPublisher) Close() {}

func TestHandleWebhook_ValidToken(t *testing.T) {
	os.Setenv("SECRET_SALT", "test_salt")
	os.Setenv("ENCRYPTION_KEY", "01234567890123456789012345678901")
	os.Setenv("TELEGRAM_ID_ENCRYPTION_KEY", "12345678901234567890123456789012")
	_ = InitEncryptionKey()
	privacyKeys = []string{"first_name"}

	secretToken := "testtoken"
	webhookID := ComputeWebhookID(secretToken, "test_salt")

	reqBody := []byte(`{"update_id": 12345, "first_name": "bob"}`)
	req := httptest.NewRequest("POST", "/api/webhook/"+webhookID, bytes.NewReader(reqBody))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretToken)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("webhook_id", webhookID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	mock := &MockPublisher{}

	HandleWebhook(w, req, mock)

	assert.Equal(t, 200, w.Result().StatusCode)
	assert.True(t, mock.Published)
	assert.Equal(t, "telegram.messages.in", mock.LastKey)

	var payload hookpb.TelegramWebhookPayload
	err := proto.Unmarshal(mock.LastBody, &payload)
	assert.NoError(t, err)
	assert.Equal(t, webhookID, payload.WebhookId)
	assert.NotEmpty(t, payload.EncryptedPayload)
	assert.True(t, len(payload.EncryptedPayload) > 16)

	// ✅ Now verify decrypted content
	decrypted, err := DecryptWithKey(payload.EncryptedPayload, PayloadEncryptionKey)
	assert.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal([]byte(decrypted), &m)
	assert.NoError(t, err)
	assert.Contains(t, m, "update_id")
	assert.Equal(t, string("[redacted]"), m["first_name"])
	assert.Equal(t, float64(12345), m["update_id"])
}

func TestHandleWebhook_InvalidToken(t *testing.T) {
	os.Setenv("SECRET_SALT", "test_salt")
	req := httptest.NewRequest("POST", "/api/webhook/invalid", nil)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("webhook_id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	mock := &MockPublisher{}

	HandleWebhook(w, req, mock)

	assert.Equal(t, 403, w.Result().StatusCode)
	assert.False(t, mock.Published)
}

func TestHandleWebhook_FilterDropsPayload(t *testing.T) {
	os.Setenv("SECRET_SALT", "test_salt")
	privacyKeys = []string{"must.exist"}

	secretToken := "dropit"
	webhookID := ComputeWebhookID(secretToken, "test_salt")

	reqBody := []byte(`{"something_else": true}`)
	req := httptest.NewRequest("POST", "/api/webhook/"+webhookID, bytes.NewReader(reqBody))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretToken)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("webhook_id", webhookID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	mock := &MockPublisher{}

	HandleWebhook(w, req, mock)

	assert.Equal(t, 200, w.Result().StatusCode)
	assert.False(t, mock.Published)
}
