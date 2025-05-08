package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"fmt"

	"github.com/eugene-ruby/xconnect/rabbitmq"
	"github.com/go-chi/chi/v5"
	"murmapp.hook/internal/config"
	"murmapp.hook/internal/webhook"
)

type OutboundHandler struct {
	Channel rabbitmq.Channel
	Config  config.Config
}

func StartHookServer(ctx context.Context, h *OutboundHandler) error {
	addr := ":" + h.Config.AppPort

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	path := fmt.Sprintf("%s/{webhook_id}", h.Config.WebhookPath)
	r.Post(path, func(w http.ResponseWriter, r *http.Request) {
		webhook.HandleWebhook(w, r, &webhook.OutboundHandler{ Channel: h.Channel, Config: h.Config })
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Channel for receiving system signals
	idleConnsClosed := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		select {
		case <-sigChan:
		case <-ctx.Done():
		}

		log.Println("🌐 shutting down hook server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("🌐 Starting hook server on %s...", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed
	return nil
}
