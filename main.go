package main

import (
	"log/slog"
	"math"
	"mexc-bot/client"
	"mexc-bot/client/mexc"
	"mexc-bot/robot"
	"os"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	apiKey, ok := os.LookupEnv("API_KEY")
	if !ok {
		panic("API_KEY environment variable not found")
	}

	secretKey, ok := os.LookupEnv("SECRET")
	if !ok {
		panic("SECRET_KEY environment variable not found")
	}

	mexcClient := mexc.NewMexcClient(apiKey, secretKey)
	rob := robot.NewRobot(mexcClient)
	rob.Init()

	// go func() {
	// 	for deal := range rob.DealsStream {
	// 		slog.Info("[ROBOT] DEAL UPDATE", "deal", deal)
	// 	}
	// }()

	go func() {
		for balance := range rob.BalancesStream {
			slog.Info("[ROBOT] BALANCE UPDATE", "balance", balance)
			for tickers := range rob.TickersStream {
				rob.OrdersMu.Lock()
				orders := rob.Orders
				ordersId := []string{}
				for id, order := range orders {
					if order.TradeType == client.TradeTypeSell && order.Symbol == client.STETHUSDC && order.Status == client.OrderStatusNew {
						ordersId = append(ordersId, id)
					}
				}
				rob.OrdersMu.Unlock()
				if len(ordersId) == 0 {
					if balance["STETH"].Free >= 0.0011 {
						go func() {
							rob.TickersMu.RLock()
							ethAskPrice := tickers[client.ETHUSDC].AskPrice
							stethAskPrice := tickers[client.STETHUSDC].AskPrice
							rob.TickersMu.RUnlock()
							if ethAskPrice > 0 && stethAskPrice > 0 {
								price := math.Max(ethAskPrice+20, stethAskPrice-0.01)
								order := &client.Order{
									Symbol:  client.STETHUSDC,
									Price:   price,
									Type:    client.LimitOrderType,
									Side:    client.SellOrderSide,
									OrigQty: balance["STETH"].Free,
								}
								slog.Info("[ROBOT] PREPARED ORDER: ", "eth ask price", ethAskPrice, "steth ask price", stethAskPrice, "order", order)
								mexcClient.PlaceOrder(order)
							}
						}()
					}
				} else {
					for _, id := range ordersId {
						go func(orderId string) {
							err := mexcClient.CancelOrder(client.STETHUSDC, orderId)
							if err != nil {
								rob.OrdersMu.Lock()
								delete(rob.Orders, orderId)
								rob.OrdersMu.Unlock()
							}
						}(id)
					}
				}
			}

			// if balance["USDC"].Free >= 5 {
			// 	go func() {
			// 		for tickers := range rob.TickersStream {
			// 			rob.TickersMu.RLock()
			// 			ethBidPrice := tickers[client.ETHUSDC].BidPrice
			// 			stethBidPrice := tickers[client.STETHUSDC].BidPrice
			// 			rob.TickersMu.RUnlock()
			// 			if ethBidPrice > 0 && stethBidPrice > 0 {
			// 				price := math.Min(ethBidPrice-20, stethBidPrice+0.01)
			// 				quantity := balance["USDC"].Free / price
			// 				order := &client.Order{
			// 					Symbol:  client.STETHUSDC,
			// 					Price:   price,
			// 					Type:    client.LimitOrderType,
			// 					Side:    client.BuyOrderSide,
			// 					OrigQty: quantity,
			// 				}
			// 				slog.Info("[ROBOT] PREPARED ORDER: ", "eth bid price", ethBidPrice, "steth bid price", stethBidPrice, "order", order)
			// 				mexcClient.PlaceOrder(order)
			// 				break
			// 			}
			// 		}
			// 	}()
			// }
		}
	}()

	// go func() {
	// 	for orders := range rob.OrdersStream {
	// 		slog.Info("[ROBOT] ORDERS UPDATE", "orders", orders)
	// 	}
	// }()

	// go func() {
	// 	for r := range mexcClient.TickersStream {
	// 		log.Printf("[ROBOT] TICKER UPDATE: %+v\n", r)
	// 	}
	// }()

	// time.Sleep(time.Second * 3)

	// order := &client.Order{
	// 	Symbol:  "BTCUSDC",
	// 	Type:    client.LimitOrderType,
	// 	Side:    client.SellOrderSide,
	// 	OrigQty: 0.00011,
	// 	Price:   58000,
	// }
	// mexcClient.PlaceOrder(order)
	// time.Sleep(time.Second * 2)
	// order = &client.Order{
	// 	Symbol:  "BTCUSDC",
	// 	Type:    client.LimitOrderType,
	// 	Side:    client.BuyOrderSide,
	// 	OrigQty: 0.00011,
	// 	Price:   100000,
	// }
	// mexcClient.PlaceOrder(order)

	// RunTrading(mexcClient)
	select {}
}

// func RunTrading(mexcClient *mexc.MexcClient) {
// 	balances, err := mexcClient.Balances()
// 	if err != nil {
// 		log.Fatalln("[ROBOT] FAILED TO FETCH BALANCES. ", err)
// 	}
// 	log.Printf("[ROBOT] BALANCES: %+v\n", balances)

// 	if balances["ETH"].Free < BASE_QUANTITY && balances["STETH"].Free < BASE_QUANTITY {
// 		log.Fatalln("[ROBOT] NO ENOUGH ETH AND/OR STETH BALANCE. ABORTING...")
// 	}

// 	getSymbolPrice := func(o *client.OrderBookTicker, isSell bool) float64 {
// 		if isSell {
// 			return o.AskPrice
// 		} else {
// 			return o.BidPrice
// 		}
// 	}

// 	isSell := true
// 	if balances["STETH"].Free >= BASE_QUANTITY {
// 		log.Printf("[ROBOT] STETH BALANCE (%f) > BASE_QUANTITY (%f). FETCHING TICKERS...", balances["STETH"].Free, BASE_QUANTITY)
// 		isSell = false
// 	} else {
// 		log.Printf("[ROBOT] ETH BALANCE (%f) > BASE_QUANTITY (%f). FETCHING TICKERS...", balances["ETH"].Free, BASE_QUANTITY)
// 	}

// }

func getPrice(ethPrice float64, stethPrice float64) float64 {
	return ethPrice + 20
	// return math.Max(ethTicker.AskPrice+20.0, stethTicker.AskPrice-0.01)
}
