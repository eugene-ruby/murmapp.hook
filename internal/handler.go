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
		return
	}
	defer r.Body.Close()

	expectedID := ComputeWebhookID(token, os.Getenv("SECRET_SALT"))

	log.Printf("[hook] IP=%s WebhookID=%s TokenLen=%d -> Valid=%v", ip, webhookID, len(token), expectedID == webhookID)

	if expectedID != webhookID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	payload := &hookpb.TelegramWebhookPayload{
		WebhookId:      webhookID,
		RawBody:        body,
		ReceivedAtUnix: time.Now().Unix(),
	}

	msg, err := proto.Marshal(payload)
	if err != nil {
		log.Printf("marshal error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := mq.Publish("murmapp.messages.in", "telegram.raw", msg); err != nil {
		log.Printf("publish error: %v", err)
		http.Error(w, "mq error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
