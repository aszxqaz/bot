package mexc

import (
	"mexc-bot/client"
	"strconv"
	"time"
)

type accountResponse struct {
	Balances []client.Balance `json:"balances"`
}
type orderBookTicker struct {
	Symbol      string `json:"symbol"`
	BidPrice    string `json:"bidPrice"`
	BidQuantity string `json:"bidQty"`
	AskPrice    string `json:"askPrice"`
	AskQuantity string `json:"askQty"`
}

func (o *orderBookTicker) toOrderBookTicker() (*client.OrderBookTicker, error) {
	bidPrice, err := strconv.ParseFloat(o.BidPrice, 64)
	if err != nil {
		return nil, err
	}
	bidQuantity, err := strconv.ParseFloat(o.BidQuantity, 64)
	if err != nil {
		return nil, err
	}
	askPrice, err := strconv.ParseFloat(o.AskPrice, 64)
	if err != nil {
		return nil, err
	}
	askQuantity, err := strconv.ParseFloat(o.AskQuantity, 64)
	if err != nil {
		return nil, err
	}
	return &client.OrderBookTicker{
		Symbol:      o.Symbol,
		BidPrice:    bidPrice,
		BidQuantity: bidQuantity,
		AskPrice:    askPrice,
		AskQuantity: askQuantity,
	}, nil
}

// d	json	dealsInfo
// > S	int	tradetype 1:buy 2:sell
// > T	long	tradeTime
// > c	string	clientOrderId
// > i	string	orderId
// > m	int	isMaker
// > p	string	price
// > st	byte	isSelfTrade
// > t	string	tradeId
// > v	string	quantity
// > a	string	deals amount
// > n	string	commission fee
// > N	string	commissionAsset）
// s	string	symbol
// t	long	eventTime

type deal struct {
	TradeType int    `json:"S"`
	Price     string `json:"p"`
	OrderId   string `json:"i"`
	TradeId   string `json:"t"`
	Quantity  string `json:"v"`
	TradeTime int64  `json:"T"`
}

func (d *deal) toDeal(symbol string) (*client.Deal, error) {
	price, err := strconv.ParseFloat(d.Price, 64)
	if err != nil {
		return nil, err
	}
	quantity, err := strconv.ParseFloat(d.Quantity, 64)
	if err != nil {
		return nil, err
	}
	tradeTime := time.UnixMilli(d.TradeTime)
	return &client.Deal{
		TradeType: d.TradeType,
		Price:     price,
		OrderId:   d.OrderId,
		TradeId:   d.TradeId,
		Quantity:  quantity,
		TradeTime: tradeTime,
		Symbol:    symbol,
	}, nil
}

type wsDealResponse struct {
	Endpoint  string `json:"c"`
	Symbol    string `json:"s"`
	Deal      deal   `json:"d"`
	Timestamp int    `json:"t"`
}

type wsResponse struct {
	Endpoint string `json:"c"`
}

type wsTicker struct {
	AskQuantity string `json:"A"`
	BidQuantity string `json:"B"`
	AskPrice    string `json:"a"`
	BidPrice    string `json:"b"`
}

type wsTickerResponse struct {
	Endpoint  string   `json:"c"`
	Ticker    wsTicker `json:"d"`
	Timestamp int      `json:"t"`
	Symbol    string   `json:"s"`
}

func (w *wsTickerResponse) toTicker() (*client.OrderBookTicker, error) {
	askPrice, err := strconv.ParseFloat(w.Ticker.AskPrice, 64)
	if err != nil || askPrice == 0 {
		return nil, err
	}
	bidPrice, err := strconv.ParseFloat(w.Ticker.BidPrice, 64)
	if err != nil || bidPrice == 0 {
		return nil, err
	}
	askQuantity, err := strconv.ParseFloat(w.Ticker.AskQuantity, 64)
	if err != nil || askQuantity == 0 {
		return nil, err
	}
	bidQuantity, err := strconv.ParseFloat(w.Ticker.BidQuantity, 64)
	if err != nil || bidQuantity == 0 {
		return nil, err
	}
	return &client.OrderBookTicker{
		Symbol:      w.Symbol,
		BidPrice:    bidPrice,
		BidQuantity: bidQuantity,
		AskPrice:    askPrice,
		AskQuantity: askQuantity,
	}, nil
}

// WS ACCOUNT UPDATES

type wsAccountUpdate struct {
	Asset  string `json:"a"`
	Free   string `json:"f"`
	Locked string `json:"l"`
}

