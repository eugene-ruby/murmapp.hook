package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"

	"google.golang.org/protobuf/proto"
	"github.com/streadway/amqp"
	hookpb "murmapp.hook/proto"
)

func StartRegistrationConsumer(ch *amqp.Channel) error {
	q, err := ch.QueueDeclare("murmapp.hook.webhook.registration", true, false, false, false, nil)
	if err != nil {
		return err
	}

	if err := ch.QueueBind(q.Name, "webhook.registration", "murmapp", false, nil); err != nil {
		return err
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var req hookpb.RegisterWebhookRequest
			if err := proto.Unmarshal(d.Body, &req); err != nil {
				log.Printf("[registrations] ❌ failed to unmarshal protobuf: %v", err)
				continue
			}

			log.Printf("[registrations] 📅 received registration request for bot_id: %s", req.BotId)

			secretToken := generateSecretToken()
			webhookID := ComputeWebhookID(secretToken, os.Getenv("SECRET_SALT"))
			webhookURL := fmt.Sprintf("%s/api/webhook/%s", os.Getenv("WEB_HOOK_HOST"), webhookID)

			decrypt_api_key, err := DecryptWithKey(req.ApiKeyBot, SecretEncryptionKey)
			if err != nil {
				log.Printf("[hook] ❌ failed to decrypt api key: %v", err)
				return
			}
			
			if err := registerTelegramWebhook(decrypt_api_key, webhookURL, secretToken); err != nil {
				log.Printf("[registrations] ❌ webhook registration failed: %v", err)
				continue
			}

			resp := &hookpb.RegisterWebhookResponse{
				BotId:     req.BotId,
				WebhookId: webhookID,
			}

			body, err := proto.Marshal(resp)
			if err != nil {
				log.Printf("[registrations] ❌ failed to marshal response: %v", err)
				continue
			}

			err = ch.Publish("murmapp", "webhook.registered", false, false, amqp.Publishing{
				ContentType: "application/protobuf",
				Body:        body,
			})

			if err != nil {
				log.Printf("[registrations] ❌ publish error: %v", err)
			} else {
				log.Printf("[registrations] ✅ registered webhook_id: %s for bot_id: %s", webhookID, req.BotId)
			}
		}
	}()

	log.Println("[registrations] 🚀 consumer started and listening...")
	return nil
}

func generateSecretToken() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func registerTelegramWebhook(apiKey, url, secretToken string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", apiKey)

	payload := map[string]string{
		"url":          url,
		"secret_token": secretToken,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}
