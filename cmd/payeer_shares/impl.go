package main

import (
	"automata/client/binance"
	"automata/client/payeer"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

func (s *PayeerSharesStrategy) Run() {
	s.initInfo()
	s.initMyOrders()
	s.initBinanceTickers()
	s.initBalance()
	go s.runOrdersFetchLoop()
	time.Sleep(time.Second)
	for _, share := range s.options.Shares {
		go s.runShareLoop(&share)
	}
	select {}
}

/*
** Loops
 */
func (s *PayeerSharesStrategy) runShareLoop(share *PayeerSharesStrategyShare) {
	init := true
	for {
		if !init {
			time.Sleep(share.LoopInterval)
		} else {
			init = false
		}
		orderCached, ok := s.store.shareOrders.Get(share.ID)
		if !ok {
			order := s.tryPlaceOrder(share)
			if order != nil && order.Success {
				s.store.shareOrders.Set(share.ID, ShareOrderInfo{
					OrderId: order.OrderId,
					Order:   &order.Params,
					Time:    time.Now(),
				})
				time.Sleep(s.options.RefetchBalanceDelay)
				s.updateBalanceByOrderParams(share, &order.Params, true)
			} else if !order.Success {
				if order.Error.Code == payeer.ERR_INSUFFICIENT_FUNDS ||
					order.Error.Code == payeer.ERR_INSUFFICIENT_VOLUME {
					s.initBalance()
					continue
				}
			}
		} else {
			diff := time.Since(orderCached.Time)
			if diff.Minutes() < 1 {
				time.Sleep(time.Minute - diff)
			}

			// Checking if the order has been fulfilled
			orderFetched := s.fetcher.OrderDetails(orderCached.OrderId)
			if decimal.RequireFromString(orderFetched.ValueRemaining).IsZero() {
				s.store.shareOrders.Delete(share.ID)
				s.updateBalanceByOrderDetails(share, orderFetched)
				continue
			}

			// Checking if the price has changed
			if s.hasPriceChanged(share, &orderCached) {
				rsp := s.fetcher.CancelOrder(orderCached.OrderId)
				if rsp.Success {
					s.store.shareOrders.Delete(share.ID)
					orderRefetched := s.fetcher.OrderDetails(orderCached.OrderId)
					s.updateBalanceByOrderDetails(share, orderRefetched)
				} else {
					slog.Error("[Share "+share.ID+"] Cancelling order failed.", "error", rsp.Error)
					continue
				}
			}
		}
	}
}

func (s *PayeerSharesStrategy) updateBalanceByOrderParams(share *PayeerSharesStrategyShare, order *payeer.OrderParams, in bool) {
	slog.Info("[Share "+share.ID+"] Updating balance by order params", "order", order)
	mul := 1.0
	if !in {
		mul = -1.0
	}
	if share.Action == payeer.ACTION_SELL {
		base, _ := s.store.balance.Get(share.Pair.Base())
		amount := decimal.RequireFromString(order.Amount).InexactFloat64()
		base.Available -= amount * mul
		base.Hold += amount * mul
		s.store.balance.Set(share.Pair.Base(), base)
	} else {
		quote, _ := s.store.balance.Get(share.Pair.Quote())
		value := decimal.RequireFromString(order.Value).InexactFloat64()
		quote.Available -= value * mul
		quote.Hold += value * mul
		s.store.balance.Set(share.Pair.Quote(), quote)
	}
}

func (s *PayeerSharesStrategy) updateBalanceByOrderDetails(share *PayeerSharesStrategyShare, order *payeer.OrderDetails) {
	slog.Info("[Share "+share.ID+"] Updating balance by order details...", "order", order)
	for _, trade := range order.Trades {
		base, _ := s.store.balance.Get(share.Pair.Base())
		quote, _ := s.store.balance.Get(share.Pair.Quote())
		mul := decimal.NewFromInt(1)
		if share.Action == payeer.ACTION_SELL {
			mul = decimal.NewFromInt(-1)
		}
		base.Available += decimal.RequireFromString(trade.Amount).Mul(mul).InexactFloat64()
		quote.Available -= decimal.RequireFromString(trade.Value).Mul(mul).InexactFloat64()
		s.store.balance.Set(share.Pair.Base(), base)
		s.store.balance.Set(share.Pair.Quote(), quote)
	}
	s.updateBalanceByOrderParams(share, &payeer.OrderParams{
		Amount: order.AmountRemaining,
		Value:  order.ValueRemaining,
	}, false)
}

func (s *PayeerSharesStrategy) hasPriceChanged(share *PayeerSharesStrategyShare, order *ShareOrderInfo) bool {
	binanceTickersData, ok := s.store.binanceTickers.Get(share.BinanceSymbol)
	if !ok {
		slog.Warn("[Share "+share.ID+"] No binance tickers cahed for, Skipping", "symbol", share.BinanceSymbol)
		return false
	}

	ordersData, ok := s.store.orders.Get(share.Pair)
	if !ok {
		slog.Warn("[Share "+share.ID+"] No orders cached for. Skipping...", "pair", share.Pair)
		return false
	}

	price := resolvePriceWithElevation(share.Action, share.BinancePriceRatio, &binanceTickersData, &ordersData, s.getMyPrices(share.Pair, share.Action))

	slog.Info("[Share "+share.ID+"] Checking price for cancellation...", "old price", order.Order.Price, "new price", price)

	if decimal.RequireFromString(order.Order.Price).Equal(price) {
		slog.Info("[Share " + share.ID + "] PRICES EQUAL!")
		return false
	}

	return true
}

func (s *PayeerSharesStrategy) tryPlaceOrder(share *PayeerSharesStrategyShare) *payeer.PostOrderResponse {
	binanceTickersData, ok := s.store.binanceTickers.Get(share.BinanceSymbol)
	if !ok {
		slog.Warn("[Share "+share.ID+"] No binance tickers cahed for, Skipping", "symbol", share.BinanceSymbol)
		time.Sleep(time.Second * 1)
		return nil
	}

	ordersData, ok := s.store.orders.Get(share.Pair)
	if !ok {
		slog.Warn("[Share "+share.ID+"] No orders cached for. Skipping...", "pair", share.Pair)
		time.Sleep(time.Second * 1)
		return nil
	}

	price := resolvePriceWithElevation(share.Action, share.BinancePriceRatio, &binanceTickersData, &ordersData, s.getMyPrices(share.Pair, share.Action))

	var mainAssetName string
	var mainAssetPrecision int32

	amountPrec := int32(s.store.info.Pairs[share.Pair].AmountPrecision)

	if share.Action == payeer.ACTION_SELL {
		mainAssetName = share.Pair.Base()
		mainAssetPrecision = amountPrec
	} else {
		mainAssetName = share.Pair.Quote()
		mainAssetPrecision = int32(s.store.info.Pairs[share.Pair].ValuePrecision)
	}

	balance, ok := s.store.balance.Get(mainAssetName)
	if !ok {
		slog.Warn("[Share "+share.ID+"] No balance cached for. Skipping...", "asset", mainAssetName)
		time.Sleep(time.Second * 1)
		return nil
	}

	mainAssetQty := decimal.NewFromFloat(balance.Total).Mul(share.Share).RoundDown(mainAssetPrecision)

	if decimal.NewFromFloat(balance.Available).LessThan(mainAssetQty) {
		slog.Warn("[Share "+share.ID+"] Not enough main asset for share. Skipping...", "share", share.Share, "asset", mainAssetName, "available", balance.Available, "total", balance.Total, "required", mainAssetQty.String())
		time.Sleep(time.Second * 1)
		return nil
	}

	var amount decimal.Decimal
	if share.Action == payeer.ACTION_SELL {
		amount = mainAssetQty
	} else {
		amount = mainAssetQty.Div(price).RoundDown(amountPrec)
	}

	minAmount := decimal.NewFromFloat(s.store.info.Pairs[share.Pair].MinAmount)
	if amount.LessThan(minAmount) {
		slog.Info("[Share "+share.ID+"] Order amount less than minAmount. Skipping...", "minAmount", minAmount.String(), "orderAmount", amount.String())
		time.Sleep(time.Second * 1)
		return nil
	}

	// slog.Info("[Share "+share.ID+"] Prepared order request", "action", share.Action, "pair", share.Pair, "amount", amount, "price", price)

	return s.fetcher.PlaceOrder(share.Action, share.Pair, amount.String(), price.String())
}

func (s *PayeerSharesStrategy) runOrdersFetchLoop() {
	pairs := make([]payeer.Pair, 0, len(s.options.Shares))
	for _, share := range s.options.Shares {
		if !slices.Contains(pairs, share.Pair) {
			pairs = append(pairs, share.Pair)
		}
	}
	init := true
	for {
		if !init {
			time.Sleep(s.options.OrdersFetchInterval)
		} else {
			init = false
		}
		orders := s.fetcher.OrdersByPairs(pairs)
		for pair, info := range orders {
			s.store.orders.Set(pair, info)
		}
	}
}

/*
** Initializations
 */

func (s *PayeerSharesStrategy) initInfo() {
	info := s.fetcher.Info()
	s.store.info = info
}

func (s *PayeerSharesStrategy) initBinanceTickers() {
	slog.Info("[PayeerSharesStrategy] Initializing binance tickers...")
	for _, share := range s.options.Shares {
		if share.BinanceTickerInterval.Milliseconds() == int64(0) {
			share.BinanceTickerInterval = time.Millisecond * 200
		}
		ch := s.binanceClient.SubscribeTicker(share.BinanceSymbol, share.BinanceTickerInterval)
		go func(symbol binance.Symbol) {
			for ticker := range ch {
				s.store.binanceTickers.Set(symbol, ticker)
			}
		}(share.BinanceSymbol)
	}
	s.logInfo("Binance tickers initialized")
}

func (s *PayeerSharesStrategy) initMyOrders() {
	orders := s.fetcher.MyOrders()
	s.logInfo("Cancelling pending orders...")
	for _, order := range orders {
		orderTime := time.Unix(order.Date, 0)
		diff := time.Since(orderTime)
		if diff.Minutes() <= 1 {
			wait := time.Minute - diff
			s.logInfo("Wait for order cancel...", "orderId", order.Id, "time", wait)
			time.Sleep(wait)
		}
		orderId, _ := strconv.Atoi(order.Id)
		s.fetcher.CancelOrder(orderId)
		time.Sleep(200 * time.Millisecond)
	}
	s.logInfo("Pending orders canceled")
}

func (s *PayeerSharesStrategy) initBalance() {
	s.logInfo("(Re-)initializing balance...")
	for asset, balance := range s.fetcher.Balance() {
		if balance.Available > 0 {
			s.store.balance.Set(asset, balance)
			slog.Info("Balance update:", "asset", asset, "balance", balance)
		}
	}
	s.logInfo("Balance (re-)initialized")
}

/*
** Logging
 */

func (s *PayeerSharesStrategy) logError(msg string, args ...any) {
	slog.Error("[PayeerSharesStrategy] "+msg, args...)
}

func (s *PayeerSharesStrategy) logInfo(msg string, args ...any) {
	slog.Info("[PayeerSharesStrategy] "+msg, args...)
}

/*
** Helpers
 */
type PriceAmount struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
}

