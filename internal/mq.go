package internal

import (
    "time"

    "github.com/streadway/amqp"
)

type MQPublisher struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

type Publisher interface {
	Publish(exchange, routingKey string, body []byte) error
}

func (mq *MQPublisher) GetChannel() *amqp.Channel {
    return mq.channel
}

func InitMQ(url string) (*MQPublisher, error) {
    var conn *amqp.Connection
    var err error

    for retries := 0; retries < 30; retries++ {
        conn, err = amqp.Dial(url)
        if err == nil {
            break
        }
        time.Sleep(5 * time.Second)
    }
    if err != nil {
        return nil, err
    }

    ch, err := conn.Channel()
    if err != nil {
        return nil, err
    }

    return &MQPublisher{
        conn:    conn,
        channel: ch,
    }, nil
}

func (p *MQPublisher) Publish(exchange, routingKey string, body []byte) error {
    return p.channel.Publish(
        exchange,
        routingKey,
        false, false,
        amqp.Publishing{
            ContentType: "application/octet-stream",
            Body:        body,
        })
}

func (p *MQPublisher) Close() {
    if p.channel != nil {
        p.channel.Close()
    }
    if p.conn != nil {
        p.conn.Close()
    }
}
