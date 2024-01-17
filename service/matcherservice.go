package service

import (
	"context"
	"trade-order-processing-service/external/OPS"
	"trade-order-processing-service/utils"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type MatcherService struct {
	orderStorage  iOrderStorage
	ticketStorage iTicketStorage
}

func NewMatcherService(orderStorage iOrderStorage, ticketStorage iTicketStorage) *MatcherService {
	return &MatcherService{orderStorage: orderStorage, ticketStorage: ticketStorage}
}

func (m *MatcherService) MatchOrder(ctx context.Context, matchData *OPS.OpsOrderInfo) {

	lockId := uuid.NewString()

	orders, err := m.orderStorage.GetOrdersForMatch(ctx, matchData.OrderId)

	if err != nil {
		logrus.WithField("orderId", matchData.OrderId).Infoln("Orders stock book is empty, reject matching...")
		if err = m.rejectOrderMatching(ctx, matchData); err != nil {
			logrus.WithField("orderId", matchData.OrderId).Errorln("Failed reject matching, exit...")
		}
		logrus.WithField("orderId", matchData.OrderId).Infoln("Matching was rejected, exit...")
		return
	}

	for _, oId := range orders {
		if err = m.orderStorage.TryLockOrder(ctx, oId, lockId); err != nil {
			logrus.WithFields(logrus.Fields{
				"orderId":        matchData.OrderId,
				"matchedOrderId": oId}).Warningln("MatchedOrderId is locked, skipping...")
			continue
		}

	}
}

func (m *MatcherService) rejectOrderMatching(ctx context.Context, orderData *OPS.OpsOrderInfo) error {

	if orderData.Type == OPS.OpsOrderType_OPS_ORDER_TYPE_MARKET {
		logrus.WithField("orderId", orderData.OrderId).Infoln("Order is market, convert to limit")

		orderData.Type = OPS.OpsOrderType_OPS_ORDER_TYPE_LIMIT

		if err := m.orderStorage.UpdateOrderInfo(ctx, utils.MapProtoOrderInfoToModel(orderData)); err != nil {
			return err
		}
	}

	orderModel := utils.MapProtoOrderInfoToModel(orderData)

	logrus.WithField("orderId", orderData.OrderId).Infoln("Add order in stock book")

	if err := m.orderStorage.AddInStockBook(ctx, orderModel); err != nil {
		return err
	}

	return nil
}
