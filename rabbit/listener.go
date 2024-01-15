package rabbit

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
)

type channelConfig struct {
	channel         *amqp091.Channel
	deliveryChannel <-chan amqp091.Delivery
}

type Listener[T any] struct {
	config    channelConfig
	processor Processor[T]
}

func NewListener[T any](ctx context.Context, channel *amqp091.Channel, qName string, processor Processor[T]) (*Listener[T], error) {
	deliveryChannel, err := channel.ConsumeWithContext(ctx, qName, "", false, false, false, false, nil)

	if err != nil {
		return nil, err
	}

	return &Listener[T]{
		config: channelConfig{
			channel:         channel,
			deliveryChannel: deliveryChannel,
		},
		processor: processor,
	}, nil
}

func (l *Listener[T]) Run(ctx context.Context) {
	for {
		select {
		case msg, ok := <-l.config.deliveryChannel:
			if !ok {
				continue
			}
			go l.processor.processMessage(ctx, msg)
		case <-ctx.Done():
			l.config.channel.Close()
			return
		}
	}
}
