# murmapp.hook

**murmapp.hook** is a minimal Go service designed to securely receive Telegram webhook events, verify their authenticity, redact and encrypt sensitive data, and forward them as encrypted Protobuf messages to RabbitMQ for further processing.

---

## 📌 Features

- Secure webhook validation using SHA1(secret_token + SECRET_SALT)
- AES-GCM encryption of `telegram_id` using `TELEGRAM_ID_ENCRYPTION_KEY`
- AES-GCM encryption of the **entire redacted payload** using `ENCRYPTION_KEY` before putting into RabbitMQ
- Redaction of fields like `username`, `first_name`, `last_name` to `"[redacted]"`
- Declarative redaction paths via embedded `privacy_keys.yml`
- Only passes through payloads with at least one matched redaction key
- Fully self-contained Go binary with embedded config
- Clean HTTP API using `chi` router
- Docker-compatible and GitHub Actions ready

---

## 🔐 Privacy & Security

Before forwarding any Telegram payload:

- The `hook` service checks the JSON payload against a list of `privacy_keys` embedded at build time
- Fields like `message.from.id` are AES-GCM encrypted using `TELEGRAM_ID_ENCRYPTION_KEY`
- Identity-related fields like `username`, `first_name`, and `last_name` are replaced with `"[redacted]"`
- After redaction, the entire cleaned payload is AES-GCM encrypted with a **separate** `ENCRYPTION_KEY`
- The encrypted payload is stored in `TelegramWebhookPayload.encrypted_payload` as raw `bytes`
- If no redaction keys matched — the payload is silently dropped

> ⚠️ **Disclaimer**: Telegram does not provide real anonymity. Bot owners can technically access original `telegram_id` and messages at the API level. We encrypt to reduce exposure within the system, but not eliminate it.

---

### 🔐 Example Output (after redaction + encryption):

```proto
message TelegramWebhookPayload {
  string webhook_id = 1;
  bytes encrypted_payload = 2;
  int64 received_at_unix = 3;
}
```

> The `encrypted_payload` contains AES-GCM sealed bytes of cleaned JSON, never stored or routed in plaintext

---

## 🟦 Registration Flow Payloads

Bot registration messages exchanged via `murmapp` use the following Protobuf definitions:

```proto
message RegisterWebhookRequest {
  string bot_id = 1;
  bytes api_key_bot = 2; // encrypted with TELEGRAM_ID_ENCRYPTION_KEY
}

message RegisterWebhookResponse {
  string bot_id = 1;
  string webhook_id = 2;
}
```

The `api_key_bot` is encrypted at the sender side and decrypted by `hook` before calling the real Telegram API.

---

## 📡 Message Queues

The system uses a topic exchange named `murmapp`, with routing keys like:

| routing_key             | From  | To     | Description                              |
|-------------------------|--------|--------|------------------------------------------|
| `telegram.messages.in`  | hook   | core   | Fully redacted and encrypted Telegram message |
| `webhook.registration`  | core   | hook   | Command to register bot webhook          |
| `webhook.registered`    | hook   | core   | Acknowledgment after webhook setup       |

---

## 🔧 Development

- `internal/encrypt.go` — AES-GCM encryption utils
- `internal/filter.go` — field-level redaction and ID encryption
- `internal/handler.go` — HTTP endpoint, applies filtering, encryption, MQ publish
- `proto/telegram_webhook.proto` — contains `TelegramWebhookPayload` and registration messages
- `internal/registration_consumer.go` — decrypts bot API key and registers it

---

## 🚀 Ready to Deploy?

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will:
- Build the binary via GitHub Actions
- Embed `privacy_keys.yml`
- Deliver and restart the hook service on your server

---

## License

MIT