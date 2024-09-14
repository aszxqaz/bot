package main

import (
	"automata/client/binance"
	"automata/client/payeer"
	"automata/msync"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

type ValueOffsetStrategyOptions struct {
	// PlacementValueOffset   string
	MaxPriceRatio          string
	ReplacementValueOffset string
	SelectorConfig         *payeer.PayeerPriceSelectorConfig
	Pairs                  map[payeer.Pair]binance.Symbol
	BinanceTickerInterval  time.Duration
	BuyEnabled             bool
	SellEnabled            bool
	Amount                 decimal.Decimal
}

type constansts struct {
	// placementValueOffset   decimal.Decimal
	replacementValueOffset decimal.Decimal
	maxPriceRatio          decimal.Decimal
}

type placedMetadata struct {
	binancePrice decimal.Decimal
	action       payeer.Action
}

type store struct {
	orders             *msync.MuMap[int, payeer.OrderParams]
	times              *msync.MuMap[int, time.Time]
	binancePricePlaced *msync.MuMap[int, placedMetadata]
	minWeights         *msync.Mu[int]
	weightsTimestamp   *msync.Mu[time.Time]
	binanceTickers     *msync.MuMap[binance.Symbol, binance.OrderBookTickerStreamResult]
	wait               *msync.MuMap[payeer.Pair, bool]
	balance            *msync.MuMap[string, payeer.Balance]
	info               *payeer.InfoResponse
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
	replacementValueOffset, err := decimal.NewFromString(options.ReplacementValueOffset)
	if err != nil {
		panic(err)
	}
	binanceTickers := msync.NewMuMap[binance.Symbol, binance.OrderBookTickerStreamResult]()
	for _, symbol := range options.Pairs {
		ch := binanceClient.SubscribeTicker(symbol, options.BinanceTickerInterval)
		go func() {
			for ticker := range ch {
				slog.Debug("setting store binance tickers", "symbol", symbol, "ticker", ticker)
				binanceTickers.Set(symbol, ticker)
			}
		}()
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
			binanceTickers:     binanceTickers,
			wait:               msync.NewMuMap[payeer.Pair, bool](),
			balance:            msync.NewMuMap[string, payeer.Balance](),
		},
		selector: payeer.NewPayeerPriceSelector(
			options.SelectorConfig,
			binanceTickers,
		),
	}
}

func (s *ValueOffsetStrategy) Run() {
	s.cancelInitialOrders()
	s.resetBalance()
	s.resetInfo()
	for pair := range s.options.Pairs {
		if s.options.BuyEnabled {
			go s.PlaceOrderLoop(payeer.ACTION_BUY, pair)
		}
		if s.options.SellEnabled {
			go s.PlaceOrderLoop(payeer.ACTION_SELL, pair)
		}
		if s.options.SellEnabled || s.options.BuyEnabled {
			go s.CheckAndCancelLoop(pair)
			go s.OrdersUpdateLoop()
		}
	}
	select {}
}

func (s *ValueOffsetStrategy) OrdersUpdateLoop() {
	for {
		time.Sleep(time.Second * 5)
		slog.Info("[ValueOffsetStrategy] checking orders inner list", "orderIds", s.orders.Keys())
		orderIdsToDelete := []int{}
		s.orders.Range(func(orderId int, details payeer.OrderParams) bool {
			order := s.fetchOrderDetails(orderId)
			if decimal.RequireFromString(order.ValueRemaining).IsZero() {
				orderId, _ := strconv.Atoi(order.Id)
				orderIdsToDelete = append(orderIdsToDelete, orderId)
			}
			slog.Info("[ValueOffsetStrategy] order details", "order", *order)
			return true
		})
		for _, id := range orderIdsToDelete {
			s.orders.Delete(id)
			s.times.Delete(id)
			s.binancePricePlaced.Delete(id)
		}
	}
}

