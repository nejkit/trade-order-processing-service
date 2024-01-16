package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"trade-order-processing-service/external/OPS"
	"trade-order-processing-service/models"
	"trade-order-processing-service/staticerr"
	"trade-order-processing-service/utils"

	"github.com/redis/go-redis/v9"
)

const (
	ordersHashKey              = "orders"
	ordersPriceKey             = "orders:price"
	ordersCreationDateKey      = "orders:creation_date"
	ordersCurrencyDirectionKey = "orders:%s:%d"
	ordersExpirationDateKey    = "orders:expire"
	ordersStockPrices          = "orders:stock:"
	ordersLocksKey             = "lock_order:"
	matchingCandidatesIndex    = "orders:matching:"
	limitPriceIndex            = "orders:limit:"
)

var (
	OrderPriceSortDirection = map[int]int{
		int(OPS.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY.Number()):  1,
		int(OPS.OpsOrderDirection_OPS_ORDER_DIRECTION_SELL.Number()): -1,
	}
)

func buildStockKey(currencyPair string, direction int) string {
	return fmt.Sprintf(ordersStockPrices+"%s:%d", currencyPair, direction)
}

type OrdersStorage struct {
	client *RedisClient
}

func newOrdersStorage(cleint *RedisClient) *OrdersStorage {
	return &OrdersStorage{client: cleint}
}

func (o *OrdersStorage) AddOrderToStorage(ctx context.Context, orderInfo models.OrderModel) error {
	jsonData, err := json.Marshal(orderInfo)

	if err != nil {
		return err
	}

	if err = o.client.addInHash(ctx, ordersHashKey, orderInfo.OrderId, jsonData); err != nil {
		return err
	}

	return nil
}

func (o *OrdersStorage) GetOrderFromStorage(ctx context.Context, id string) (*models.OrderModel, error) {
	jsonData, err := o.client.getFromHash(ctx, ordersHashKey, id)

	if err != nil {
		return nil, err
	}

	var orderInfo models.OrderModel

	if err = json.Unmarshal([]byte(*jsonData), &orderInfo); err != nil {
		return nil, err
	}

	return &orderInfo, nil

}

func (o *OrdersStorage) UpdateOrderInfo(ctx context.Context, orderInfo models.OrderModel) error {
	orderInfo.UpdatedDate = time.Now().UTC().Unix()

	jsonData, err := json.Marshal(orderInfo)

	if err != nil {
		return err
	}

	if err = o.client.addInHash(ctx, ordersHashKey, orderInfo.OrderId, jsonData); err != nil {
		return err
	}

	return nil
}

func (o *OrdersStorage) DeleteOrderFromStorage(ctx context.Context, id string) error {

	if err := o.client.removeFromHash(ctx, ordersHashKey, id); err != nil {
		return err
	}

	return nil
}

