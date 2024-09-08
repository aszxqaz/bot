package main

import (
	"automata/client/binance"
	"log/slog"
	"time"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	binanceClient := binance.NewClient()
	// binanceClient.Start()
	// tickers, err := binanceClient.GetOrderBookTickers([]binance.Symbol{
	// 	binance.SYMBOL_BTCUSDT,
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// slog.Info("Tickers", "tickers", tickers)
	orders := binanceClient.SubscribeTicker(binance.SYMBOL_BTCUSDT, time.Millisecond*500)
	for order := range orders {
		slog.Info("OrderBookTickerStreamResult", "result", order)
	}
}
