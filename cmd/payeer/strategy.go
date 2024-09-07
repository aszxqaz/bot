package main

import (
	"automata/client/binance"
	"automata/client/payeer"
	"automata/msync"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

const AMOUNT = "0.0001"

var amountDecimal, _ = decimal.NewFromString(AMOUNT)

type ValueOffsetStrategyOptions struct {
	MaxPriceRatio string
	// PlacementValueOffset   string
	ReplacementValueOffset string
	SelectorConfig         *payeer.PayeerPriceSelectorConfig
}

type constansts struct {
	// placementValueOffset   decimal.Decimal
	replacementValueOffset decimal.Decimal
	maxPriceRatio          decimal.Decimal
}

type placedMetadata struct {
	binancePrice decimal.Decimal
	isSell       bool
}

type store struct {
	orders             *msync.MuMap[int, payeer.OrderParams]
	times              *msync.MuMap[int, time.Time]
	binancePricePlaced *msync.MuMap[int, placedMetadata]
	minWeights         *msync.Mu[int]
	weightsTimestamp   *msync.Mu[time.Time]
}

type ValueOffsetStrategy struct {
	options       *ValueOffsetStrategyOptions
	binanceClient *binance.Client
	payeerClient  *payeer.Client
	selector      *payeer.PayeerPriceSelector
	constansts
	store
}

func NewVolumeOffsetStrategy(
	payeerClient *payeer.Client,
	binanceClient *binance.Client,
	options *ValueOffsetStrategyOptions,
) *ValueOffsetStrategy {
	maxPriceDelta, err := decimal.NewFromString(options.MaxPriceRatio)
	if err != nil {
		panic(err)
	}
	// valueOffset, err := decimal.NewFromString(options.PlacementValueOffset)
	// if err != nil {
	// 	panic(err)
	// }
	replacementValueOffset, err := decimal.NewFromString(options.ReplacementValueOffset)
	if err != nil {
		panic(err)
	}
	return &ValueOffsetStrategy{
		options:       options,
		binanceClient: binanceClient,
		payeerClient:  payeerClient,
		constansts: constansts{
			// placementValueOffset:   valueOffset,
			replacementValueOffset: replacementValueOffset,
			maxPriceRatio:          maxPriceDelta,
		},
		store: store{
			orders:             msync.NewMuMap[int, payeer.OrderParams](),
			times:              msync.NewMuMap[int, time.Time](),
			binancePricePlaced: msync.NewMuMap[int, placedMetadata](),
			minWeights:         msync.NewMu(600),
			weightsTimestamp:   msync.NewMu(time.Now()),
		},
		selector: payeer.NewPayeerPriceSelector(options.SelectorConfig),
	}
}

func (s *ValueOffsetStrategy) Run() {
	// go s.PlaceOrderLoop(payeer.ACTION_BUY)
	go s.PlaceOrderLoop(payeer.ACTION_SELL)
	go s.CheckAndCancelLoop()
	select {}
}

func (s *ValueOffsetStrategy) PlaceOrderLoop(action payeer.Action) {
	for {
		skip := false
		s.orders.Range(func(key int, value payeer.OrderParams) bool {
			if value.Action == action {
				skip = true
				return false
			}
			return true
		})
		if skip {
			continue
		}
		isSell := false
		if action == payeer.ACTION_SELL {
			isSell = true
			btcAvailable := decimal.NewFromFloat(s.fetchPayeerBalance()["BTC"].Available)
			if btcAvailable.LessThan(amountDecimal) {
				continue
			}
		}
		orders := s.fetchPayeerOrders()
		ok, price := s.selector.SelectPrice(action, &orders)
		if ok {
			binanceResult := s.fetchBinanceBidPrice()
			rsp := s.placeOrder(isSell, AMOUNT, price.String())
			var binancePrice decimal.Decimal
			if isSell {
				binancePrice = binanceResult.Ask
			} else {
				binancePrice = binanceResult.Bid
			}
			s.binancePricePlaced.Set(rsp.OrderId, placedMetadata{
				binancePrice: binancePrice,
				isSell:       isSell,
			})
		}
	}
}

func (s *ValueOffsetStrategy) CheckAndCancelLoop() {
	for {
		if len(s.orders.Keys()) == 0 {
			continue
		}
		binanceCurrentPrice := s.fetchBinanceBidPrice()
		orders := s.fetchPayeerOrders()
		orderIds := []int{}
		s.binancePricePlaced.Range(func(key int, data placedMetadata) bool {
			t, ok := s.times.Get(key)
			if !ok {
				panic("order time not found")
			}
			if time.Since(t).Minutes() < 1 {
				return true
			}
			var violatesRatio bool
			if data.isSell {
				violatesRatio = binanceCurrentPrice.Ask.Div(data.binancePrice).GreaterThan(s.maxPriceRatio)
			} else {
				violatesRatio = data.binancePrice.Div(binanceCurrentPrice.Bid).GreaterThan(s.maxPriceRatio)
			}
			if violatesRatio {
				orderIds = append(orderIds, key)
				return true
			}
			return true
		})
		s.cancelOrders(orderIds)
		orderIds = []int{}
		s.orders.Range(func(key int, value payeer.OrderParams) bool {
			t, ok := s.times.Get(key)
			if !ok {
				panic("order time not found")
			}
			if time.Since(t).Minutes() < 1 {
				return true
			}
			price, err := decimal.NewFromString(value.Price)
			if err != nil {
				panic(err)
			}
			if s.getTopValueOffset(price, orders, value.Action).GreaterThan(s.replacementValueOffset) {
				orderIds = append(orderIds, key)
				return true
			}
			return true
		})
		s.cancelOrders(orderIds)
	}
}

func (s *ValueOffsetStrategy) fetchOrderDetails(orderId int) *payeer.OrderDetails {
	rsp, err := s.payeerClient.OrderStatus(&payeer.OrderStatusRequest{OrderId: orderId})
	if err != nil {
		panic(err)
	}
	if !rsp.Success {
		panic(rsp.Error)
	}
	return &rsp.Order
}

func (s *ValueOffsetStrategy) placeOrder(sell bool, amount string, price string) *payeer.PostOrderResponse {
	action := payeer.ACTION_BUY
	if sell {
		action = payeer.ACTION_SELL
	}
	rsp, err := s.payeerClient.PlaceOrder(&payeer.PostOrderRequest{
		Pair:   payeer.PAIR_BTCRUB,
		Type:   payeer.ORDER_TYPE_LIMIT,
		Action: action,
		Amount: amount,
		Price:  price,
	})
	if err != nil {
		panic(err)
	}
	s.updateWeights(5)
	if !rsp.Success {
		panic(rsp.Error)
	}
	s.times.Set(rsp.OrderId, time.Now())
	s.orders.Set(rsp.OrderId, rsp.Params)
	slog.Info("Order placed:", "order", rsp)
	return rsp
}

func (s *ValueOffsetStrategy) cancelOrders(orderIds []int) {
	for _, orderId := range orderIds {
		s.cancelOrder(orderId)
	}
}

func (s *ValueOffsetStrategy) cancelOrder(orderId int) *payeer.CancelOrderResponse {
	rsp, err := s.payeerClient.CancelOrder(&payeer.CancelOrderRequest{
		OrderId: orderId,
	})
	if err != nil {
		panic(err)
	}
	s.updateWeights(10)
	if !rsp.Success {
		slog.Info("Order not cancelled", "response", rsp)
		if rsp.Error.Code == payeer.ERR_INVALID_STATUS_FOR_REFUND {
			s.times.Delete(orderId)
			s.orders.Delete(orderId)
			s.binancePricePlaced.Delete(orderId)
			return rsp
		}
		panic(string(rsp.Error.Code))
	}
	slog.Info("Order canceled", "orderId", orderId)
	s.times.Delete(orderId)
	s.orders.Delete(orderId)
	s.binancePricePlaced.Delete(orderId)
	return rsp
}

func (s *ValueOffsetStrategy) getTopValueOffset(price decimal.Decimal, orders payeer.PairsOrderInfo, action payeer.Action) decimal.Decimal {
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

// func (s *ValueOffsetStrategy) selectPriceFromPayeerOrders(isSell bool, info payeer.PairsOrderInfo) decimal.Decimal {
// 	acc := decimal.NewFromInt(0)
// 	var selectedPrice decimal.Decimal
// 	orders := info.Bids
// 	if isSell {
// 		orders = info.Asks
// 	}
// 	for _, order := range orders {
// 		value, _ := decimal.NewFromString(order.Value)
// 		acc = acc.Add(value)
// 		if acc.GreaterThanOrEqual(s.placementValueOffset) {
// 			p, err := decimal.NewFromString(order.Price)
// 			if err != nil {
// 				panic(err)
// 			}
// 			cent, _ := decimal.NewFromString(".01")
// 			if isSell {
// 				selectedPrice = p.Sub(cent)
// 			} else {
// 				selectedPrice = p.Add(cent)
// 			}
// 			slog.Info("Payeer price chosen:", "price", selectedPrice.String())
// 			break
// 		}
// 	}
// 	return selectedPrice
// }

func (s *ValueOffsetStrategy) fetchPayeerBalance() map[string]payeer.Balance {
	balance, err := s.payeerClient.Balance()
	if err != nil {
		panic(err)
	}
	s.updateWeights(10)
	if !balance.Success {
		panic(balance.Error)
	}
	slog.Debug("Payeer balance:", "balance", balance.Balances)
	return balance.Balances
}

func (s *ValueOffsetStrategy) fetchPayeerOrders() payeer.PairsOrderInfo {
	orders, err := s.payeerClient.Orders([]payeer.Pair{payeer.PAIR_BTCRUB})
	if err != nil {
		panic(err)
	}
	s.updateWeights(1)
	if !orders.Success {
		panic(orders.Error)
	}
	return orders.Pairs[payeer.PAIR_BTCRUB]
}

type DecimalPrices struct {
	Bid decimal.Decimal
	Ask decimal.Decimal
}

func (s *ValueOffsetStrategy) fetchBinanceBidPrice() *DecimalPrices {
	rsp, err := s.binanceClient.GetOrderBookTickers([]binance.Symbol{binance.SYMBOL_BTCUSDT})
	if err != nil {
		panic(err)
	}
	tickers := rsp.Result[0]
	slog.Debug("Binance data:", "tickers", tickers)
	bid, err := decimal.NewFromString(tickers.BidPrice)
	if err != nil {
		panic(err)
	}
	ask, err := decimal.NewFromString(tickers.AskPrice)
	if err != nil {
		panic(err)
	}

	return &DecimalPrices{Bid: bid, Ask: ask}
}

func (s *ValueOffsetStrategy) updateWeights(count int) {
	now := time.Now()
	if now.Sub(s.weightsTimestamp.Get()).Minutes() > 1 {
		s.weightsTimestamp.Set(now)
		s.minWeights.Set(600 - count)
		return
	} else {
		s.minWeights.Update(func(value int) int {
			return value - count
		})
	}
	slog.Info("Weights info", "remaining/min", s.minWeights.Get())
}

// func (s *ValueOffsetStrategy) waitForWeights(count int) {
// 	for {
// 		if s.minWeights.Get() >= count {
// 			return
// 		}
// 		time.Sleep(time.Millisecond * 10)
// 		slog.Info("Waiting for weights to reach", "count", count)
// 	}
// }

// func (s *ValueOffsetStrategy) getBinanceConvertedBidPrice() decimal.Decimal {
// 	binanceBidPrice := s.fetchBinanceBidPrice()
// 	rsp, err := s.payeerClient.Tickers([]payeer.Pair{payeer.PAIR_USDTRUB})
// 	if err != nil {
// 		panic(err)
// 	}
// 	if !rsp.Success {
// 		panic(rsp.Error)
// 	}
// 	usdtRubBid, _ := decimal.NewFromString(rsp.Pairs[payeer.PAIR_USDTRUB].Bid)
// 	usdtRubAsk, _ := decimal.NewFromString(rsp.Pairs[payeer.PAIR_USDTRUB].Ask)
// 	usdtRubAvg := usdtRubAsk.Add(usdtRubBid).Div(decimal.NewFromInt(2))

// 	slog.Info("Payeer data:", "usdt_rub avg", usdtRubAvg.String())

// 	binanceAfterPrice := usdtRubAvg.Mul(binanceBidPrice)
// 	slog.Info("Binance price after conversion:", "price", binanceAfterPrice.String())

// 	return binanceAfterPrice
// }
