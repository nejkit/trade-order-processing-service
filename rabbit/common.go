package rabbit

import (
	"time"
	"trade-order-processing-service/staticerr"

	"github.com/rabbitmq/amqp091-go"
)

func GetRabbitConnection(connectionString string) (*amqp091.Connection, error) {
	timeout := time.After(time.Minute * 5)
	for {
		select {
		case <-timeout:
			return nil, staticerr.ErrorRabbitConnectionFail
		default:
			connect, err := amqp091.Dial(connectionString)

			if err != nil {
				time.Sleep(time.Millisecond * 100)
				continue
			}

			return connect, nil
		}
	}
}
