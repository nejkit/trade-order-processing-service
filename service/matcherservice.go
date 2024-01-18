package service

import (
	"context"
	"time"
	"trade-order-processing-service/external/bps"
	"trade-order-processing-service/external/ops"
	"trade-order-processing-service/models"
	"trade-order-processing-service/staticerr"
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

func (m *MatcherService) MatchOrder(ctx context.Context, matchData *ops.OpsOrderInfo) {

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

	orderModel := utils.MapProtoOrderInfoToModel(matchData)

	for _, oId := range orders {

		logrus.WithFields(logrus.Fields{
			"orderId":        matchData.OrderId,
			"matchedOrderId": oId}).Infoln("Matching 1 stage: lock matchedOrderData:")

		if err = m.orderStorage.TryLockOrder(ctx, oId, lockId); err != nil {
			logrus.WithFields(logrus.Fields{
				"orderId":        matchData.OrderId,
				"matchedOrderId": oId}).Warningln("MatchedOrderId is locked, skipping...")
			continue
		}

		logrus.WithFields(logrus.Fields{
			"orderId":        matchData.OrderId,
			"matchedOrderId": oId}).Infoln("Matching 2 stage: change OrdersData:")

		matchingOrderInfo, err := m.orderStorage.GetOrderFromStorage(ctx, oId)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"orderId":        matchData.OrderId,
				"matchedOrderId": oId}).Warningln("Internal error, skip this order...")
			continue
		}

		if err = m.performMatchingOrders(ctx, &orderModel, matchingOrderInfo); err != nil {
			return
		}

	}
}

func (m *MatcherService) rejectOrderMatching(ctx context.Context, orderData *ops.OpsOrderInfo) error {

	if orderData.Type == ops.OpsOrderType_OPS_ORDER_TYPE_MARKET {
		logrus.WithField("orderId", orderData.OrderId).Infoln("Order is market, convert to limit")

		orderData.Type = ops.OpsOrderType_OPS_ORDER_TYPE_LIMIT

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

func (m *MatcherService) performMatchingOrders(ctx context.Context, firstOrder *models.OrderModel, secondOrder *models.OrderModel) error {

	matchingDate := time.Now().UTC().UnixMilli()

	if firstOrder.ExpirationDate < matchingDate {
		return staticerr.ErrorOrderExpired
	}

	if secondOrder.ExpirationDate < matchingDate {
		return staticerr.ErrorOrderExpired
	}

	filledVolume := utils.Min(firstOrder.AskVolume, secondOrder.AskVolume)

	transferId := uuid.NewString()

	firstOrder.FilledVolume = filledVolume
	firstOrder.MatchingDate = matchingDate
	firstOrder.FilledPrice = secondOrder.LimitPrice
	firstOrder.TransferId = transferId
	changeStateForMatchedOrder(firstOrder)

	secondOrder.FilledVolume = filledVolume
	secondOrder.MatchingDate = matchingDate
	secondOrder.FilledPrice = secondOrder.FilledVolume
	secondOrder.TransferId = transferId
	changeStateForMatchedOrder(secondOrder)

	if err := m.orderStorage.DropFromStockBook(ctx, *secondOrder); err != nil {
		return err
	}

	if err := m.orderStorage.UpdateOrderInfo(ctx, *firstOrder); err != nil {
		return err
	}

	if err := m.orderStorage.UpdateOrderInfo(ctx, *secondOrder); err != nil {
		return err
	}

	return nil
}

func changeStateForMatchedOrder(orderInfo *models.OrderModel) {

	orderInfo.State = int(ops.OpsOrderState_OPS_ORDER_STATE_PART_FILLED)

	if orderInfo.AskVolume == orderInfo.FilledVolume {
		orderInfo.State = int(ops.OpsOrderState_OPS_ORDER_STATE_FILLED)
	}
}

func (m *MatcherService) performTransfer(ctx context.Context, transferId string, firstOrder, secondOrder models.OrderModel) error {

	amounts := make(map[string]float64)

	for _, oInfo := range []models.OrderModel{firstOrder, secondOrder} {

		amounts[oInfo.ExchangeId] = oInfo.FilledVolume

		if oInfo.Direction == int(ops.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY) {
			amounts[oInfo.ExchangeId] = oInfo.FilledPrice * oInfo.FilledVolume
		}
	}

	transferRequest := &bps.BpsCreateTransferRequest{
		Id: uuid.NewString(),
		TransferData: []*bps.BpsTransferData{
			&bps.BpsTransferData{
				BalanceId: firstOrder.ExchangeId,
				Amount:    amounts[secondOrder.ExchangeId],
			},
			&bps.BpsTransferData{
				BalanceId: secondOrder.ExchangeId,
				Amount:    amounts[firstOrder.ExchangeId],
			},
		}}

	if err := m.ticketStorage.AddNewTicket(ctx, ops.OpsTicketOperation_OPS_TICKET_OPERATION_APPROVE_CREATION, transferRequest); err != nil {
		return err
	}

	return nil

}
