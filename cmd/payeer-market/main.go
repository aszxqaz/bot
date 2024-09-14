package main

import (
	"automata/client/binance"
	"automata/client/payeer"
	"log/slog"
	"os"
	"time"

	"github.com/shopspring/decimal"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	apiId := os.Getenv("API_ID")
	secret := os.Getenv("SECRET")

	payeerClient := payeer.NewClient(&payeer.Config{
		ApiId:  apiId,
		Secret: secret,
	})
	binanceClient := binance.NewClient()

	trader := NewPayeerMarketTrader(payeerClient, binanceClient, &PayeerMarketTraderOptions{
		Pairs: map[payeer.Pair]binance.Symbol{
			payeer.PAIR_ETHUSDT: binance.SYMBOL_ETHUSDT,
		},
		BinanceTickerInterval: time.Millisecond * 20,
		TradeLoopInterval:     time.Second * 10,
		BidMinRatio:           decimal.RequireFromString("999"),
		AskMaxRatio:           decimal.RequireFromString("1.01"),
		MaxBuyAmount:          decimal.Zero,
		QuoteMult:             decimal.RequireFromString("1.0"),
	})

	trader.Start()
}
