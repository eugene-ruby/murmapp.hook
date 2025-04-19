package internal

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/proto"
	hookpb "murmapp.hook/proto"
)

func HandleWebhook(w http.ResponseWriter, r *http.Request, mq Publisher) {
	webhookID := chi.URLParam(r, "webhook_id")
	token := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	ip := r.RemoteAddr

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		log.Printf("[hook] ❌ failed to read request body from %s: %v", ip, err)
		return
	}
	defer r.Body.Close()

	expectedID := ComputeWebhookID(token, os.Getenv("SECRET_SALT"))
	valid := (expectedID == webhookID)

	log.Printf("[hook] 📩 incoming from IP=%s | webhook_id=%s | token_len=%d | valid=%v", ip, webhookID, len(token), valid)

	if !valid {
		http.Error(w, "forbidden", http.StatusForbidden)
		log.Printf("[hook] 🚨 token mismatch for IP=%s, rejecting request", ip)
		return
	}

	payload := &hookpb.TelegramWebhookPayload{
		WebhookId:      webhookID,
		RawBody:        body,
		ReceivedAtUnix: time.Now().Unix(),
	}

	msg, err := proto.Marshal(payload)
	if err != nil {
		log.Printf("[hook] ❌ failed to marshal payload: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := mq.Publish("murmapp", "telegram.messages.in", msg); err != nil {
		log.Printf("[hook] ❌ failed to publish to MQ: %v", err)
		http.Error(w, "mq error", http.StatusInternalServerError)
		return
	}

	log.Printf("[hook] ✅ accepted webhook from %s, forwarded to MQ", ip)
	w.WriteHeader(http.StatusOK)
}
