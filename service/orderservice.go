package service

import (
	"context"
	"time"
	"trade-order-processing-service/external/OPS"
	"trade-order-processing-service/external/balances"
	"trade-order-processing-service/models"
	"trade-order-processing-service/utils"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type iOrderStorage interface {
	AddOrderToStorage(ctx context.Context, orderInfo models.OrderModel) error
	GetOrderFromStorage(ctx context.Context, id string) (*models.OrderModel, error)
	UpdateOrderInfo(ctx context.Context, orderInfo models.OrderModel) error
	DeleteOrderFromStorage(ctx context.Context, id string) error
	AddInStockBook(ctx context.Context, orderInfo models.OrderModel) error
	DropFromStockBook(ctx context.Context, orderInfo models.OrderModel) error
	TryLockOrder(ctx context.Context, id string, guid string) error
	TryUnlockOrder(ctx context.Context, id string, guid string) error
	GetStockPriceByCurrencyPairAndDirection(ctx context.Context, currencyPair string, direction int) (float64, error)
}

type iTicketStorage interface {
	AddNewTicket(ctx context.Context, operationType OPS.OpsTicketOperation, ticketData protoreflect.ProtoMessage) error
}

type OrderService struct {
	orderStorage  iOrderStorage
	ticketStorage iTicketStorage
}

func NewOrderService(orderStorage iOrderStorage, ticketStorage iTicketStorage) *OrderService {
	return &OrderService{orderStorage: orderStorage, ticketStorage: ticketStorage}
}

func (o *OrderService) CreateOrder(ctx context.Context, request *OPS.OpsCreateOrderRequest) {
	orderId := uuid.NewString()

	orderInfo := models.OrderModel{
		OrderId:      orderId,
		AccountId:    request.AccountId,
		AssetId:      request.AssetId,
		CurrencyPair: request.CurrencyPair,
		Direction:    int(request.Direction),
		LimitPrice:   request.LimitPrice,
		AskVolume:    request.AskVolume,
		Type:         int(request.Type),
		CreationDate: time.Now().UTC().UnixMilli(),
		UpdatedDate:  time.Now().UTC().UnixMilli(),
		State:        int(OPS.OpsOrderState_OPS_ORDER_STATE_NEW),
	}

	logrus.WithField("requestId", request.Id).Infoln("Order id for this request: ", orderId)

	if err := o.orderStorage.AddOrderToStorage(ctx, orderInfo); err != nil {
		logrus.WithField("orderId", orderId).Errorln("Creation order failed, reason: ", err.Error())
		return
	}

	logrus.WithField("orderId", orderId).Errorln("Creation order successfully")

	lockAmount, err := o.calculateLockAmount(ctx, orderInfo)

	if err != nil {
		logrus.WithField("orderId", orderId).Errorln("Lock order failed, reason: ", err.Error())
		return
	}

	err = o.ticketStorage.AddNewTicket(ctx, OPS.OpsTicketOperation_OPS_TICKET_OPERATION_LOCK_BALANCE, &balances.BpsLockBalanceRequest{
		Id:           orderId,
		AssetId:      request.AssetId,
		AccountId:    request.AccountId,
		CurrencyCode: utils.GetOfferCurrencyCode(request.CurrencyPair, int(request.Direction)),
		Amount:       lockAmount,
	})

	if err != nil {
		logrus.WithField("orderId", orderId).Errorln("Save ticket for lock, reason: ", err.Error())

	}
}

func (s *OrderService) calculateLockAmount(ctx context.Context, model models.OrderModel) (float64, error) {

	if model.Direction == int(OPS.OpsOrderDirection_OPS_ORDER_DIRECTION_SELL) {
		return model.AskVolume, nil
	}

	price := model.LimitPrice
	var err error
	if model.Type == int(OPS.OpsOrderType_OPS_ORDER_TYPE_MARKET) {
		price, err = s.orderStorage.GetStockPriceByCurrencyPairAndDirection(ctx, model.CurrencyPair, model.Direction)
		if err != nil {
			return 0, err
		}
	}

	return price * model.AskVolume, nil

}
