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

	strategy := NewPayeerSharesStrategy(
		payeerClient,
		binanceClient,
		&PayeerSharesStrategyOptions{
			RefetchBalanceDelay: time.Millisecond * 200,
			OrdersFetchInterval: time.Millisecond * 5,
			Shares: []PayeerSharesStrategyShare{
				{
					ID:                    "BUYER-1",
					Action:                payeer.ACTION_BUY,
					Pair:                  payeer.PAIR_ETHUSDT,
					BinanceSymbol:         binance.SYMBOL_ETHUSDT,
					Share:                 decimal.RequireFromString(".1"),
					BinancePriceRatio:     decimal.RequireFromString(".9798"),
					BinanceTickerInterval: time.Millisecond * 100,
					LoopInterval:          time.Second * 10,
				},
				{
					ID:                    "BUYER-2",
					Action:                payeer.ACTION_BUY,
					Pair:                  payeer.PAIR_ETHUSDT,
					BinanceSymbol:         binance.SYMBOL_ETHUSDT,
					Share:                 decimal.RequireFromString(".15"),
					BinancePriceRatio:     decimal.RequireFromString(".9799"),
					BinanceTickerInterval: time.Millisecond * 100,
					LoopInterval:          time.Second * 10,
				},
			},
		},
	)

	strategy.Run()
}