type wsAccountUpdateMessage struct {
	Endpoint string          `json:"c"`
	Update   wsAccountUpdate `json:"d"`
}

func (u *wsAccountUpdate) toAccountUpdate() (*client.Balance, error) {
	free, err := strconv.ParseFloat(u.Free, 64)
	if err != nil {
		return nil, err
	}
	locked, err := strconv.ParseFloat(u.Locked, 64)
	if err != nil {
		return nil, err
	}
	return &client.Balance{
		Asset:  u.Asset,
		Free:   free,
		Locked: locked,
	}, nil
}

// WS ACCOUNT ORDERS
type wsAccountOrder struct {
	RemainAmount       string `json:"A"`
	TradeType          int    `json:"S"`
	RemainQuantity     string `json:"V"`
	Amount             string `json:"a"`
	OrderId            string `json:"i"`
	Price              string `json:"p"`
	Status             int    `json:"s"`
	CumulativeQuantity string `json:"cv"`
	CumulativeAmount   string `json:"ca"`
}

type wsAccountOrdersMessage struct {
	Endpoint  string         `json:"c"`
	Symbol    string         `json:"s"`
	Timestamp int64          `json:"t"`
	Data      wsAccountOrder `json:"d"`
}

func (m *wsAccountOrdersMessage) toOrderUpdate() (*client.OrderUpdate, error) {
	price, err := strconv.ParseFloat(m.Data.Price, 64)
	if err != nil {
		return nil, err
	}

	remainQuantity, err := strconv.ParseFloat(m.Data.RemainQuantity, 64)
	if err != nil {
		return nil, err
	}

	remainAmount, err := strconv.ParseFloat(m.Data.RemainAmount, 64)
	if err != nil {
		return nil, err
	}

	amount, err := strconv.ParseFloat(m.Data.Amount, 64)
	if err != nil {
		return nil, err
	}

	cumulativeQuantity, err := strconv.ParseFloat(m.Data.CumulativeQuantity, 64)
	if err != nil {
		return nil, err
	}

	cumulativeAmount, err := strconv.ParseFloat(m.Data.CumulativeAmount, 64)
	if err != nil {
		return nil, err
	}

	return &client.OrderUpdate{
		Symbol:             m.Symbol,
		Id:                 m.Data.OrderId,
		Status:             m.Data.Status,
		Price:              price,
		CumulativeQuantity: cumulativeQuantity,
		CumulativeAmount:   cumulativeAmount,
		RemainAmount:       remainAmount,
		RemainQuantity:     remainQuantity,
		Amount:             amount,
		Timestamp:          time.UnixMilli(m.Timestamp),
		TradeType:          m.Data.TradeType,
	}, nil
}

// WS PARTIAL BOOK DEPTH STREAM
type wsPartialDepth struct {
	Price    string `json:"p"`
	Quantity string `json:"v"`
}

type wsPartialDepthMessageData struct {
	Asks []wsPartialDepth `json:"asks"`
	Bids []wsPartialDepth `json:"bids"`
}

type wsPartialDepthMessage struct {
	Endpoint  string                    `json:"c"`
	Symbol    string                    `json:"s"`
	Timestamp int64                     `json:"t"`
	Data      wsPartialDepthMessageData `json:"d"`
}

func (m *wsPartialDepthMessage) toPartialDepth() (*client.PartialDepth, error) {
	asks := make([]client.PartialDepthPair, 0, len(m.Data.Asks))
	bids := make([]client.PartialDepthPair, 0, len(m.Data.Bids))

	for _, ask := range m.Data.Asks {
		price, err := strconv.ParseFloat(ask.Price, 64)
		if err != nil {
			return nil, err
		}
		quantity, err := strconv.ParseFloat(ask.Quantity, 64)
		if err != nil {
			return nil, err
		}
		asks = append(asks, client.PartialDepthPair{Price: price, Quantity: quantity})
	}

	for _, bid := range m.Data.Bids {
		price, err := strconv.ParseFloat(bid.Price, 64)
		if err != nil {
			return nil, err
		}
		quantity, err := strconv.ParseFloat(bid.Quantity, 64)
		if err != nil {
			return nil, err
		}
		bids = append(bids, client.PartialDepthPair{Price: price, Quantity: quantity})
	}

	timestamp := time.UnixMilli(m.Timestamp)

	return &client.PartialDepth{
		Symbol:    m.Symbol,
		Asks:      asks,
		Bids:      bids,
		Timestamp: timestamp,
	}, nil
}
