package main

import (
	"automata/client/payeer"
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"
)

const TAKE = 15
const PLACEMENT_VALUE_OFFSET = 5000

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	fmt.Println("PLACEMENT_VALUE_OFFSET = ", PLACEMENT_VALUE_OFFSET)
	client := createClient()
	orders := fetchOrders(client)

	fraction := decimal.RequireFromString("0.00005") // ~ 250 roubles

	selectedBuyPrice := selectPriceFromPayeerOrders(payeer.ACTION_BUY, orders, decimal.NewFromInt(PLACEMENT_VALUE_OFFSET), fraction)
	fmt.Println("bids", printOrders(orders.Bids, true, selectedBuyPrice.String()))

	selectedSellPrice := selectPriceFromPayeerOrders(payeer.ACTION_SELL, orders, decimal.NewFromInt(PLACEMENT_VALUE_OFFSET), fraction)
	fmt.Println("asks", printOrders(orders.Asks, false, selectedSellPrice.String()))

	// Averages
	bidsWma := getWeightedMeanAverage(orders.Bids)
	asksWma := getWeightedMeanAverage(orders.Asks)
	fmt.Printf("Asks WMA: %s, Bids WMA: %s\n", asksWma.StringFixed(2), bidsWma.StringFixed(2))
	avg := asksWma.Add(bidsWma).Div(decimal.NewFromInt(2))
	fmt.Printf("Average: %s\n", avg.StringFixed(2))
}
