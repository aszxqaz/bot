package main

import (
	"automata/client/binance"
	"automata/client/payeer"
	"automata/msync"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type BinancePrice struct {
	binance.OrderBookTickerStreamResult
	ts time.Time
}

func main() {
	filename := fmt.Sprintf("%d.csv", time.Now().UnixMilli())
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	slog.Info("Output file created", "name", f.Name())

	pairsAndSymbols := os.Args[1]
	if pairsAndSymbols == "" {
		panic("No pairs specified")
	}
	combined := strings.Split(pairsAndSymbols, ",")
	pairsMap := make(map[payeer.Pair]binance.Symbol, len(combined))
	pairs := make([]payeer.Pair, len(combined))
	for _, str := range combined {
		strs := strings.Split(str, ":")
		pairsMap[payeer.Pair(strs[0])] = binance.Symbol(strs[1])
		pairs = append(pairs, payeer.Pair(strs[0]))
	}

	binancePrices := msync.NewMuMap[payeer.Pair, BinancePrice]()

	slog.Info("Started")

	pc := payeer.NewClient(&payeer.Config{})
	bc := binance.NewClient()

	for pair, symbol := range pairsMap {
		prices := bc.SubscribeTicker(symbol, time.Millisecond*50)
		go func(p payeer.Pair, ch chan binance.OrderBookTickerStreamResult) {
			for price := range ch {
				binancePrices.Set(p, BinancePrice{
					OrderBookTickerStreamResult: price,
					ts:                          time.Now(),
				})
			}
		}(pair, prices)
	}

	tradesMap := make(map[string]struct{})

	startTime := time.Now()
	endTime := time.Now()

	for {
		time.Sleep(200*time.Millisecond - endTime.Sub(startTime))
		startTime = time.Now()
		tradesPairs, err := pc.Trades(pairs)
		if err != nil {
			slog.Error("Failed to fetch trades:", "error", err)
			continue
		}
		for pair, trades := range tradesPairs.Trades {
			binPrice, ok := binancePrices.Get(pair)
			if !ok {
				slog.Info("Binance price not found, skipping")
				continue
			}
			for _, trade := range trades {
				_, ok = tradesMap[trade.Id]
				if ok {
					slog.Info("Trade already processed, skipping all below", "tradeId", trade.Id)
					break
				}
				tradeTime := time.Unix(trade.Date, 0)
				if time.Since(tradeTime) >= time.Second*3 {
					slog.Info("Trade too old, skipping all below", "tradeId", trade.Id, "tradeTime", tradeTime)
					break
				}
				slog.Info("Trade processed", "tradeId", trade.Id, "tradeTime", tradeTime)
				line := fmt.Sprintf("%d,%s,%s,%s,%s\n", trade.Date, trade.Type, trade.Amount, trade.Price, binPrice.AskPrice)
				_, err := f.WriteString(line)
				if err != nil {
					slog.Error("Failed to write to CSV:", "error", err)
				}
				tradesMap[trade.Id] = struct{}{}
			}
		}
		endTime = time.Now()
	}
}
