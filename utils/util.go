package utils

import (
	"strings"
	"trade-order-processing-service/external/OPS"
)

func GetOfferCurrencyCode(currencyPair string, direction int) string {
	return strings.Split(currencyPair, "/")[direction]
}

func GetAskedCurrencyCode(currencyPair string, direction int) string {
	if direction == int(OPS.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY) {
		return strings.Split(currencyPair, "/")[0]
	}
	return strings.Split(currencyPair, "/")[1]
}

func GetDirectionForBuildMatchingIndex(direction int) int {
	if direction == int(OPS.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY) {
		return int(OPS.OpsOrderDirection_OPS_ORDER_DIRECTION_SELL)
	}
	return int(OPS.OpsOrderDirection_OPS_ORDER_DIRECTION_BUY)
}
