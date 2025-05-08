package webhook

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/eugene-ruby/xconnect/rabbitmq"
	"github.com/eugene-ruby/xencryptor/xsecrets"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/proto"
	"murmapp.hook/internal/config"
	hookpb "murmapp.hook/proto"
)

type OutboundHandler struct {
	Channel rabbitmq.Channel
	Config  config.Config
}

func HandleWebhook(w http.ResponseWriter, r *http.Request, h *OutboundHandler) {
	webhookID := chi.URLParam(r, "webhook_id")
	ip := r.RemoteAddr

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		log.Printf("[hook] ❌ failed to read request body from %s: %v", ip, err)
		return
	}
	defer r.Body.Close()

	if !isAuthorizedWebhook(r, webhookID, h) {
		http.Error(w, "forbidden", http.StatusForbidden)
		log.Printf("[hook] 🚨 token mismatch for IP=%s, rejecting request", ip)
		return
	}

	result, err := FilterPayload(raw, string(h.Config.Encryption.SecretSalt))
	log.Printf("[hook] 🔐 %d sensitive value(s) matched and processed in payload", result.Matched)
	if err != nil {
		log.Printf("[hook] ❌ dropped payload from %s: %s", ip, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := publishWebhookPayload(webhookID, result.RedactedJSON, h); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	publishTelegramIDs(result, h)

	log.Printf("[hook] ✅ accepted webhook from %s, forwarded to MQ", ip)
	w.WriteHeader(http.StatusOK)
}

func isAuthorizedWebhook(r *http.Request, webhookID string, h *OutboundHandler) bool {
	token := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	expectedID := ComputeWebhookID(token, string(h.Config.Encryption.SecretSalt))
	log.Printf("[hook] 📩 incoming from IP=%s | webhook_id=%s | token_len=%d | valid=%v", r.RemoteAddr, webhookID, len(token), expectedID == webhookID)
	return expectedID == webhookID
}

func publishWebhookPayload(webhookID string, redacted []byte, h *OutboundHandler) error {
	encrypted, err := xsecrets.EncryptBytesWithKey(redacted, h.Config.Encryption.PayloadEncryptionKey)
	if err != nil {
		log.Printf("[hook] ❌ encryption failed: %v", err)
		return err
	}

	payload := &hookpb.TelegramWebhookPayload{
		WebhookId:        webhookID,
		EncryptedPayload: encrypted,
		ReceivedAtUnix:   time.Now().Unix(),
	}

	msg, err := proto.Marshal(payload)
	if err != nil {
		log.Printf("[hook] ❌ failed to marshal payload: %v", err)
		return err
	}

	if err := h.Channel.Publish("murmapp", "telegram.messages.in", msg); err != nil {
		log.Printf("[hook] ❌ failed to publish to MQ: %v", err)
		return err
	}

	return nil
}

func publishTelegramIDs(result FilterResult, h *OutboundHandler) {
	for _, id := range result.TelegramIDs {
		encryptedID, err := xsecrets.RSAEncryptBytes(h.Config.Encryption.CasterPublicRSAKey, []byte(id.OpenTelegramID))
		if err != nil {
			log.Printf("[hook] ❌ failed to encrypt telegram_id %s: %v", id.OpenTelegramID, err)
			continue
		}

		msg := &hookpb.EncryptedTelegramID{
			TelegramXid: id.TelegramXId,
			EncryptedId: encryptedID,
		}

		data, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("[hook] ❌ failed to marshal EncryptedTelegramID: %v", err)
			continue
		}

		if err := h.Channel.Publish("murmapp", "telegram.encrypted.id", data); err != nil {
			log.Printf("[hook] ❌ failed to publish encrypted telegram_id to MQ: %v", err)
			continue
		}
	}
}

func ComputeWebhookID(secretToken, secretSalt string) string {
	h := sha1.New()
	h.Write([]byte(secretToken + secretSalt))
	return hex.EncodeToString(h.Sum(nil))
}
