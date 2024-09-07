package main

// func selectPriceFromPayeerOrders(
// 	action payeer.Action,
// 	info payeer.PairsOrderInfo,
// 	placementValueOffset decimal.Decimal,
// 	elevationPriceFraction decimal.Decimal,
// ) decimal.Decimal {
// 	acc := decimal.NewFromInt(0)
// 	var selectedPrice decimal.Decimal
// 	orders := info.Bids
// 	if action == payeer.ACTION_SELL {
// 		orders = info.Asks
// 	}
// 	endIndex := 0
// 	for i, order := range orders {
// 		value, _ := decimal.NewFromString(order.Value)
// 		acc = acc.Add(value)
// 		if acc.GreaterThanOrEqual(placementValueOffset) {
// 			p, err := decimal.NewFromString(order.Price)
// 			if err != nil {
// 				panic(err)
// 			}
// 			if action == payeer.ACTION_SELL {
// 				selectedPrice = p.Sub(cent)
// 			} else {
// 				selectedPrice = p.Add(cent)
// 			}
// 			endIndex = i
// 			break
// 		}
// 	}
// 	afterOffset := selectedPrice.Copy()
// 	fractionAbs := elevationPriceFraction.Mul(selectedPrice)
// 	for i := endIndex - 1; i >= 0; i-- {
// 		price := decimal.RequireFromString(orders[i].Price)
// 		diff := price.Sub(selectedPrice).Abs().Add(cent)
// 		if diff.LessThanOrEqual(fractionAbs) {
// 			if action == payeer.ACTION_SELL {
// 				selectedPrice = selectedPrice.Sub(diff)
// 			} else {
// 				selectedPrice = selectedPrice.Add(diff)
// 			}
// 			fractionAbs = fractionAbs.Sub(diff)
// 			if fractionAbs.LessThanOrEqual(decimal.Zero) {
// 				break
// 			}
// 		}
// 		// if decimal.RequireFromString(orders[i].Price).Equal(selectedPrice) {
// 		// if action == payeer.ACTION_SELL {
// 		// 	selectedPrice = selectedPrice.Sub(cent)
// 		// } else {
// 		// 	selectedPrice = selectedPrice.Add(cent)
// 		// }
// 		// }
// 	}
// 	slog.Debug("Price chosen:", "after offset", afterOffset.String(), "after elevation", selectedPrice.String(), "diff", selectedPrice.Sub(afterOffset).Abs())
// 	return selectedPrice
// }

// func getTopValueOffset(action payeer.Action, price decimal.Decimal, orders payeer.PairsOrderInfo) decimal.Decimal {
// 	acc := decimal.NewFromInt(0)
// 	prices := orders.Bids
// 	if action == payeer.ACTION_SELL {
// 		prices = orders.Asks
// 	}
// 	for _, order := range prices {
// 		orderPrice, err := decimal.NewFromString(order.Price)
// 		if err != nil {
// 			panic(err)
// 		}
// 		shouldInclude := orderPrice.GreaterThan(price)
// 		if action == payeer.ACTION_SELL {
// 			shouldInclude = orderPrice.LessThan(price)
// 		}
// 		if shouldInclude {
// 			orderValue, err := decimal.NewFromString(order.Value)
// 			if err != nil {
// 				panic(err)
// 			}
// 			acc = acc.Add(orderValue)
// 		} else {
// 			break
// 		}
// 	}
// 	return acc
// }
