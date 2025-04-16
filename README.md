# murmapp.hook

**murmapp.hook** is a minimal Go service designed to securely receive Telegram webhook events, verify their authenticity, and forward them as raw Protobuf messages to RabbitMQ for further processing.

---

## 📌 Features

- Secure webhook validation using a SHA1 hash of `token + SECRET_SALT`
- Raw request body forwarding (no Telegram JSON parsing)
- Protobuf serialization
- RabbitMQ integration with automatic retry/reconnect
- Clean HTTP API using `chi` router
- Docker & Docker Compose support
- GitHub Actions CI with health check

---

## ⚙️ API Specification

### Endpoint

```
POST /api/webhook/{webhook_id}
```

- `webhook_id`: extracted from the URL
- Telegram sends a special header:
  ```
  X-Telegram-Bot-Api-Secret-Token: your_secret_token
  ```
- The server computes:

  ```
  sha1(secret_token + SECRET_SALT) == webhook_id
  ```

  If the check fails, the request is rejected with `403 Forbidden`.

---

## 📡 Message Queues

The `murmapp` system uses **RabbitMQ topic exchanges** to route messages between services like `hook`, `core`, and `caster`.

---

### 🟦 Exchange: `murmapp.registrations`

Handles bot registration flows between `core` and `hook`.

| routing_key  | From   | To     | Description                              |
|--------------|--------|--------|------------------------------------------|
| `registration` | core   | hook   | Command to register bot webhook          |
| `registered`   | hook   | core   | Acknowledgment after successful setup    |

📸 Registration flow:  
![registrations](docs/murmapp_registrations.png)

---

### 🟩 Exchange: `murmapp.messages.in`

Handles all incoming messages from Telegram via webhook.

| routing_key    | From  | To     | Description                     |
|----------------|-------|--------|---------------------------------|
| `telegram.raw` | hook  | core   | Raw Telegram webhook payload    |

📸 Telegram message flow:  
![messages in](docs/murmapp_messages_in.png)


---

### Payload

Raw Telegram webhook JSON is **not parsed**, only wrapped into a protobuf:

```proto
message TelegramWebhookPayload {
  string webhook_id = 1;
  bytes raw_body = 2;
  int64 received_at_unix = 3;
}
```

---

### Registration Flow Payloads

Bot registration messages exchanged via `murmapp.registrations` use the following Protobuf definitions:

```proto
message RegisterWebhookRequest {
  string bot_id = 1;
  string api_key_bot = 2;
}

message RegisterWebhookResponse {
  string bot_id = 1;
  string webhook_id = 2;
}
```

---

## 🚀 Getting Started

### 1. Clone and build

```bash
git clone https://github.com/eugene-ruby/murmapp.hook.git
cd murmapp.hook
go mod tidy
go build -o app ./cmd/main.go
```

### 2. Run locally with Docker

Make sure [Docker](https://docs.docker.com/get-docker/) is installed.

```bash
docker-compose up --build
```

This will launch:

- the webhook service at `http://localhost:8080`
- RabbitMQ with management UI at `http://localhost:15672` (guest/guest)

### 3. Test the webhook

```bash
curl -X POST "http://localhost:8080/api/webhook/<webhook_id>" \
  -H "Content-Type: application/json" \
  -H "X-Telegram-Bot-Api-Secret-Token: <secret_token>" \
  -d '{"update_id": 12345, "message": {"text": "Hello"}}'
```

Replace:

- `<secret_token>` with the secret used when registering the webhook
- `<webhook_id>` with the SHA1 hash of `<secret_token><SECRET_SALT>`

You can compute it like this:

```bash
echo -n "<secret_token><SECRET_SALT>" | sha1sum
```

---

## ✅ Health Check

Simple GET endpoint:

```
GET /healthz
```

Returns `200 OK` if the service is running.


---

## 👨‍💻 For Developers

### 🧱 Project Structure

- `Dockerfile` — minimal production image (no `protoc`)
- `Dockerfile.dev` — development image with `protoc` and Go tooling
- `.github/workflows/deploy.yml` — deployment workflow triggered on `v*` tags

---

### 🔐 Required GitHub Secrets

To enable CI deployment, set the following secrets in **GitHub → Settings → Secrets and variables → Actions**:

| Secret Name       | Description                                |
|-------------------|--------------------------------------------|
| `SSH_HOST`        | IP or hostname of your production server   |
| `SSH_USER`        | SSH user used for deployment               |
| `SSH_KEY`         | SSH private RSA key (single-line format)   |
| `SECRET_SALT`     | Secret salt used for webhook verification  |
| `RABBITMQ_URL`    | RabbitMQ connection string (AMQP URL)      |
| `WEB_HOOK_HOST`   | The domain with ssl to which Telegram will send webhooks        |


Example:

```env
WEB_HOOK_HOST=https://domain.com
SECRET_SALT=supersecret
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
```

---


### 🚀 Trigger Deployment

Create a Git tag and push it — this triggers the GitHub Actions workflow:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The server will automatically:

- pull the latest image from Harbor
- recreate the container with updated env vars and config

---

## License

MIT
