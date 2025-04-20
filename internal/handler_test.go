package internal

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
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
	privacyKeys = []string{"update_id"}

	secretToken := "testtoken"
	webhookID := ComputeWebhookID(secretToken, "test_salt")

	reqBody := []byte(`{"update_id": 12345}`)
	req := httptest.NewRequest("POST", "/api/webhook/"+webhookID, bytes.NewReader(reqBody))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secretToken)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("webhook_id", webhookID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mock := &MockPublisher{}

	HandleWebhook(w, req, mock)

	assert.Equal(t, 200, w.Result().StatusCode)
	assert.True(t, mock.Published)
	assert.Equal(t, "telegram.messages.in", mock.LastKey)
}

func TestHandleWebhook_InvalidToken(t *testing.T) {
	os.Setenv("SECRET_SALT", "test_salt")

	req := httptest.NewRequest("POST", "/api/webhook/invalid", nil)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("webhook_id", "invalid")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

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
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mock := &MockPublisher{}

	HandleWebhook(w, req, mock)

	assert.Equal(t, 200, w.Result().StatusCode)
	assert.False(t, mock.Published)
}
