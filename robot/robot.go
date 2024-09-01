package robot

import (
	"mexc-bot/client"
	"mexc-bot/client/mexc"
	"sync"
)

type BalancesMap = map[string]client.Balance
type TickersMap = map[string]client.OrderBookTicker
type OrdersMap = map[string]client.OrderUpdate

type RobotContext struct {
	balances BalancesMap
	tickers  TickersMap
	orders   OrdersMap
}

type Robot struct {
	m              *mexc.MexcClient
	balances       BalancesMap
	balancesMu     sync.RWMutex
	BalancesStream chan BalancesMap

	tickers       TickersMap
	TickersMu     sync.RWMutex
	TickersStream chan TickersMap

	Orders       OrdersMap
	OrdersMu     sync.RWMutex
	OrdersStream chan OrdersMap

	DealsStream chan *client.Deal
	// ContextStream chan RobotContext
}

func NewRobot(m *mexc.MexcClient) *Robot {
	return &Robot{
		m:              m,
		balances:       make(BalancesMap),
		tickers:        make(TickersMap),
		BalancesStream: make(chan BalancesMap, 1024),
		TickersStream:  make(chan TickersMap, 1024),
		OrdersStream:   make(chan OrdersMap, 1024),
		DealsStream:    make(chan *client.Deal, 1024),
		Orders:         make(OrdersMap),
		// ContextStream: make(chan RobotContext),
	}
}

// type CycleOptions struct {
// 	isSell        bool
// 	usdcBalance   float64
// 	stethQuantity float64
// }

func (r *Robot) WithTickers(do func(tickers TickersMap) any) any {
	r.TickersMu.RLock()
	defer r.TickersMu.RUnlock()
	return do(r.tickers)
}

func (r *Robot) WithBalance(do func(balance BalancesMap) any) any {
	r.balancesMu.RLock()
	defer r.balancesMu.RUnlock()
	return do(r.balances)
}

func (r *Robot) Init() error {
	r.m.Start()
	r.startListenAccountUpdates()
	r.startListenTickers()
	r.startListenOrderUpdates()
	r.startListenDeals()
	return nil
}

func (r *Robot) startListenDeals() {
	go func() {
		for deal := range r.m.DealStream {
			r.DealsStream <- deal
		}
	}()
}

func (r *Robot) startListenOrderUpdates() {
	go func() {
		for order := range r.m.OrderUpdateStream {
			r.OrdersMu.Lock()
			if order.RemainQuantity == 0 {
				delete(r.Orders, order.OrderId)
			} else {
				r.Orders[order.OrderId] = *order
			}

			r.OrdersMu.Unlock()
			r.OrdersStream <- r.Orders
		}
	}()
}

func (r *Robot) startListenAccountUpdates() error {
	balances, err := r.m.Balances()
	if err != nil {
		return err
	}
	r.balances = balances
	r.BalancesStream <- balances
	go func() {
		for update := range r.m.BalanceStream {
			r.balancesMu.Lock()
			r.balances[update.Asset] = *update
			r.balancesMu.Unlock()
			r.BalancesStream <- r.balances
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

	r.tickers[client.ETHUSDC] = *ethusdc
	r.tickers[client.STETHUSDC] = *stethusdc

	go func() {
		for ticker := range r.m.TickersStream {
			r.TickersMu.Lock()
			r.tickers[ticker.Symbol] = *ticker
			r.TickersMu.Unlock()
			r.TickersStream <- r.tickers
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
