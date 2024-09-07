package main

import (
	"automata/client/payeer"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/shopspring/decimal"
)

func printOrders(orders []payeer.OrdersOrder) {
	for _, order := range orders {
		fmt.Printf("%12s | %11s | %s\n", order.Price, order.Amount, order.Value)
	}
}

func printOrdersWithHeroPrice(orders []payeer.OrdersOrder, isDesc bool, heroPrice string) string {
	orders = append(orders, payeer.OrdersOrder{Price: heroPrice, Value: "0"})
	slices.SortFunc(orders, func(o1 payeer.OrdersOrder, o2 payeer.OrdersOrder) int {
		price1 := decimal.RequireFromString(o1.Price)
		price2 := decimal.RequireFromString(o2.Price)
		if price1.Equal(price2) {
			return 0
		}
		pos := price1.GreaterThan(price2)
		if pos {
			if isDesc {
				return -1
			} else {
				return 1
			}
		} else {
			if isDesc {
				return 1
			} else {
				return -1
			}
		}
	})
	sb := strings.Builder{}
	sb.WriteString("\n")
	for i, order := range orders {
		if i == TAKE {
			break
		}
		if decimal.RequireFromString(order.Value).Equal(decimal.Zero) {
			yellow := color.New(color.FgYellow).SprintFunc()
			sb.WriteString(yellow(fmt.Sprintf("%12s | %11s | %s\n", order.Price, order.Amount, order.Value)))
		} else {
			sb.WriteString(fmt.Sprintf("%12s | %11s | %s\n", order.Price, order.Amount, order.Value))
		}
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
