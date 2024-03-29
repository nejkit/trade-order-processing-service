package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"trade-order-processing-service/external/ops"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	ticketsListKey = "tickets:ops"
)

type TicketStorage struct {
	client *RedisClient
}

func NewTicketStorage(client *RedisClient) *TicketStorage {
	return &TicketStorage{client: client}
}

func (t *TicketStorage) AddNewTicket(ctx context.Context, operationType ops.OpsTicketOperation, ticketData protoreflect.ProtoMessage) error {
	ticketId := uuid.NewString()

	bytes, err := proto.Marshal(ticketData)

	if err != nil {
		return err
	}

	data, err := base64.StdEncoding.DecodeString(string(bytes))

	if err != nil {
		return err
	}

	ticketDto := &ops.Ticket{
		TicketId:      ticketId,
		OperationType: operationType,
		State:         ops.OpsTicketState_OPS_TICKET_STATE_NEW,
		Data:          data,
	}

	jsonData, err := json.Marshal(ticketDto)

	if err != nil {
		return err
	}

	return t.client.addInList(ctx, ticketsListKey, jsonData)

}

func (t *TicketStorage) GetTicketFromStorage(ctx context.Context) (*ops.Ticket, error) {
	jsonData, err := t.client.getFromList(ctx, ticketsListKey)

	if err != nil {
		return nil, err
	}

	var ticketDto ops.Ticket

	err = json.Unmarshal([]byte(*jsonData), &ticketDto)

	if err != nil {
		return nil, err
	}

	return &ticketDto, nil
}

func (t *TicketStorage) UpdateTicketInStorage(ctx context.Context, request *ops.Ticket) error {
	jsonData, err := json.Marshal(request)

	if err != nil {
		return err
	}

	return t.client.addInList(ctx, ticketsListKey, jsonData)
}
