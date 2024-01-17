package service

import (
	"context"
	"testing"
	"trade-order-processing-service/external/OPS"
)

func TestOrderService_CreateOrder(t *testing.T) {
	type fields struct {
		orderStorage  iOrderStorage
		ticketStorage iTicketStorage
	}
	type args struct {
		ctx     context.Context
		request *OPS.OpsCreateOrderRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OrderService{
				orderStorage:  tt.fields.orderStorage,
				ticketStorage: tt.fields.ticketStorage,
			}
			o.CreateOrder(tt.args.ctx, tt.args.request)
		})
	}
}
