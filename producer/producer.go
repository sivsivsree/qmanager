package producer

import (
	"context"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sivsivsree/qmanger"
)

var ErrPublishing = errors.New("publishing failed")

type Publisher struct {
	conn qmanger.Connector
}

func New(connection qmanger.Connector) *Publisher {
	return &Publisher{
		connection,
	}
}

func (p *Publisher) Publish(ctx context.Context, queueName string, data []byte) error {

	ch, err := p.conn.GetChannel()
	if err != nil {
		return qmanger.ErrChannel
	}
	defer func(ch *amqp.Channel) {
		_ = ch.Close()
	}(ch)

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return ErrPublishing
	}

	err = ch.PublishWithContext(
		ctx,
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)

	if err != nil {
		return ErrPublishing
	}

	return nil

}
