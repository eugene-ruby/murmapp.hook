package config_test

import (
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/require"
	"murmapp.hook/internal/config"
)

func TestLoadConfig_Success(t *testing.T) {
	salt := []byte("somesalt")
	payloadKey := []byte("payload-32-byte-key-abc123456789")

	cfg, err := config.LoadConfig()
	require.NoError(t, err)

	require.Equal(t, "3998", cfg.AppPort)
	require.Equal(t, "/api/webhook", cfg.WebhookPath)
	require.Equal(t, "amqp://guest:guest@localhost:5672", cfg.RabbitMQ.URL)
	require.Equal(t, salt, cfg.Encryption.SecretSalt)
	require.Equal(t, payloadKey, cfg.Encryption.PayloadEncryptionKey)
	require.IsType(t, &rsa.PublicKey{}, cfg.Encryption.CasterPublicRSAKey)
}
