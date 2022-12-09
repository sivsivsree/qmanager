package consumer

import (
	"context"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sivsivsree/qmanager"
	"strconv"
	"sync/atomic"
	"time"
)

var consumerSeq uint64

type ConsumeErr struct {
	Data []byte
	Err  error
}

type Consumer struct {
	conn     qmanager.Connector
	consumer string
	ticker   *time.Ticker
}

func New(connection qmanager.Connector, consumer string) *Consumer {
	return &Consumer{
		conn:     connection,
		consumer: consumer,
		ticker:   time.NewTicker(5 * time.Second),
	}
}

func (c Consumer) Consume(ctx context.Context, queue string, processIncoming func(data []byte) error) error {

connect:
	ch, err := c.conn.GetChannel()
	if err != nil {
		time.Sleep(10 * time.Second)
		goto connect
	}
	defer func(ch *amqp.Channel) {
		_ = ch.Close()
	}(ch)

	if c.consumer == "" {
		c.consumer = "qc"
	}
	tag := strconv.FormatUint(atomic.AddUint64(&consumerSeq, 1), 10)
	unique := fmt.Sprintf("%s-%s-%d", c.consumer, tag, time.Now().Nanosecond())

	delivery, err := ch.Consume(
		queue,
		unique, // Consumer
		false,  // Auto-Ack
		false,  // Exclusive
		false,  // No-local
		false,  // No-Wait
		nil,    // Args
	)

	for {
		select {
		case d := <-delivery:
			if d.Body != nil {
				err := processIncoming(d.Body)
				if err == nil {
					_ = d.Ack(false)
				}
			}
		case <-c.ticker.C:
			if !c.conn.IsConnected() {
				goto connect
			}

		case <-ctx.Done():
			fmt.Println("worker queue closed for ", c.consumer)
			return nil
		}
	}
}

func (c Consumer) ConsumeWithError(ctx context.Context, queue string, errCh chan<- ConsumeErr, processIncoming func(data []byte) error) error {

connect:
	ch, err := c.conn.GetChannel()
	if err != nil {
		time.Sleep(10 * time.Second)
		goto connect
	}
	defer func(ch *amqp.Channel) {
		_ = ch.Close()
	}(ch)

	if c.consumer == "" {
		c.consumer = "qc"
	}
	tag := strconv.FormatUint(atomic.AddUint64(&consumerSeq, 1), 10)
	unique := fmt.Sprintf("%s-%s-%d", c.consumer, tag, time.Now().Nanosecond())

	delivery, err := ch.Consume(
		queue,
		unique, // Consumer
		false,  // Auto-Ack
		false,  // Exclusive
		false,  // No-local
		false,  // No-Wait
		nil,    // Args
	)

	for {
		select {
		case d := <-delivery:
			if d.Body != nil {
				err := processIncoming(d.Body)
				if err == nil {
					go processError(errCh, err, d.Body)
					_ = d.Ack(false)
				}
			}
		case <-c.ticker.C:
			if !c.conn.IsConnected() {
				goto connect
			}

		case <-ctx.Done():
			fmt.Println("worker queue closed for ", c.consumer)
			return nil
		}
	}
}

func processError(ch chan<- ConsumeErr, err error, body []byte) {
	ch <- ConsumeErr{
		Data: body,
		Err:  err,
	}
}
