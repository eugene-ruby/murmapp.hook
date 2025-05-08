package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"

	"fmt"
	"os"

	"github.com/eugene-ruby/xencryptor/xsecrets"
)

// MasterEncryptionKey is the master secret key injected at build time via -ldflags.
// It is used for decrypting sensitive data like private RSA keys.
var MasterEncryptionKey string

func MasterKeyBytes() []byte {
	return []byte(MasterEncryptionKey)
}

// Config holds all configuration for the application.
type Config struct {
	AppPort     string
	WebhookPath string
	MasterKey   string
	RabbitMQ    RabbitMQConfig
	Encryption  EncryptionConfig
}

type RabbitMQConfig struct {
	URL string
}

type EncryptionConfig struct {
	SecretSaltStr           string
	SecretSalt              []byte
	MasterKeyBytes          []byte
	PayloadEncryptionKeyStr string
	CasterPublicRSAKeyStr   string
	PayloadEncryptionKey    []byte
	CasterPublicRSAKey      *rsa.PublicKey
}

type defaultENV struct {
	appPort string
}

// LoadConfig reads environment variables and returns a Config instance.
func LoadConfig() (*Config, error) {
	defaultValues := &defaultENV{
		appPort: "8080",
	}

	cfg := &Config{
		AppPort:     os.Getenv("APP_PORT"),
		WebhookPath: os.Getenv("WEB_HOOK_PATH"),
		RabbitMQ: RabbitMQConfig{
			URL: os.Getenv("RABBITMQ_URL"),
		},
		Encryption: EncryptionConfig{
			SecretSaltStr:           os.Getenv("SECRET_SALT"),
			PayloadEncryptionKeyStr: os.Getenv("PAYLOAD_ENCRYPTION_KEY"),
			CasterPublicRSAKeyStr:   os.Getenv("CASTER_PUBLIC_KEY_RAW_BASE64"),
		},
	}

	if cfg.WebhookPath == "" {
		return nil, fmt.Errorf("WEB_HOOK_PATH environment variable must be set")
	}
	if cfg.RabbitMQ.URL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL environment variable must be set")
	}
	if cfg.Encryption.SecretSaltStr == "" {
		return nil, fmt.Errorf("SECRET_SALT environment variable must be set")
	}
	if cfg.Encryption.PayloadEncryptionKeyStr == "" {
		return nil, fmt.Errorf("PAYLOAD_ENCRYPTION_KEY environment variable must be set")
	}
	if cfg.Encryption.CasterPublicRSAKeyStr == "" {
		return nil, fmt.Errorf("CASTER_PUBLIC_KEY_RAW_BASE64 environment variable must be set")
	}
	if cfg.AppPort == "" {
		cfg.AppPort = defaultValues.appPort
	}
	if MasterEncryptionKey == "" {
		return nil, fmt.Errorf("MasterEncryptionKey must be injected at build time with -ldflags")
	}
	cfg.Encryption.MasterKeyBytes = MasterKeyBytes()

	if err := decryptKeys(&cfg.Encryption); err != nil {
		return nil, err
	}
	if err := loadPublicKey(&cfg.Encryption); err != nil {
		return nil, err
	}

	return cfg, nil
}

func decryptKeys(enc *EncryptionConfig) error {
	keyPayload := xsecrets.DeriveKey(MasterKeyBytes(), "payload")
	decryptedPayloadKey, err := xsecrets.DecryptBase64WithKey(enc.PayloadEncryptionKeyStr, keyPayload)

	if err != nil {
		return fmt.Errorf("failed to decrypt PAYLOAD_ENCRYPTION_KEY: %w", err)
	}
	enc.PayloadEncryptionKey = decryptedPayloadKey

	keySalt := xsecrets.DeriveKey(MasterKeyBytes(), "salt")
	decryptedSecretSaltKey, err := xsecrets.DecryptBase64WithKey(enc.SecretSaltStr, keySalt)
	if err != nil {
		return fmt.Errorf("failed to decrypt SECRET_SALT: %w", err)
	}
	enc.SecretSalt = decryptedSecretSaltKey

	return nil
}

func loadPublicKey(enc *EncryptionConfig) error {
	encRSABase64 := enc.CasterPublicRSAKeyStr
	derBytes, err := base64.RawStdEncoding.DecodeString(encRSABase64)
	if err != nil {
		return fmt.Errorf("failed to decode CASTER_PUBLIC_KEY_RAW_BASE64: %w", err)
	}
	pubKey, err := x509.ParsePKIXPublicKey(derBytes)
	if err != nil {
		return fmt.Errorf("failed to parse PublicKey: %w", err)
	}
	publicKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("failed to decrypt PublicKey")
	}
	enc.CasterPublicRSAKey = publicKey
	return nil
}
