package main

import (
	"automata/client/payeer"
	"log/slog"

	"github.com/shopspring/decimal"
)

func selectPriceFromPayeerOrders(action payeer.Action, info payeer.PairsOrderInfo, placementValueOffset decimal.Decimal) decimal.Decimal {
	acc := decimal.NewFromInt(0)
	var selectedPrice decimal.Decimal
	orders := info.Bids
	if action == payeer.ACTION_SELL {
		orders = info.Asks
	}
	for _, order := range orders {
		value, _ := decimal.NewFromString(order.Value)
		acc = acc.Add(value)
		if acc.GreaterThanOrEqual(placementValueOffset) {
			p, err := decimal.NewFromString(order.Price)
			if err != nil {
				panic(err)
			}
			cent, _ := decimal.NewFromString(".01")
			if action == payeer.ACTION_SELL {
				selectedPrice = p.Sub(cent)
			} else {
				selectedPrice = p.Add(cent)
			}
			slog.Info("Payeer price chosen:", "price", selectedPrice.String())
			break
		}
	}
	return selectedPrice
}

func getTopValueOffset(action payeer.Action, price decimal.Decimal, orders payeer.PairsOrderInfo) decimal.Decimal {
	acc := decimal.NewFromInt(0)
	prices := orders.Bids
	if action == payeer.ACTION_SELL {
		prices = orders.Asks
	}
	for _, order := range prices {
		orderPrice, err := decimal.NewFromString(order.Price)
		if err != nil {
			panic(err)
		}
		shouldInclude := orderPrice.GreaterThan(price)
		if action == payeer.ACTION_SELL {
			shouldInclude = orderPrice.LessThan(price)
		}
		if shouldInclude {
			orderValue, err := decimal.NewFromString(order.Value)
			if err != nil {
				panic(err)
			}
			acc = acc.Add(orderValue)
		} else {
			break
		}
	}
	return acc
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
