package main

import (
	"fmt"
	"log/slog"
)

const TAKE = 15
const PLACEMENT_VALUE_OFFSET = 5000

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	fmt.Println("PLACEMENT_VALUE_OFFSET = ", PLACEMENT_VALUE_OFFSET)
	client := createClient()
	orders := fetchOrders(client)

	fmt.Printf("%+v", orders)

	// fmt.Println("Bids")
	// printOrders(orders.Bids)
	// fmt.Println("Asks")
	// printOrders(orders.Asks)

	// selector := payeer.NewPayeerPriceSelector(&payeer.PayeerPriceSelectorConfig{
	// 	PlacementValueOffset:   decimal.NewFromInt(5000),
	// 	ElevationPriceFraction: decimal.RequireFromString(".00005"),
	// 	MaxWmaRatio:            decimal.RequireFromString("1.005"),
	// 	WmaTake:                TAKE,
	// })

	// _, buyPrice := selector.SelectPrice(payeer.ACTION_BUY, &orders)
	// fmt.Println("bids", printOrdersWithHeroPrice(orders.Bids, true, buyPrice.String()))

	// _, sellPrice := selector.SelectPrice(payeer.ACTION_SELL, &orders)
	// fmt.Println("asks", printOrdersWithHeroPrice(orders.Asks, false, sellPrice.String()))
}
