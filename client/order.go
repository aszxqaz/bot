package client

import (
	"encoding/json"
	"strconv"
	"time"
)

type OrderSide string

const (
	BuyOrderSide  OrderSide = "BUY"
	SellOrderSide OrderSide = "SELL"
)

type OrderType string

const (
	LimitOrderType  OrderType = "LIMIT"
	MarketOrderType OrderType = "MARKET"
)

type jsonorder struct {
	Symbol  string    `json:"symbol"`
	Id      string    `json:"orderId"`
	Price   string    `json:"price"`
	OrigQty string    `json:"origQty"`
	Type    OrderType `json:"type"`
	Side    OrderSide `json:"side"`
	Time    int       `json:"transactTime"`
}

type Order struct {
	Symbol  string
	Id      string
	Price   float64
	OrigQty float64
	Type    OrderType
	Side    OrderSide
	Time    time.Time
}

func (o *Order) UnmarshalJSON(data []byte) error {
	var jsonorder jsonorder
	err := json.Unmarshal(data, &jsonorder)
	if err != nil {
		return err
	}
	o.Symbol = jsonorder.Symbol
	o.Id = jsonorder.Id
	o.Price, err = strconv.ParseFloat(jsonorder.Price, 64)
	if err != nil {
		return err
	}
	o.OrigQty, err = strconv.ParseFloat(jsonorder.OrigQty, 64)
	if err != nil {
		return err
	}
	o.Type = OrderType(jsonorder.Type)
	o.Side = OrderSide(jsonorder.Side)
	o.Time = time.Unix(int64(jsonorder.Time), 0)
	return nil
}
