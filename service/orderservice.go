package service

import (
	"context"
	"time"

	"trade-order-processing-service/external/bps"
	"trade-order-processing-service/external/ops"
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
	GetOrdersForMatch(ctx context.Context, id string) ([]string, error)
}

type iTicketStorage interface {
	AddNewTicket(ctx context.Context, operationType ops.OpsTicketOperation, ticketData protoreflect.ProtoMessage) error
}

type OrderService struct {
	orderStorage  iOrderStorage
	ticketStorage iTicketStorage
}

func NewOrderService(orderStorage iOrderStorage, ticketStorage iTicketStorage) *OrderService {
	return &OrderService{orderStorage: orderStorage, ticketStorage: ticketStorage}
}

func (o *OrderService) CreateOrder(ctx context.Context, request *ops.OpsCreateOrderRequest) {
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
		State:        int(ops.OpsOrderState_OPS_ORDER_STATE_NEW),
	}

	logrus.WithField("requestId", request.Id).Infoln("Order id for this request: ", orderId)

	if err := o.enrichMarketOrderStockPrice(ctx, &orderInfo); err != nil {
		logrus.WithField("orderId", orderId).Errorln("Fail enrich market order stockPrice, reason: ", err.Error())
	}

	if err := o.orderStorage.AddOrderToStorage(ctx, orderInfo); err != nil {
		logrus.WithField("orderId", orderId).Errorln("Creation order failed, reason: ", err.Error())
		return
	}

	logrus.WithField("orderId", orderId).Errorln("Creation order successfully")

	if err := o.ticketStorage.AddNewTicket(ctx, ops.OpsTicketOperation_OPS_TICKET_OPERATION_ORDER_NOTIFICATION, utils.MapOrderInfoToProto(orderInfo)); err != nil {
		logrus.WithField("orderId", orderId).Errorln("Fail save ticket for lock, reason: ", err.Error())

	}

	lockAmount, err := o.calculateLockAmount(ctx, orderInfo)

	if err != nil {
		logrus.WithField("orderId", orderId).Errorln("Lock balance order failed, reason: ", err.Error())
		return
	}

	err = o.ticketStorage.AddNewTicket(ctx, ops.OpsTicketOperation_OPS_TICKET_OPERATION_LOCK_BALANCE, &bps.BpsLockBalanceRequest{
		Id:           orderId,
		AssetId:      request.AssetId,
		AccountId:    request.AccountId,
		CurrencyCode: utils.GetOfferCurrencyCode(request.CurrencyPair, int(request.Direction)),
		Amount:       lockAmount,
	})

	if err != nil {
		logrus.WithField("orderId", orderId).Errorln("Fail save ticket for lock, reason: ", err.Error())

	}
}

func (s *OrderService) ApproveOrderCreation(ctx context.Context, request *bps.BpsLockBalanceResponse) {

	logrus.WithField("orderId", request.Id).Infoln("Received response from bps, lockBalance: ", request.String())

	orderInfo, err := s.orderStorage.GetOrderFromStorage(ctx, request.Id)

	if err != nil {
		logrus.WithField("orderId", request.Id).Errorln("Internal error: ", err.Error())
		return
	}

	if request.Error != nil {
		logrus.WithField("orderId", request.Id).Infoln("Order not approved, reason: ", request.Error.ErrorCode, " Try to reject order")
		orderInfo.State = int(ops.OpsOrderState_OPS_ORDER_STATE_REJECTED)
		orderInfo.UpdatedDate = time.Now().UTC().UnixMilli()

		if err = s.orderStorage.DeleteOrderFromStorage(ctx, orderInfo.OrderId); err != nil {
			logrus.WithField("orderId", request.Id).Errorln("Internal error: ", err.Error())
			return
		}
		protoModel := utils.MapOrderInfoToProto(*orderInfo)
		protoModel.Cause = utils.MapBpsErrorToOpsError(request.Error)

		if err = s.ticketStorage.AddNewTicket(ctx, ops.OpsTicketOperation_OPS_TICKET_OPERATION_ORDER_NOTIFICATION, protoModel); err != nil {
			logrus.WithField("orderId", request.Id).Errorln("Internal error: ", err.Error())
			return
		}
		logrus.WithField("orderId", request.Id).Infoln("Order is rejected, send notification: ")
		return
	}
	logrus.WithField("orderId", request.Id).Infoln("Order is approved, Try to complete order")

	orderInfo.State = int(ops.OpsOrderState_OPS_ORDER_STATE_APPROVED)
	orderInfo.UpdatedDate = time.Now().UTC().UnixMilli()
	orderInfo.ExpirationDate = time.Now().UTC().Add(time.Hour * 72).UnixMilli()
	orderInfo.ExchangeId = request.BalanceId

	if err = s.orderStorage.UpdateOrderInfo(ctx, *orderInfo); err != nil {
		logrus.WithField("orderId", request.Id).Errorln("Internal error: ", err.Error())
		return
	}

	logrus.WithField("orderId", request.Id).Infoln("Order is completed")

	protoModel := utils.MapOrderInfoToProto(*orderInfo)

	if err = s.ticketStorage.AddNewTicket(ctx, ops.OpsTicketOperation_OPS_TICKET_OPERATION_ORDER_NOTIFICATION, protoModel); err != nil {
		logrus.WithField("orderId", request.Id).Errorln("Internal error: ", err.Error())
	}

	if err = s.ticketStorage.AddNewTicket(ctx, ops.OpsTicketOperation_OPS_TICKET_OPERATION_MATCH_ORDER, protoModel); err != nil {
		logrus.WithField("orderId", request.Id).Errorln("Internal error: ", err.Error())
	}

}

func (s *OrderService) calculateLockAmount(ctx context.Context, model models.OrderModel) (float64, error) {

	if model.Direction == int(ops.OpsOrderDirection_OPS_ORDER_DIRECTION_SELL) {
		return model.AskVolume, nil
	}

	return model.LimitPrice * model.AskVolume, nil

}

func (s *OrderService) enrichMarketOrderStockPrice(ctx context.Context, model *models.OrderModel) error {
	var err error
	if model.Type == int(ops.OpsOrderType_OPS_ORDER_TYPE_MARKET) {
		model.LimitPrice, err = s.orderStorage.GetStockPriceByCurrencyPairAndDirection(ctx, model.CurrencyPair, model.Direction)

		if err != nil {
			return err
		}
	}
	return nil
}
