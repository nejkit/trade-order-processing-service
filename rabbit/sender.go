package rabbit

import (
	"context"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Sender struct {
	channel *amqp091.Channel
}

func NewSender(ctx context.Context, channel *amqp091.Channel) Sender {
	s := Sender{channel: channel}
	go s.handleGraceful(ctx)
	return s
}

func (s *Sender) SendMessage(ctx context.Context, message protoreflect.ProtoMessage, exchange, rk string) error {
	bytes, err := proto.Marshal(message)

	if err != nil {
		return err
	}

	err = s.channel.PublishWithContext(ctx, exchange, rk, false, false, amqp091.Publishing{
		ContentType: "text/plain",
		Body:        bytes,
	})

	if err != nil {
		return err
	}
	return nil
}

func (s *Sender) handleGraceful(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.channel.Close()
			return
		default:
			time.Sleep(time.Millisecond * 100)
		}

	}
}
