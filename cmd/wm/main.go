package main

import (
	"automata/client/payeer"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/shopspring/decimal"
)

const TAKE = 15

func main() {
	// slog.SetLogLoggerLevel(slog.LevelInfo)
	client := createClient()
	orders := fetchOrders(client)
	fmt.Println("asks\n", printOrders(orders.Asks))
	fmt.Println("bids\n", printOrders(orders.Bids))
	asksWma := getWeightedMeanAverage(orders.Asks)
	bidsWma := getWeightedMeanAverage(orders.Bids)
	fmt.Printf("Asks WMA: %s, Bids WMA: %s\n", asksWma.StringFixed(2), bidsWma.StringFixed(2))
	avg := asksWma.Add(bidsWma).Div(decimal.NewFromInt(2))
	fmt.Printf("Average: %s\n", avg.StringFixed(2))
}

func printOrders(orders []payeer.OrdersOrder) string {
	sb := strings.Builder{}
	for i, order := range orders {
		if i == TAKE {
			break
		}
		sb.WriteString(fmt.Sprintf("price=%s amount=%s value=%s\n", order.Price, order.Amount, order.Value))
	}
	return sb.String()
}

func getWeightedMeanAverage(orders []payeer.OrdersOrder) decimal.Decimal {
	totalValue := decimal.NewFromInt(0)
	totalAmount := decimal.NewFromInt(0)
	for i, order := range orders {
		if i == TAKE {
			break
		}
		value, _ := decimal.NewFromString(order.Value)
		amount, _ := decimal.NewFromString(order.Amount)
		totalValue = totalValue.Add(value)
		totalAmount = totalAmount.Add(amount)
	}
	return totalValue.Div(totalAmount)
}

func fetchOrders(c *payeer.Client) payeer.PairsOrderInfo {
	orders, err := c.Orders([]payeer.Pair{payeer.PAIR_BTCRUB})
	if err != nil {
		panic(err)
	}
	if !orders.Success {
		slog.Error("FetchOrders failed", "error", orders.Error)
		os.Exit(1)
	}
	return orders.Pairs[payeer.PAIR_BTCRUB]
}

func createClient() *payeer.Client {
	return payeer.NewClient(&payeer.Config{})
}
