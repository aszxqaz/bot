package main

import (
	"automata/client/binance"
	"automata/client/payeer"

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
	binanceClient.Start()
	strategy := NewVolumeOffsetStrategy(payeerClient, binanceClient, &ValueOffsetStrategyOptions{
		MaxPriceRatio: "1.001",
		// PlacementValueOffset:   "1000",
		ReplacementValueOffset: "10000",
		SelectorConfig: &payeer.PayeerPriceSelectorConfig{
			PlacementValueOffset:   decimal.NewFromInt(5000),
			ElevationPriceFraction: decimal.RequireFromString(".00005"),
			MaxWmaSurplus:          decimal.RequireFromString(".005"),
			WmaTake:                15,
		},
	})
	strategy.Run()
}