func (s *PayeerSharesStrategy) getMyPrices(pair payeer.Pair, action payeer.Action) []PriceAmount {
	prices := []PriceAmount{}
	s.store.shareOrders.Range(func(_ string, order ShareOrderInfo) bool {
		if order.Order.Pair == pair && order.Order.Action == action {
			orderPrice := decimal.RequireFromString(order.Order.Price)
			orderAmount := decimal.RequireFromString(order.Order.Amount)
			index := slices.IndexFunc(prices, func(p PriceAmount) bool {
				return p.Price.Equal(orderPrice)
			})
			if index != -1 {
				prices[index].Amount = prices[index].Amount.Add(orderAmount)
			} else {
				prices = append(prices, PriceAmount{
					Price:  orderPrice,
					Amount: orderAmount,
				})
			}
		}
		return true
	})
	return prices
}

func resolvePriceWithElevation(
	action payeer.Action,
	binancePriceRatio decimal.Decimal,
	binanceTickersData *binance.OrderBookTickerStreamResult,
	ordersData *payeer.PairsOrderInfo,
	myOrders []PriceAmount,
) decimal.Decimal {
	var binancePrice decimal.Decimal
	var orders []payeer.OrdersOrder
	if action == payeer.ACTION_BUY {
		orders = ordersData.Bids
		binancePrice = decimal.RequireFromString(binanceTickersData.BidPrice)
	} else {
		orders = ordersData.Asks
		binancePrice = decimal.RequireFromString(binanceTickersData.AskPrice)
	}

	price := binancePrice.Mul(binancePriceRatio)

	slog.Info("[PayeerPriceSelector] binance price multiplied", "action", action, "original", binancePrice.String(), "ratio", binancePriceRatio.String(), "multiplied", price.String())

	var priceFound func(orderPrice decimal.Decimal) bool
	var elevate func(orderPrice decimal.Decimal) decimal.Decimal
	cent := decimal.RequireFromString(".01")

	shouldSkip := func(price decimal.Decimal, amount decimal.Decimal) bool {
		for _, myOrder := range myOrders {
			if price.Equal(myOrder.Price) && amount.Equal(myOrder.Amount) {
				return true
			}
		}
		return false
	}

	if action == payeer.ACTION_BUY {
		priceFound = func(orderPrice decimal.Decimal) bool { return orderPrice.LessThan(price) }
		elevate = func(orderPrice decimal.Decimal) decimal.Decimal { return orderPrice.Add(cent) }
	} else {
		priceFound = func(orderPrice decimal.Decimal) bool { return orderPrice.GreaterThan(price) }
		elevate = func(orderPrice decimal.Decimal) decimal.Decimal { return orderPrice.Sub(cent) }
	}

	for i, order := range orders {
		orderPrice := decimal.RequireFromString(order.Price)
		orderAmount := decimal.RequireFromString(order.Amount)
		if !shouldSkip(orderPrice, orderAmount) && priceFound(orderPrice) {
			price = elevate(orderPrice)
			for j := i - 1; j >= 0; j-- {
				topPrice := decimal.RequireFromString(orders[j].Price)
				topAmount := decimal.RequireFromString(orders[j].Amount)
				if shouldSkip(topPrice, topAmount) {
					continue
				}
				if topPrice.Equal(price) {
					price = elevate(price)
				} else {
					break
				}
			}
			break
		}
	}
	return price
}
