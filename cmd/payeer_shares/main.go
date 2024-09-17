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
					Share:                 decimal.RequireFromString(".35"),
					BinancePriceRatio:     decimal.RequireFromString(".98"),
					BinanceTickerInterval: time.Millisecond * 100,
					LoopInterval:          time.Millisecond * 500,
				},
				{
					ID:                    "BUYER-2",
					Action:                payeer.ACTION_BUY,
					Pair:                  payeer.PAIR_ETHUSDT,
					BinanceSymbol:         binance.SYMBOL_ETHUSDT,
					Share:                 decimal.RequireFromString(".30"),
					BinancePriceRatio:     decimal.RequireFromString(".99"),
					BinanceTickerInterval: time.Millisecond * 100,
					LoopInterval:          time.Millisecond * 500,
				},
				{
					ID:                    "SELLER-1",
					Action:                payeer.ACTION_SELL,
					Pair:                  payeer.PAIR_ETHUSDT,
					BinanceSymbol:         binance.SYMBOL_ETHUSDT,
					Share:                 decimal.RequireFromString(".30"),
					BinancePriceRatio:     decimal.RequireFromString("1.01"),
					BinanceTickerInterval: time.Millisecond * 100,
					LoopInterval:          time.Millisecond * 500,
				},
				{
					ID:                    "SELLER-2",
					Action:                payeer.ACTION_SELL,
					Pair:                  payeer.PAIR_ETHUSDT,
					BinanceSymbol:         binance.SYMBOL_ETHUSDT,
					Share:                 decimal.RequireFromString(".35"),
					BinancePriceRatio:     decimal.RequireFromString("1.02"),
					BinanceTickerInterval: time.Millisecond * 100,
					LoopInterval:          time.Millisecond * 500,
				},
			},
		},
	)

	strategy.Run()
}
