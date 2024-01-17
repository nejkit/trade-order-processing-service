package models

type OrderModel struct {
	OrderId        string  `json:"order_id,omitempty"`
	AccountId      string  `json:"account_id,omitempty"`
	AssetId        string  `json:"asset_id,omitempty"`
	CurrencyPair   string  `json:"currency_pair,omitempty"`
	Direction      int     `json:"direction,omitempty"`
	LimitPrice     float64 `json:"limit_price,omitempty"`
	AskVolume      float64 `json:"ask_volume,omitempty"`
	FilledVolume   float64 `json:"filled_volume,omitempty"`
	Type           int     `json:"type,omitempty"`
	FilledPrice    float64 `json:"filled_price,omitempty"`
	CreationDate   int64   `json:"creation_date,omitempty"`
	UpdatedDate    int64   `json:"updated_date,omitempty"`
	ExpirationDate int64   `json:"expiration_date,omitempty"`
	MatchingDate   int64   `json:"matching_date,omitempty"`
	TransferId     string  `json:"transfer_id,omitempty"`
	State          int     `json:"state,omitempty"`
	ParentId       string  `json:"parent_id,omitempty"`
	ExchangeId     string  `json:"exchange_id,omitempty"`
}