func (s *ValueOffsetStrategy) PlaceOrderLoop(action payeer.Action, pair payeer.Pair) {
	for {
		time.Sleep(time.Second * 2)
		if shouldWait, ok := s.wait.Get(pair); ok {
			if shouldWait {
				continue
			}
		}
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
		time.Sleep(500 * time.Millisecond)
		orders := s.fetchOrders(pair)
		ok, price := s.selector.SelectPrice(action, &orders)
		if ok {
			if action == payeer.ACTION_BUY {
				quote, ok := s.balance.Get(pair.Quote())
				if !ok {
					slog.Warn("[ValueOffsetStrategy] no balance found for", "quote", pair.Quote())
					continue
				}
				available := decimal.NewFromFloat(quote.Available)
				required := price.Mul(s.options.Amount)
				if available.LessThan(required) {
					slog.Warn("[ValueOffsetStrategy] not enough quote", "action", action, "quote", pair.Quote(), "required", required.String(), "available", available.String())
					continue
				}
			} else {
				base, ok := s.balance.Get(pair.Base())
				if !ok {
					slog.Warn("[ValueOffsetStrategy] no balance found for", "base", pair.Base())
					continue
				}
				available := decimal.NewFromFloat(base.Available)
				required := s.options.Amount
				if available.LessThan(required) {
					slog.Warn("[ValueOffsetStrategy] not enough base", "action", action, "base", pair.Base(), "required", required.String(), "available", available.String())
					continue
				}
			}
			binancePrices, ok := s.binanceTickers.Get(s.options.Pairs[pair])
			if !ok {
				slog.Warn("[ValueOffsetStrategy] no binance ticker found", "symbol", s.options.Pairs[pair])
				continue
			}
			rsp := s.placeOrder(action, pair, s.options.Amount.String(), price.String())
			var binancePrice decimal.Decimal
			if action == payeer.ACTION_SELL {
				binancePrice = decimal.RequireFromString(binancePrices.AskPrice)
			} else {
				binancePrice = decimal.RequireFromString(binancePrices.BidPrice)
			}
			s.binancePricePlaced.Set(rsp.OrderId, placedMetadata{
				binancePrice: binancePrice,
				action:       action,
			})
			s.resetBalance()
		}
	}
}

func (s *ValueOffsetStrategy) CheckAndCancelLoop(pair payeer.Pair) {
	for {
		if len(s.orders.Keys()) == 0 {
			continue
		}
		binancePrices, ok := s.binanceTickers.Get(s.options.Pairs[pair])
		if !ok {
			slog.Warn("[ValueOffsetStrategy] no binance ticker found", "symbol", s.options.Pairs[pair])
			continue
		}
		time.Sleep(500 * time.Millisecond)
		orders := s.fetchOrders(pair)
		priceChangedOrderIds := []int{}
		// s.binancePricePlaced.Range(func(key int, data placedMetadata) bool {
		// 	t, ok := s.times.Get(key)
		// 	if !ok {
		// 		panic("order time not found")
		// 	}
		// 	if time.Since(t).Minutes() < 1 {
		// 		return true
		// 	}
		// 	var violatesRatio bool
		// 	if data.action == payeer.ACTION_SELL {
		// 		binanceAskPrice := decimal.RequireFromString(binancePrices.AskPrice)
		// 		violatesRatio = binanceAskPrice.Div(data.binancePrice).GreaterThan(s.maxPriceRatio)
		// 	} else {
		// 		binanceBidPrice := decimal.RequireFromString(binancePrices.BidPrice)
		// 		violatesRatio = data.binancePrice.Div(binanceBidPrice).GreaterThan(s.maxPriceRatio)
		// 	}
		// 	if violatesRatio {
		// 		priceChangedOrderIds = append(priceChangedOrderIds, key)
		// 		s.wait.Set(pair, true)
		// 		time.AfterFunc(time.Minute, func() {
		// 			s.wait.Delete(pair)
		// 		})
		// 		return true
		// 	}
		// 	return true
		// })
		// s.cancelOrders(priceChangedOrderIds)
		cancelableOrderIds := []int{}
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
			// cancel by value offset
			if s.getTopValueOffset(price, orders, value.Action).GreaterThan(s.replacementValueOffset) {
				slog.Info("[ValueOffsetStrategy] order should be replaced due to top value offset", "orderId", key)
				cancelableOrderIds = append(cancelableOrderIds, key)
				return true
			}
			// cancel by binance price
			if value.Action == payeer.ACTION_BUY {
				binPrice := decimal.RequireFromString(binancePrices.BidPrice)
				ok := price.Div(binPrice).LessThan(s.selector.Config.BidMaxBinancePriceRatio)
				if !ok {
					slog.Info("[PayeerPriceSelector] canceled by binance price", "orderId", key, "action", value.Action, "ok", ok, "binance bid price", binPrice.String(), "price", price.String())
					cancelableOrderIds = append(cancelableOrderIds, key)
					return true
				}
			} else {
				binPrice := decimal.RequireFromString(binancePrices.AskPrice)
				ok := price.Div(binPrice).GreaterThan(s.selector.Config.AskMinBinancePriceRatio)
				if !ok {
					slog.Info("[PayeerPriceSelector] canceled by binance price", "orderId", key, "action", value.Action, "ok", ok, "binance ask price", binPrice.String(), "price", price.String())
					cancelableOrderIds = append(cancelableOrderIds, key)
					return true
				}
			}
			return true
		})
		s.cancelOrders(cancelableOrderIds)
		if len(priceChangedOrderIds) > 0 || len(cancelableOrderIds) > 0 {
			s.resetBalance()
		}
	}
}