func (o *OrdersStorage) AddInStockBook(ctx context.Context, orderInfo models.OrderModel) error {
	tx := o.client.performTx(ctx)

	err := tx.
		addInZSet(ctx, ordersPriceKey, orderInfo.OrderId, orderInfo.LimitPrice).
		addInZSet(ctx, ordersCreationDateKey, orderInfo.OrderId, float64(orderInfo.CreationDate)).
		addInSet(ctx, fmt.Sprintf(ordersCurrencyDirectionKey, orderInfo.CurrencyPair, orderInfo.Direction), orderInfo.OrderId).
		addInHash(ctx, ordersExpirationDateKey, orderInfo.OrderId, orderInfo.ExpirationDate).
		incrementHash(ctx, buildStockKey(orderInfo.CurrencyPair, orderInfo.Direction), fmt.Sprintf("%f", orderInfo.LimitPrice), orderInfo.AskVolume).
		execTx(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o OrdersStorage) GetStockPriceByCurrencyPairAndDirection(ctx context.Context, currencyPair string, direction int) (float64, error) {
	values, err := o.client.getAllFromHash(ctx, buildStockKey(currencyPair, direction))

	if err != nil {
		return 0, err
	}

	totalVolume := 0.00
	avgPrice := 0.00

	for price, volume := range values {
		floatPrice, err := strconv.ParseFloat(price, 2)
		if err != nil {
			continue
		}
		floatVolume, err := strconv.ParseFloat(volume, 2)

		if err != nil {
			continue
		}

		totalVolume += floatVolume
		avgPrice += floatPrice * floatVolume
	}

	if totalVolume == 0 {
		return 0, staticerr.ErrorStockBookIsEmpty
	}

	return avgPrice / totalVolume, nil
}

func (o *OrdersStorage) DropFromStockBook(ctx context.Context, orderInfo models.OrderModel) error {
	tx := o.client.performTx(ctx)

	err := tx.
		removeFromZSet(ctx, ordersPriceKey, orderInfo.OrderId).
		removeFromZSet(ctx, ordersCreationDateKey, orderInfo.OrderId).
		removeFromSet(ctx, fmt.Sprintf(ordersCurrencyDirectionKey, orderInfo.CurrencyPair, orderInfo.Direction), orderInfo.OrderId).
		removeFromHash(ctx, ordersExpirationDateKey, orderInfo.OrderId).
		decrementHash(ctx, buildStockKey(orderInfo.CurrencyPair, orderInfo.Direction), fmt.Sprintf("%f", orderInfo.LimitPrice), orderInfo.AskVolume).
		execTx(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *OrdersStorage) TryLockOrder(ctx context.Context, id string, guid string) error {
	return o.client.setNX(ctx, ordersLocksKey+id, guid, time.Minute)
}

func (o *OrdersStorage) TryUnlockOrder(ctx context.Context, id string, guid string) error {
	return o.client.deleteWithValue(ctx, ordersLocksKey+id, guid)
}

func (o *OrdersStorage) GetOrdersForMatch(ctx context.Context, id string) ([]string, error) {
	orderInfo, err := o.GetOrderFromStorage(ctx, id)

	if err != nil {
		return nil, err
	}

	priceIndex := ordersPriceKey
	if orderInfo.Type == int(OPS.OpsOrderType_OPS_ORDER_TYPE_LIMIT) {
		priceIndex, err := o.getPriceIndexForLimit(ctx, *orderInfo)

		if err != nil {
			return nil, err
		}

		defer o.client.deleteKey(ctx, *priceIndex)
	}

	_, err = o.client.cli.ZInterStore(ctx, matchingCandidatesIndex+id, &redis.ZStore{
		Keys:    []string{priceIndex, ordersCreationDateKey, fmt.Sprintf(ordersCurrencyDirectionKey, orderInfo.CurrencyPair, utils.GetDirectionForBuildMatchingIndex(orderInfo.Direction))},
		Weights: []float64{float64(time.Now().Unix()*100) * float64(OrderPriceSortDirection[orderInfo.Direction]), 1, 0},
	}).Result()

	if err != nil {
		return nil, err
	}
	defer o.client.deleteKey(ctx, matchingCandidatesIndex+id)

	ids, err := o.client.cli.ZRange(ctx, matchingCandidatesIndex+id, 0, -1).Result()

	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (o OrdersStorage) getPriceIndexForLimit(ctx context.Context, orderInfo models.OrderModel) (*string, error) {
	if OPS.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY == OPS.OpsOrderDirection(orderInfo.Direction) {
		return o.prepareIndexWithLimitOption(ctx, LimitOptions{
			maxPrice: orderInfo.LimitPrice,
			minPrice: 0,
		})
	}
	return o.prepareIndexWithLimitOption(ctx, LimitOptions{
		maxPrice: 0,
		minPrice: orderInfo.LimitPrice,
	})
}

func (o *OrdersStorage) prepareIndexWithLimitOption(ctx context.Context, options LimitOptions) (*string, error) {
	indexName := limitPriceIndex + fmt.Sprintf("%f", options.minPrice) + ":" + fmt.Sprintf("%f", options.maxPrice)
	_, err := o.client.cli.ZInterStore(ctx, indexName, &redis.ZStore{
		Keys: []string{ordersPriceKey},
	}).Result()

	if err != nil {
		return nil, err
	}

	_, err = o.client.cli.ZRemRangeByScore(ctx, indexName, "-inf", fmt.Sprintf("%f", options.minPrice-0.01)).Result()

	if err != nil {
		return nil, err
	}

	if options.maxPrice > 0 {
		_, err = o.client.cli.ZRemRangeByScore(ctx, indexName, fmt.Sprintf("%f", options.maxPrice+0.01), "+inf").Result()

		if err != nil {
			return nil, err
		}
	}
	return &indexName, nil
}
