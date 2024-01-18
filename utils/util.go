package utils

import (
	"strings"
	"trade-order-processing-service/external/ops"
)

func GetOfferCurrencyCode(currencyPair string, direction int) string {
	return strings.Split(currencyPair, "/")[direction]
}

func GetAskedCurrencyCode(currencyPair string, direction int) string {
	if direction == int(ops.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY) {
		return strings.Split(currencyPair, "/")[0]
	}
	return strings.Split(currencyPair, "/")[1]
}

func GetDirectionForBuildMatchingIndex(direction int) int {
	if direction == int(ops.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY) {
		return int(ops.OpsOrderDirection_OPS_ORDER_DIRECTION_SELL)
	}
	return int(ops.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY)
}

func Min(a float64, b float64) float64 {
	if a < b {
		return a
	}

	return b

}
