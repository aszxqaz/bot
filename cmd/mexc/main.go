package main

import (
	"automata/client"
	"automata/client/mexc"
	"automata/robot"
	"log/slog"
	"sync"
)

const PRICE_OFFSET = 5

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	// apiKey, ok := os.LookupEnv("API_KEY")
	// if !ok {
	// 	panic("API_KEY environment variable not found")
	// }

	// secretKey, ok := os.LookupEnv("SECRET")
	// if !ok {
	// 	panic("SECRET_KEY environment variable not found")
	// }

	// mexcClient := mexc.NewMexcClient(apiKey, secretKey)
	// store := robot.NewRobot(mexcClient)
	// store.Init()

	// go SellLoop(store, mexcClient)
	// go BuyLoop(store, mexcClient)
	// select {}

	select {}
}

func SellLoop(store *robot.Robot, mexcClient *mexc.Client) {
	lastPrice := 0.0
	for {
		depth, ok := store.PartialDepth.Get(client.STETHUSDC)
		if !ok {
			continue
		}
		ethTicker, ok := store.Tickers.Get(client.ETHUSDC)
		if !ok || ethTicker.AskPrice == 0 {
			continue
		}
		// stethTicker, ok := store.Tickers.Get(client.STETHUSDC)
		// if !ok || stethTicker.AskPrice == 0 {
		// 	continue
		// }
		price := getAskPrice(ethTicker.AskPrice, 0, depth.Asks)
		if price == lastPrice {
			continue
		}
		cancelableOrderIds := []string{}
		store.Orders.Range(func(key string, order client.OrderUpdate) bool {
			if order.Status == client.OrderStatusNew &&
				order.TradeType == client.TradeTypeSell &&
				order.Symbol == client.STETHUSDC {
				cancelableOrderIds = append(cancelableOrderIds, order.Id)
			}
			return true
		})
		var wg sync.WaitGroup
		for _, id := range cancelableOrderIds {
			wg.Add(1)
			go func(id string) {
				err := mexcClient.CancelOrder(client.STETHUSDC, id)
				if err == nil {
					store.Orders.Delete(id)
					lastPrice = 0
				}
				wg.Done()
			}(id)
		}
		wg.Wait()
		balance, ok := store.Balances.Get("STETH")
		if !ok {
			continue
		}
		if balance.Free >= 0.0011 {
			depth, _ := store.PartialDepth.Get(client.STETHUSDC)
			ethTicker, _ := store.Tickers.Get(client.ETHUSDC)
			// stethTicker, ok := store.Tickers.Get(client.STETHUSDC)
			// if !ok || stethTicker.AskPrice == 0 {
			// 	continue
			// }
			price := getAskPrice(ethTicker.AskPrice, 0, depth.Asks)
			order := &client.Order{Type: client.LimitOrderType, Side: client.SellOrderSide, Symbol: client.STETHUSDC, Price: price, OrigQty: balance.Free}
			err := mexcClient.PlaceOrder(order)
			if err == nil {
				slog.Info("[ROBOT] Order placed", "order", order, "asks", depth.Asks, "ethAskPrice", ethTicker.AskPrice)
				store.Orders.Set(order.Id, client.OrderUpdate{
					Id:             order.Id,
					Symbol:         order.Symbol,
					Price:          order.Price,
					Status:         client.OrderStatusNew,
					RemainQuantity: order.OrigQty,
					TradeType:      client.TradeTypeSell,
				})
				store.Balances.Set("STETH", client.Balance{Asset: "STETH", Free: 0})
				lastPrice = price
			} else {
				lastPrice = 0
			}
		}
	}
}

func BuyLoop(store *robot.Robot, mexcClient *mexc.Client) {
	lastPrice := 0.0
	for {
		depth, ok := store.PartialDepth.Get(client.STETHUSDC)
		if !ok {
			continue
		}
		ethTicker, ok := store.Tickers.Get(client.ETHUSDC)
		if !ok || ethTicker.BidPrice == 0 {
			continue
		}
		// stethTicker, ok := store.Tickers.Get(client.STETHUSDC)
		// if !ok || stethTicker.BidPrice == 0 {
		// 	continue
		// }
		price := getBidPrice(ethTicker.BidPrice, 0, depth.Bids)
		if price == lastPrice {
			continue
		}
		cancelableOrderIds := []string{}
		store.Orders.Range(func(key string, order client.OrderUpdate) bool {
			if order.Status == client.OrderStatusNew &&
				order.TradeType == client.TradeTypeBuy &&
				order.Symbol == client.STETHUSDC {
				cancelableOrderIds = append(cancelableOrderIds, order.Id)
			}
			return true
		})
		var wg sync.WaitGroup
		for _, id := range cancelableOrderIds {
			wg.Add(1)
			go func(id string) {
				err := mexcClient.CancelOrder(client.STETHUSDC, id)
				if err == nil {
					store.Orders.Delete(id)
					lastPrice = 0
				}
				wg.Done()
			}(id)
		}
		wg.Wait()
		balance, ok := store.Balances.Get("USDC")
		if !ok {
			continue
		}
		if balance.Free >= 5 {
			depth, _ := store.PartialDepth.Get(client.STETHUSDC)
			ethTicker, _ := store.Tickers.Get(client.ETHUSDC)
			// stethTicker, ok := store.Tickers.Get(client.STETHUSDC)
			// if !ok || stethTicker.AskPrice == 0 {
			// 	continue
			// }
			price := getBidPrice(ethTicker.BidPrice, 0, depth.Bids)
			order := &client.Order{
				Type:    client.LimitOrderType,
				Side:    client.BuyOrderSide,
				Symbol:  client.STETHUSDC,
				Price:   price,
				OrigQty: balance.Free / price,
			}
			err := mexcClient.PlaceOrder(order)
			if err == nil {
				slog.Info("[ROBOT] Order placed", "order", order, "bids", depth.Bids, "ethBidPrice", ethTicker.BidPrice)
				store.Orders.Set(order.Id, client.OrderUpdate{
					Id:             order.Id,
					Symbol:         order.Symbol,
					Price:          order.Price,
					Status:         client.OrderStatusNew,
					RemainQuantity: order.OrigQty,
					TradeType:      client.TradeTypeBuy,
				})
				store.Balances.Set("USDC", client.Balance{Asset: "USDC", Free: 0})
				lastPrice = price
			} else {
				lastPrice = 0
			}
		}
	}
}

func getAskPrice(ethPrice float64, stethPrice float64, asks []client.PartialDepthPair) float64 {
	base := ethPrice + PRICE_OFFSET
	for _, a := range asks {
		if a.Price >= base {
			return a.Price - 0.1
		}
	}
	return base
	// return math.Max(ethPrice+PRICE_OFFSET, stethPrice-0.01)
}

func getBidPrice(ethPrice float64, stethPrice float64, bids []client.PartialDepthPair) float64 {
	base := ethPrice - PRICE_OFFSET
	for _, d := range bids {
		if d.Price <= base {
			return d.Price + 0.1
		}
	}
	return base
	// return math.Min(ethPrice-PRICE_OFFSET, stethPrice+0.01)
}
