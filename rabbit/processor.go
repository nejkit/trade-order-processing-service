package rabbit

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
)

type ParserFunc[T any] func([]byte) (*T, error)
type HandlerFunc[T any] func(context.Context, *T)

type Processor[T any] struct {
	parser  ParserFunc[T]
	handler HandlerFunc[T]
}

func NewProcessor[T any](parser ParserFunc[T], handler HandlerFunc[T]) Processor[T] {
	return Processor[T]{parser: parser, handler: handler}
}

func (p *Processor[T]) processMessage(ctx context.Context, msg amqp091.Delivery) {
	body, err := p.parser(msg.Body)

	if err != nil {
		msg.Nack(false, false)
		return
	}

	msg.Ack(false)
	p.handler(ctx, body)
}
