package main

import (
	"automata/client/binance"
	"log"
	"log/slog"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	binanceClient := binance.NewClient()
	binanceClient.Start()
	tickers, err := binanceClient.GetOrderBookTickers([]binance.Symbol{
		binance.SYMBOL_BTCUSDT,
	})
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("Tickers", "tickers", tickers)
}
