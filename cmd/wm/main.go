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
	slog.SetLogLoggerLevel(slog.LevelError)
	client := createClient()
	orders := fetchOrders(client)

	fmt.Println("bids\n", printOrders(orders.Bids))
	fmt.Println("asks\n", printOrders(orders.Asks))

	bidsWma := getWeightedMeanAverage(orders.Bids)
	asksWma := getWeightedMeanAverage(orders.Asks)

	fmt.Printf("Asks WMA: %s, Bids WMA: %s\n", asksWma.StringFixed(2), bidsWma.StringFixed(2))

	avg := asksWma.Add(bidsWma).Div(decimal.NewFromInt(2))

	fmt.Printf("Average: %s\n", avg.StringFixed(2))

	selectedBuyPrice := selectPriceFromPayeerOrders(payeer.ACTION_BUY, orders, decimal.NewFromInt(PLACEMENT_VALUE_OFFSET))
	selectedSellPrice := selectPriceFromPayeerOrders(payeer.ACTION_SELL, orders, decimal.NewFromInt(PLACEMENT_VALUE_OFFSET))

	fmt.Printf("Buy Price: %s, Sell Price: %s\n", selectedBuyPrice.StringFixed(2), selectedSellPrice.StringFixed(2))
}
