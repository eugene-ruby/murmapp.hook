package internal

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/eugene-ruby/xconnect/rabbitmq"
	"github.com/streadway/amqp"
	"murmapp.hook/internal/config"
	"murmapp.hook/internal/rabbitmqinit"
	"murmapp.hook/internal/server"
	"murmapp.hook/internal/webhook"
)

// AppRabbitMQ manages RabbitMQ connection and channel lifecycle.
type AppRabbitMQ struct {
	conn    *amqp.Connection
	rawCh   *amqp.Channel
	channel rabbitmq.Channel
}

// Close gracefully shuts down the AMQP connection and channel.
func (a *AppRabbitMQ) Close() {
	if a.rawCh != nil {
		_ = a.rawCh.Close()
	}
	if a.conn != nil {
		_ = a.conn.Close()
	}
}

// Run initializes configuration, connects to RabbitMQ,
// loads privacy keys, starts the HTTP server, and blocks until shutdown.
func Run() error {
	conf, err := config.LoadConfig()
	if err != nil {
		return err
	}

	rmq, ch, err := initRabbitMQ(*conf)
	if err != nil {
		return err
	}
	defer Shutdown(rmq)

	if err := rabbitmqinit.DeclareExchanges(ch); err != nil {
		return err
	}

	if err := webhook.LoadPrivacyKeys(); err != nil {
		return err // changed from fatal to return for testability
	}

	// Listen for OS signals to handle graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start the webhook HTTP server in a background goroutine
	go func() {
		h := &server.OutboundHandler{
			Channel: ch,
			Config:  *conf,
		}
		if err := server.StartHookServer(ctx, h); err != nil {
			log.Printf("Hook server error: %v", err)
			cancel()
		}
	}()

	// Wait until shutdown signal is received
	<-ctx.Done()
	log.Println("✅ app shut down cleanly")
	return nil
}

// initRabbitMQ establishes a RabbitMQ connection and returns both
// the raw AMQP channel and a wrapped xconnect-compatible channel.
func initRabbitMQ(cfg config.Config) (*AppRabbitMQ, rabbitmq.Channel, error) {
	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	wrapped := rabbitmq.WrapAMQPChannel(ch)
	return &AppRabbitMQ{
		conn:    conn,
		rawCh:   ch,
		channel: wrapped,
	}, wrapped, nil
}

// Shutdown safely closes RabbitMQ resources.
func Shutdown(rmq *AppRabbitMQ) {
	rmq.Close()
}
