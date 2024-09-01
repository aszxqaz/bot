package client

import (
	"encoding/json"
	"strconv"
	"time"
)

const (
	ETHUSDC   = "ETHUSDC"
	STETHUSDC = "STETHUSDC"
)

type jsonbalance struct {
	Asset  string
	Free   string
	Locked string
}

type Balance struct {
	Asset  string
	Free   float64
	Locked float64
}

func (b *Balance) UnmarshalJSON(data []byte) error {
	var jsonbalance jsonbalance
	err := json.Unmarshal(data, &jsonbalance)
	if err != nil {
		return err
	}
	b.Asset = jsonbalance.Asset
	b.Free, err = strconv.ParseFloat(jsonbalance.Free, 64)
	if err != nil {
		return err
	}
	b.Locked, err = strconv.ParseFloat(jsonbalance.Locked, 64)
	if err != nil {
		return err
	}
	return nil
}

type OrderBookTicker struct {
	Symbol      string
	BidPrice    float64
	BidQuantity float64
	AskPrice    float64
	AskQuantity float64
}

const (
	TradeTypeBuy  int = 1
	TradeTypeSell int = 2
)

type Deal struct {
	Symbol    string
	TradeType int
	Price     float64
	Quantity  float64
	OrderId   string
	TradeId   string
	TradeTime time.Time
}

const (
	OrderStatusNew                      = 1
	OrderStatusFilled                   = 2
	OrderStatusPartiallyFilled          = 3
	OrderStatusCanceled                 = 4
	OrderStatusPartiallyFilledCancelled = 5
)

type OrderUpdate struct {
	Symbol             string
	Timestamp          time.Time
	RemainAmount       float64
	TradeType          int
	RemainQuantity     float64
	Amount             float64
	Id                 string
	Price              float64
	CumulativeQuantity float64
	CumulativeAmount   float64
	Status             int
}

type PartialDepthPair struct {
	Price    float64
	Quantity float64
}

type PartialDepth struct {
	Symbol    string
	Timestamp time.Time
	Asks      []PartialDepthPair
	Bids      []PartialDepthPair
}
