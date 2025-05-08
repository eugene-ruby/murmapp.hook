package webhook_test

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/eugene-ruby/xconnect/rabbitmq/mocks"
	"github.com/eugene-ruby/xencryptor/xsecrets"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"murmapp.hook/internal/config"
	"murmapp.hook/internal/webhook"
	hookpb "murmapp.hook/proto"
)

func TestHandleWebhook_success(t *testing.T) {
	// Load encryption config
	conf, err := config.LoadConfig()
	require.NoError(t, err)

	salt := string(conf.Encryption.SecretSalt)
	openID := "456"
	expectedXID := webhook.TelegramXID(openID, salt)

	// Simulate incoming JSON with sensitive ID
	payload := map[string]any{
		"message": map[string]any{
			"from": map[string]any{
				"id": openID,
			},
			"text": "hi",
		},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)

	// Compute valid webhook ID based on known token and salt
	token := "abc123"
	webhookID := webhook.ComputeWebhookID(token, salt)

	// Build HTTP request with context and webhook_id param
	req := httptest.NewRequest("POST", "/hook", bytes.NewReader(raw))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", token)
	req.RemoteAddr = "1.2.3.4"

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("webhook_id", webhookID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	// Use mocked RabbitMQ channel
	channel := mocks.NewMockChannel()
	handler := &webhook.OutboundHandler{
		Config:  *conf,
		Channel: channel,
	}

	// Execute handler
	rec := httptest.NewRecorder()
	webhook.HandleWebhook(rec, req, handler)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, channel.PublishedMessages, 2)

	var foundPayload, foundEncryptedID bool

	// Inspect published messages
	for _, msg := range channel.PublishedMessages {
		switch msg.RoutingKey {
		case "telegram.messages.in":
			var p hookpb.TelegramWebhookPayload
			err := proto.Unmarshal(msg.Body, &p)
			require.NoError(t, err)
			require.Equal(t, webhookID, p.WebhookId)
			foundPayload = true

		case "telegram.encrypted.id":
			var enc hookpb.EncryptedTelegramID
			err := proto.Unmarshal(msg.Body, &enc)
			require.NoError(t, err)
			require.Equal(t, expectedXID, enc.TelegramXid)

			// Decrypt and verify original ID
			decryptedID, err := xsecrets.RSADecryptBytes(enc.EncryptedId, privateKey(t))
			require.NoError(t, err)
			require.Equal(t, openID, string(decryptedID))
			foundEncryptedID = true
		}
	}

	require.True(t, foundPayload, "expected TelegramWebhookPayload to be published")
	require.True(t, foundEncryptedID, "expected EncryptedTelegramID to be published")
}

func TestHandleWebhook_invalidToken(t *testing.T) {
	conf, err := config.LoadConfig()
	require.NoError(t, err)

	raw := []byte(`{"message": {"from": {"id": "123"}}}`)

	req := httptest.NewRequest("POST", "/hook", bytes.NewReader(raw))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-token")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("webhook_id", "invalid-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	channel := mocks.NewMockChannel()
	handler := &webhook.OutboundHandler{Config: *conf, Channel: channel}

	rec := httptest.NewRecorder()
	webhook.HandleWebhook(rec, req, handler)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Len(t, channel.PublishedMessages, 0)
}

func TestHandleWebhook_invalidJSON(t *testing.T) {
	conf, _ := config.LoadConfig()

	req := httptest.NewRequest("POST", "/hook", bytes.NewReader([]byte(`{ not json }`)))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "abc")
	rctx := chi.NewRouteContext()
	webhookID := webhook.ComputeWebhookID("abc", string(conf.Encryption.SecretSalt))
	rctx.URLParams.Add("webhook_id", webhookID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	channel := mocks.NewMockChannel()
	handler := &webhook.OutboundHandler{Config: *conf, Channel: channel}

	rec := httptest.NewRecorder()
	webhook.HandleWebhook(rec, req, handler)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, channel.PublishedMessages, 0)
}

func TestHandleWebhook_payloadNoMatches(t *testing.T) {
	conf, _ := config.LoadConfig()

	raw := []byte(`{"message": {"text": "nothing to redact"}}`)
	token := "abc"
	webhookID := webhook.ComputeWebhookID(token, string(conf.Encryption.SecretSalt))

	req := httptest.NewRequest("POST", "/hook", bytes.NewReader(raw))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", token)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("webhook_id", webhookID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := &webhook.OutboundHandler{
		Config:  *conf,
		Channel: mocks.NewMockChannel(),
	}

	rec := httptest.NewRecorder()
	webhook.HandleWebhook(rec, req, handler)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, handler.Channel.(*mocks.MockChannel).PublishedMessages, 0)
}

// privateKey loads the RSA private key from an environment variable
func privateKey(t *testing.T) *rsa.PrivateKey {
	raw := os.Getenv("CASTER_PRIVATE_KEY_RAW_BASE64")
	require.NotEmpty(t, raw, "CASTER_PRIVATE_KEY_RAW_BASE64 must be set")

	decoded, err := base64.RawStdEncoding.DecodeString(raw)
	require.NoError(t, err)

	block, _ := pem.Decode(decoded)
	require.NotNil(t, block, "failed to decode PEM block")

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	return priv
}
