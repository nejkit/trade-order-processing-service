package staticerr

import "errors"

var (
	ErrorRabbitConnectionFail = errors.New("RabbitUnvailable")
	ErrorResourceIsLocked     = errors.New("ResourceIsLocked")
	ErrorStockBookIsEmpty     = errors.New("StockBookIsEmpty")
)