func (s *ValueOffsetStrategy) cancelInitialOrders() {
	orders := s.fetchMyOrders()
	slog.Info("[ValueOffsetStrategy] Cancelling pending orders...")
	for _, order := range orders {
		orderTime := time.Unix(order.Date, 0)
		diff := time.Since(orderTime)
		if diff.Minutes() <= 1 {
			wait := time.Minute - diff
			slog.Info("[ValueOffsetStrategy] Init should wait for order cancel", "orderId", order.Id, "time", wait)
			time.Sleep(wait)
		}
		orderId, _ := strconv.Atoi(order.Id)
		s.cancelOrder(orderId)
	}
}

func (s *ValueOffsetStrategy) resetBalance() {
	slog.Info("[ValueOffsetStrategy] Fetching balances...")
	for currency, balance := range s.fetchBalance() {
		if balance.Available > 0 {
			s.balance.Set(currency, balance)
		}
	}
}

func (s *ValueOffsetStrategy) resetInfo() {
	info, err := s.payeerClient.Info()
	if err != nil {
		panic(err)
	}
	if !info.Success {
		slog.Error("[ValueOffsetStrategy] Info response error", "error", info.Error)
		os.Exit(1)
	}
	s.info = info
}

func (s *ValueOffsetStrategy) fetchMyOrders() map[string]payeer.MyOrdersOrder {
	ordersRsp, err := s.payeerClient.MyOrders(&payeer.MyOrdersRequest{})
	if err != nil {
		panic(err)
	}
	s.updateWeights(60)
	if !ordersRsp.Success {
		slog.Error("[ValueOffsetStrategy] MyOrders response error", "error", ordersRsp.Error)
		os.Exit(1)
	}
	return ordersRsp.Orders
}

func (s *ValueOffsetStrategy) fetchOrderDetails(orderId int) *payeer.OrderDetails {
	orderStatusRsp, err := s.payeerClient.OrderStatus(&payeer.OrderStatusRequest{OrderId: orderId})
	if err != nil {
		panic(err)
	}
	s.updateWeights(5)
	if !orderStatusRsp.Success {
		slog.Error("[ValueOffsetStrategy] Order status response error", "error", orderStatusRsp.Error)
		os.Exit(1)
	}
	return &orderStatusRsp.Order
}

