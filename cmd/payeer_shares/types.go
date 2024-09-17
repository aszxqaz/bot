package main

import (
	"automata/client/binance"
	"automata/client/payeer"
	payeerFetcher "automata/client/payeer/fetcher"
	"automata/msync"
	"time"

	"github.com/shopspring/decimal"
)

type PayeerSharesStrategyOptions struct {
	Shares              []PayeerSharesStrategyShare
	RefetchBalanceDelay time.Duration
	OrdersFetchInterval time.Duration
}

type PayeerSharesStrategyShare struct {
	ID                    string
	Action                payeer.Action
	Pair                  payeer.Pair
	BinanceSymbol         binance.Symbol
	BinanceTickerInterval time.Duration
	Share                 decimal.Decimal
	BinancePriceRatio     decimal.Decimal
	RemainingAmountChange decimal.Decimal
	LoopInterval          time.Duration
}

type ShareOrderInfo struct {
	OrderId int
	Order   *payeer.OrderParams
	Time    time.Time
}

type payeerSharesStrategyStore struct {
	info           *payeer.InfoResponse
	binanceTickers *msync.MuMap[binance.Symbol, binance.OrderBookTickerStreamResult]
	balance        *msync.MuMap[string, payeer.Balance]
	orders         *msync.MuMap[payeer.Pair, payeer.PairsOrderInfo]
	shareOrders    *msync.MuMap[string, ShareOrderInfo]
}

type payeerSharesStrategyState struct {
}

type PayeerSharesStrategy struct {
	options       *PayeerSharesStrategyOptions
	fetcher       *payeerFetcher.Fetcher
	binanceClient *binance.Client
	store         *payeerSharesStrategyStore
	state         *payeerSharesStrategyState
}

func NewPayeerSharesStrategy(
	payeerClient *payeer.Client,
	binanceClient *binance.Client,
	options *PayeerSharesStrategyOptions,
) *PayeerSharesStrategy {
	fetcher := payeerFetcher.NewFetcher(payeerClient)
	return &PayeerSharesStrategy{
		binanceClient: binanceClient,
		options:       options,
		fetcher:       fetcher,
		store: &payeerSharesStrategyStore{
			orders:         msync.NewMuMap[payeer.Pair, payeer.PairsOrderInfo](),
			balance:        msync.NewMuMap[string, payeer.Balance](),
			binanceTickers: msync.NewMuMap[binance.Symbol, binance.OrderBookTickerStreamResult](),
			shareOrders:    msync.NewMuMap[string, ShareOrderInfo](),
		},
	}
}
