package main

import (
	"automata/client/binance"
	"automata/client/payeer"
	"time"

	"log/slog"
	"os"

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

	strategy := NewVolumeOffsetStrategy(payeerClient, binanceClient, &ValueOffsetStrategyOptions{
		Pairs: map[payeer.Pair]binance.Symbol{
			payeer.PAIR_BTCRUB: binance.SYMBOL_BTCUSDT,
		},
		BinanceTickerInterval: time.Millisecond * 500,
		MaxPriceRatio:         "1.001",
		// PlacementValueOffset:   "1000",
		ReplacementValueOffset: "10000",
		SelectorConfig: &payeer.PayeerPriceSelectorConfig{
			PlacementValueOffset:   decimal.NewFromInt(5000),
			ElevationPriceFraction: decimal.RequireFromString(".00005"),
			MaxWmaSurplus:          decimal.RequireFromString(".005"),
			WmaTakeAmount:          decimal.RequireFromString(".025"),
			WmaTake:                0,
		},
		BuyEnabled:  true,
		SellEnabled: true,
	})
	strategy.Run()
}
