package main

import (
	"automata/client/binance"
	"automata/client/payeer"

	"log/slog"
	"os"
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
		MaxPriceRatio:          "1.001",
		PlacementValueOffset:   "1000",
		ReplacementValueOffset: "10000",
	})
	strategy.Run()
}
