package main

import (
	"automata/client/payeer"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

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
