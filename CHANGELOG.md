# Changelog

All notable changes to this project will be documented in this file.

---

## [v0.1.8] - 2025-04-20

### Added
- AES-GCM encryption of entire redacted Telegram payload before sending to RabbitMQ (`EncryptedPayload` field in protobuf)
- Support for encrypted `api_key_bot` in `RegisterWebhookRequest` (as `bytes`)
- Decryption utilities in `encrypt.go` for payload and ID fields
- Full test coverage for encryption/decryption and webhook flow
- End-to-end test for `HandleWebhook` validating encrypted payload

### Changed
- Replaced `raw_body` with `encrypted_payload` in `TelegramWebhookPayload` proto
- `filter.go` no longer encrypts message text fields (now handled as whole payload)
- `handler.go` now encrypts payload before producing to `telegram.messages.in`

### Removed
- Per-field text encryption logic

---

## [v0.1.7] - pre-release

Initial implementation of:
- Webhook handler with token validation
- Field-level redaction and telegram ID encryption
- Protobuf serialization and RabbitMQ publishing
