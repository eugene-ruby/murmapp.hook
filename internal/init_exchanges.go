package internal

import (
	"github.com/streadway/amqp"
	"log"
)

// InitExchanges declares all topic exchanges used by the system
func InitExchanges(ch *amqp.Channel) error {
	exchanges := []string{
		"murmapp.messages.in",
		"murmapp.messages.out",
		"murmapp.registrations",
	}

	for _, ex := range exchanges {
		err := ch.ExchangeDeclare(
			ex,
			"topic", // exchange type
			true,    // durable
			false,   // auto-deleted
			false,   // internal
			false,   // no-wait
			nil,     // arguments
		)
		if err != nil {
			log.Printf("failed to declare exchange %s: %v", ex, err)
			return err
		}
		log.Printf("exchange declared: %s", ex)
	}

	return nil
}
