package robot

import (
	"log/slog"
	"mexc-bot/client"
	"mexc-bot/client/mexc"
)

type Robot struct {
	m            *mexc.MexcClient
	Balances     *MuMap[client.Balance]
	Tickers      *MuMap[client.OrderBookTicker]
	Orders       *MuMap[client.OrderUpdate]
	Deals        *MuMap[client.Deal]
	PartialDepth *MuMap[client.PartialDepth]
}

func NewRobot(m *mexc.MexcClient) *Robot {
	return &Robot{
		m:            m,
		Balances:     NewMuMap[client.Balance](),
		Tickers:      NewMuMap[client.OrderBookTicker](),
		Orders:       NewMuMap[client.OrderUpdate](),
		Deals:        NewMuMap[client.Deal](),
		PartialDepth: NewMuMap[client.PartialDepth](),
	}
}

func (r *Robot) Init() error {
	r.m.Start()
	r.startListenAccountUpdates()
	r.startListenTickers()
	r.startListenOrderUpdates()
	r.startListenDeals()
	r.startPartialDepthUpdates()
	return nil
}

func (r *Robot) startListenDeals() {
	go func() {
		for deal := range r.m.DealStream {
			r.Deals.Set(deal.TradeId, *deal)
		}
	}()
}

func (r *Robot) startPartialDepthUpdates() {
	go func() {
		for depth := range r.m.PartialDepthStream {
			r.PartialDepth.Set(depth.Symbol, *depth)
		}
	}()
}

func (r *Robot) startListenOrderUpdates() {
	go func() {
		for order := range r.m.OrderUpdateStream {
			if order.RemainQuantity == 0 {
				r.Orders.Delete(order.Id)
			} else {
				r.Orders.Set(order.Id, *order)
			}
		}
	}()
}

func (r *Robot) startListenAccountUpdates() error {
	balances, err := r.m.Balances()
	slog.Info("[ROBOT] prefetched balances", "balances", balances)
	if err != nil {
		return err
	}
	for asset, balance := range balances {
		r.Balances.Set(asset, balance)
	}
	go func() {
		for balance := range r.m.BalanceStream {
			r.Balances.Set(balance.Asset, *balance)
		}
	}()
	return nil
}

func (r *Robot) startListenTickers() {
	ethusdc, err := r.m.OrderBookTicker(client.ETHUSDC)
	if err != nil {
		panic("couldn't get order book ticker")
	}
	stethusdc, err := r.m.OrderBookTicker(client.ETHUSDC)
	if err != nil {
		panic("couldn't get order book ticker")
	}

	r.Tickers.Set(client.ETHUSDC, *ethusdc)
	r.Tickers.Set(client.STETHUSDC, *stethusdc)

	go func() {
		for ticker := range r.m.TickersStream {
			r.Tickers.Set(ticker.Symbol, *ticker)
		}
	}()
}

// func (r *Robot) DoCycle(opts *CycleOptions) {
// 	baseSide := client.SideBuy
// 	if opts.isSell {
// 		baseSide = client.SideSell
// 	}

// 	go func() {
// 		var lastOrder *client.Order
// 		for {
// 			select {

// 			default:
// 				if r.m.EthTicker == nil || r.m.StethTicker == nil {
// 					continue
// 				}
// 				ethPrice := r.m.EthTicker.AskPrice
// 				stethPrice := r.m.StethTicker.AskPrice
// 				if !opts.isSell {
// 					ethPrice = r.m.EthTicker.BidPrice
// 					stethPrice = r.m.StethTicker.BidPrice
// 				}
// 				price := getPrice(ethPrice, stethPrice)
// 				// log.Printf("[ROBOT] OBTAINED TICKERS ETHUSDC=%f STETHUSDC=%f CALCED_PRICE=%f", ethAskPrice, stethAskPrice, price)
// 				if lastOrder != nil && price == lastOrder.Price {
// 					// log.Println("[ROBOT] SAME PRICE. CONTINUE...")
// 					continue
// 				}
// 				log.Printf("[ROBOT] NEW PRICE CHOSEN: %f.", price)
// 				if lastOrder != nil {
// 					log.Printf("[ROBOT] PREVIOUS ORDER FOUND. CANCELLING ORDER %s...", lastOrder.Id)
// 					for {
// 						err := r.m.CancelOrder(client.ETHUSDC, lastOrder.Id)
// 						if err == nil {
// 							break
// 						}
// 						log.Println("[ROBOT] CANCEL ORDER ERROR: ", err)
// 					}
// 					log.Println("[ROBOT] ORDER CANCELLED: ", lastOrder.Id)
// 				}
// 				order := client.Order{
// 					Symbol:  client.STETHUSDC,
// 					Side:    baseSide,
// 					Type:    client.OrderLimit,
// 					Price:   price,
// 					OrigQty: r.m.StethBalance,
// 				}
// 				time.Sleep(time.Millisecond * 50)

// 				for {
// 					err := r.m.PlaceOrder(&order)
// 					if err == nil {
// 						break
// 					}
// 					log.Println("[ROBOT] FAILED TO PLACE NEW ORDER: ", err)
// 				}
// 				log.Printf("[ROBOT] NEW ORDER PLACED: %+v. WAITING FOR THE DEAL...\n", order)
// 				lastOrder = &order
// 			}
// 		}
// 	}()
// }

// func (r *Robot) ListenStethDeals() {
// 	for d := range r.m.DealStream {
// 		if d.Symbol != client.STETHUSDC {
// 			continue
// 		}
// 		log.Printf("[STETH DEALS LISTENER] STETH DEAL OCCURED: %+v. MAKING BACK-ORDER...", d)
// 		go func(d *client.Deal) {
// 			price := 10000.0
// 			side := client.SideBuy
// 			if d.TradeType == client.TradeTypeBuy {
// 				price = 1.0
// 				side = client.SideSell
// 			}
// 			order := client.Order{
// 				Symbol:  client.ETHUSDC,
// 				Side:    side,
// 				Type:    client.OrderLimit,
// 				Price:   price,
// 				OrigQty: d.Quantity,
// 			}
// 			log.Printf("[STETH DEALS LISTENER] ORDER PREPARED: %+v\n", order)
// 			for {
// 				err := r.m.PlaceOrder(&order)
// 				if err != nil {
// 					log.Println("[STETH DEALS LISTENER] FAILED TO PLACE ORDER. RETRYING...", err)
// 				} else {
// 					log.Printf("[STETH DEALS LISTENER] ORDER PLACED: %+v\n", order)
// 					break
// 				}
// 			}
// 		}(d)
// 	}
// }

// func getPrice(ethPrice float64, stethPrice float64) float64 {
// 	return ethPrice + 20
// 	// return math.Max(ethTicker.AskPrice+20.0, stethTicker.AskPrice-0.01)
// }

// func getBaseOrder(side client.OrderSide, price float64, isSell bool) client.Order {
// 	return client.Order{
// 		Symbol:  client.STETHUSDC,
// 		Side:    side,
// 		Type:    client.OrderLimit,
// 		Price:   price,
// 		OrigQty: BASE_QUANTITY,
// 	}
// }
