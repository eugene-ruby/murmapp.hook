# 🔎 murmapp.hook

```bash
╭────────────────────────────────────────────────╮
│      murmapp.hook: secure Telegram entrypoint   │
│      for redacting, hashing, and publishing     │
╰────────────────────────────────────────────────╯
       ↳ token-verified, crypto-clean, stateless
```

[![Go Report Card](https://goreportcard.com/badge/github.com/eugene-ruby/murmapp.hook)](https://goreportcard.com/report/github.com/eugene-ruby/murmapp.hook)
[![Build Status](https://github.com/eugene-ruby/murmapp.hook/actions/workflows/ci.yml/badge.svg)](https://github.com/eugene-ruby/murmapp.hook/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**murmapp.hook** is a secure and minimal webhook receiver for the Murmapp system.
It listens for incoming Telegram webhook requests, verifies the signature, redacts and encrypts sensitive data, and pushes encrypted payloads and telegram IDs into RabbitMQ for downstream processing.

---

## ✨ Features

* Token-based signature validation per webhook
* JSON redaction engine with path-based rules (`privacy_keys.conf`)
* Automatic XID generation for `telegram_id` using salted SHA256
* RSA encryption of original Telegram IDs
* Encrypted payload forwarding via RabbitMQ
* Clean separation of `config`, `run`, `webhook`, `server` logic
* Graceful shutdown via OS signal handling
* Fully covered with unit tests and mocks

## 🚀 Quick Start

A template file env_test_example is provided for development and testing purposes. It included in Makefile via:

```bash
include .env_test
export
```

To use it, simply rename the template:

```bash
mv env_test_example .env_test
```

and adjust the values to match your environment.

```bash
make build
```

---

## ⚙️ Environment Variables

| Variable                 | Required | Description                                 |
| ------------------------ | -------- | ------------------------------------------- |
| `APP_PORT`               | No       | Port to bind HTTP server (default `8080`)   |
| `WEB_HOOK_PATH`          | Yes      | Route prefix (e.g. `api/webhook`)           |
| `RABBITMQ_URL`           | Yes      | AMQP URI to connect to RabbitMQ             |
| `SECRET_SALT`            | Yes      | Encrypted base64 of SHA salt for ID hashing |
| `PAYLOAD_ENCRYPTION_KEY` | Yes      | Encrypted base64 AES-256 key for payloads   |
| `PUBLIC_KEY_RAW_BASE64`  | Yes      | Base64 encoded raw RSA public key (X.509)   |
| `MASTER_ENCRYPTION_KEY`  | Yes      | Supplied via `-ldflags` at build time       |

---

## 💡 How it works

```bash
          ┌────────────┐
          │  Telegram  │
          │   Server   │
          └─────┬──────┘
                │
                ▼
      ┌────────────────────┐
      │  POST /webhook     │
      │  with JSON payload │
      └────────┬───────────┘
               ▼
     ┌────────────────────────┐
     │ isAuthorizedWebhook()  │◄── X-Telegram-Bot-Api-Secret-Token
     └────────┬───────────────┘
              ▼
     ┌────────────────────────┐
     │ FilterPayload()        │
     │ redact + extract ID(s) │
     └────────┬───────────────┘
              ▼
     ┌────────────────────────────────────┐
     │ Encrypt payload with AES-256       │
     │ Send to `telegram.messages.in`     │
     └────────┬───────────────────────────┘
              ▼
     ┌────────────────────────────────────┐
     │ For each ID:                       │
     │   - SHA256(id + salt) → xid        │
     │   - RSA encrypt original ID        │
     │   - Send to `telegram.encrypted.id`│
     └────────────────────────────────────┘

```

1. Incoming request is validated against webhook token
2. JSON is scanned with rules like `message.from.id`, `message.chat.id`
3. Any field named `id` is:

   * converted to `telegram_xid` via `SHA256(id + salt)`
   * collected as `{telegram_id, telegram_xid}`
4. Payload is encrypted with AES-256 and sent to `telegram.messages.in`
5. Each `telegram_id` is encrypted with RSA and sent to `telegram.encrypted.id`

---

## 📅 Example message flows

| Source             | Queue                   | Message                         |
| ------------------ | ----------------------- | ------------------------------- |
| Telegram HTTP POST |                         | `raw json`                      |
| hook               | `telegram.messages.in`  | `TelegramWebhookPayload`        |
| hook               | `telegram.encrypted.id` | `EncryptedTelegramID`           |
| caster             | uses both               | decrypts and processes outbound |

---

## ⚖️ Security

* Raw `telegram_id` never written to disk or logs
* Master encryption key is passed at build only (via `-ldflags`)
* All AES and RSA crypto uses xencryptor wrapper (AES-GCM, 2048-bit RSA)
* Salted hash used as XID avoids linking across payloads

---

## 🛠️ Components

* `config/`    — env + crypto key loader
* `run.go`     — app init, signal handler, shutdown
* `webhook/`   — HTTP handler, filter, encrypt, publish
* `server/`    — chi router, mount endpoints

---

## ✅ Testing

```bash
rename env_test_example .env_test
make test
```

---

## ™ License

This project is licensed under the [MIT License](/LICENSE).
