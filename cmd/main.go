package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"murmapp.hook/internal"
)

func main() {
	r := chi.NewRouter()

	mq, err := internal.InitMQ(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("RabbitMQ error: %v", err)
	}
	defer mq.Close()
	
	if err := internal.InitExchanges(mq.GetChannel()); err != nil {
		log.Fatalf("Exchange init failed: %v", err)
	}

	r.Post("/api/webhook/{webhook_id}", func(w http.ResponseWriter, r *http.Request) {
		internal.HandleWebhook(w, r, mq)
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Start registration consumer
	go func() {
		if err := internal.StartRegistrationConsumer(mq.GetChannel()); err != nil {
			log.Fatalf("failed to start registration consumer: %v", err)
		}
	}()

	addr := ":8080"
	log.Printf("Starting server on %s...", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
