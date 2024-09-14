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
			payeer.PAIR_BTCUSDT: binance.SYMBOL_BTCUSDT,
		},
		BinanceTickerInterval: time.Millisecond * 100,
		MaxPriceRatio:         "1.001",
		// PlacementValueOffset:   "1000",
		ReplacementValueOffset: "50",
		SelectorConfig: &payeer.PayeerPriceSelectorConfig{
			PlacementValueOffset:   decimal.NewFromInt(15),
			ElevationPriceFraction: decimal.RequireFromString(".00005"),
			MaxWmaSurplus:          decimal.RequireFromString(".003"),
			WmaTakeAmount:          decimal.RequireFromString(".025"),
			WmaTake:                0,

			Symbol:                  binance.SYMBOL_BTCUSDT,
			BidMaxBinancePriceRatio: decimal.RequireFromString(".999"),
			AskMinBinancePriceRatio: decimal.RequireFromString("1.001"),
		},
		BuyEnabled:  true,
		SellEnabled: true,
		Amount:      decimal.RequireFromString("0.0001"),
	})
	strategy.Run()
}