func (s *ValueOffsetStrategy) placeOrder(action payeer.Action, pair payeer.Pair, amount string, price string) *payeer.PostOrderResponse {
	rsp, err := s.payeerClient.PlaceOrder(&payeer.PostOrderRequest{
		Pair:   pair,
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
		slog.Error("[ValueOffsetStrategy] Place order response error", "response", rsp)
		os.Exit(1)
	}
	s.times.Set(rsp.OrderId, time.Now())
	s.orders.Set(rsp.OrderId, rsp.Params)
	slog.Info("[ValueOffsetStrategy] Order placed:", "order", rsp)
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
		slog.Error("[ValueOffsetStrategy] Cancel order error", "error", rsp.Error)
		os.Exit(1)
	}
	slog.Info("[ValueOffsetStrategy] Order canceled", "orderId", orderId)
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

func (s *ValueOffsetStrategy) fetchBalance() map[string]payeer.Balance {
	balance, err := s.payeerClient.Balance()
	if err != nil {
		panic(err)
	}
	s.updateWeights(10)
	if !balance.Success {
		slog.Error("[ValueOffsetStrategy] Balance response error", "error", balance.Error)
		os.Exit(1)
	}
	slog.Debug("Payeer balance:", "balance", balance.Balances)
	return balance.Balances
}

func (s *ValueOffsetStrategy) fetchOrders(pair payeer.Pair) payeer.PairsOrderInfo {
	orders, err := s.payeerClient.Orders([]payeer.Pair{pair})
	if err != nil {
		panic(err)
	}
	s.updateWeights(1)
	if !orders.Success {
		slog.Error("[ValueOffsetStrategy] Orders response error", "error", orders.Error)
		os.Exit(1)
	}
	return orders.Pairs[pair]
}

type DecimalPrices struct {
	Bid decimal.Decimal
	Ask decimal.Decimal
}

// func (s *ValueOffsetStrategy) fetchBinancePrice() *DecimalPrices {
// 	rsp, err := s.binanceClient.GetOrderBookTickers([]binance.Symbol{binance.SYMBOL_BTCUSDT})
// 	if err != nil {
// 		panic(err)
// 	}
// 	tickers := rsp.Result[0]
// 	slog.Debug("Binance data:", "tickers", tickers)
// 	bid, err := decimal.NewFromString(tickers.BidPrice)
// 	if err != nil {
// 		panic(err)
// 	}
// 	ask, err := decimal.NewFromString(tickers.AskPrice)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return &DecimalPrices{Bid: bid, Ask: ask}
// }

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

// if action == payeer.ACTION_SELL {
// 	balance, ok := s.balance.Get(pair.Quote())
// 	if !ok {
// 		slog.Warn("[ValueOffsetStrategy] no balance found for", "quote", pair.Quote())
// 		continue
// 	}
// 	fee := decimal.
// 		NewFromFloat(s.info.Pairs[pair].FeeTakerPercent).
// 		Div(decimal.NewFromInt(100)).
// 		Add(decimal.NewFromInt(1)).
// 		Mul(s.options.Amount).
// 		Mul(price)
// 	total := s.options.Amount.Mul(price).Add(fee)
// 	if ok && decimal.NewFromFloat(balance.Available).LessThan(total) {
// 		slog.Warn("[ValueOffsetStrategy] insufficient funds", "quote", pair.Quote(), "available", balance.Available, "required", total.String())
// 		continue
// 	}
// }
// if action == payeer.ACTION_BUY {
// quoteBalance, ok := s.balance.Get(pair.Quote())
// if !ok {
// 	slog.Warn("[ValueOffsetStrategy] no balance found for", "quote", pair.Quote())
// 	continue
// }
// 	fee := decimal.
// 		NewFromFloat(s.info.Pairs[pair].FeeTakerPercent).
// 		Div(decimal.NewFromInt(100)).
// 		Add(decimal.NewFromInt(1)).
// 		Mul(s.options.Amount)
// 	required := s.options.Amount.Add(fee).Mul(price)
// 	if decimal.NewFromFloat(quoteBalance.Available).LessThan(required) {
// 		slog.Warn("[ValueOffsetStrategy] insufficient funds", "quote", pair.Quote(), "available", quoteBalance.Available, "required", required.String())
// 		continue
// 	}
// }
