package main

import (
	"automata/client/binance"
	"automata/client/payeer"
	"automata/msync"
	"log/slog"
	"os"
	"time"

	"github.com/shopspring/decimal"
)

type PayeerMarketTraderOptions struct {
	Pairs                 map[payeer.Pair]binance.Symbol
	BinanceTickerInterval time.Duration
	TradeLoopInterval     time.Duration
	BidMinRatio           decimal.Decimal
	AskMaxRatio           decimal.Decimal
	MaxBuyAmount          decimal.Decimal
	QuoteMult             decimal.Decimal
}

type PayeerMarketTrader struct {
	payeerClient     *payeer.Client
	binanceClient    *binance.Client
	minWeights       *msync.Mu[int]
	info             *payeer.InfoResponse
	balance          *msync.MuMap[string, payeer.Balance]
	orders           *msync.MuMap[payeer.Pair, payeer.PairsOrderInfo]
	weightsTimestamp *msync.Mu[time.Time]
	binanceTickers   *msync.MuMap[binance.Symbol, binance.OrderBookTickerStreamResult]
	options          *PayeerMarketTraderOptions
}

func NewPayeerMarketTrader(p *payeer.Client, b *binance.Client, o *PayeerMarketTraderOptions) *PayeerMarketTrader {
	binanceTickers := msync.NewMuMap[binance.Symbol, binance.OrderBookTickerStreamResult]()
	for _, symbol := range o.Pairs {
		ch := b.SubscribeTicker(symbol, o.BinanceTickerInterval)
		go func() {
			for ticker := range ch {
				binanceTickers.Set(symbol, ticker)
			}
		}()
	}
	return &PayeerMarketTrader{
		payeerClient:     p,
		binanceClient:    b,
		options:          o,
		minWeights:       msync.NewMu[int](600),
		weightsTimestamp: msync.NewMu[time.Time](time.Now()),
		binanceTickers:   binanceTickers,
		orders:           msync.NewMuMap[payeer.Pair, payeer.PairsOrderInfo](),
		balance:          msync.NewMuMap[string, payeer.Balance](),
	}
}

func (s *PayeerMarketTrader) Start() {
	s.resetInfo()
	s.fetchAndUpdateBalance()
	go s.fetchOrdersLoop()
	for _, pair := range s.getPairs() {
		go s.tradeLoop(pair, payeer.ACTION_BUY)
		go s.tradeLoop(pair, payeer.ACTION_SELL)
	}
	select {} // endless block of the process
}

func (s *PayeerMarketTrader) tradeLoop(pair payeer.Pair, action payeer.Action) {
	for {
		time.Sleep(s.options.TradeLoopInterval)

		// Getting cached binance tickers data for the symbol
		binanceTickersData, ok := s.binanceTickers.Get(s.options.Pairs[pair])
		if !ok {
			slog.Error("[PayeerMarketTrader] No binance ticker cached for", "symbol", s.options.Pairs[pair])
			continue
		}

		// Getting cached payeer orders data for the pair
		ordersData, ok := s.orders.Get(pair)
		if !ok {
			slog.Error("[PayeerMarketTrader] No orders cached for", "pair", pair)
			continue
		}

		// Resolving action-specific data based on the action
		var orders []payeer.OrdersOrder
		var doesPriceSatisfy func(price decimal.Decimal) bool
		if action == payeer.ACTION_BUY {
			orders = ordersData.Asks
			binanceAskPrice := decimal.RequireFromString(binanceTickersData.AskPrice)
			doesPriceSatisfy = func(price decimal.Decimal) bool {
				return price.Div(binanceAskPrice).LessThan(s.options.AskMaxRatio)
			}
		} else {
			orders = ordersData.Bids
			binanceBidPrice := decimal.RequireFromString(binanceTickersData.BidPrice)
			doesPriceSatisfy = func(price decimal.Decimal) bool {
				return price.Div(binanceBidPrice).GreaterThan(s.options.BidMinRatio)
			}
		}

		// Summing up the satisfying orders amounts to get the total amount
		totalAmount := decimal.Zero
		satisfyingOrders := []payeer.OrdersOrder{}
		for _, order := range orders {
			orderPrice := decimal.RequireFromString(order.Price)
			if doesPriceSatisfy(orderPrice) {
				orderAmount := decimal.RequireFromString(order.Amount)
				totalAmount = totalAmount.Add(orderAmount)
				satisfyingOrders = append(satisfyingOrders, order)
			}
		}

		if totalAmount.IsZero() {
			slog.Info("[PayeerMarketTrader] Total amount is zero", "pair", pair, "action", action)
			continue
		}

		// Resolving the amount of an order to be placed based on the total amount, balance and configuration
		var orderAmount decimal.Decimal
		if action == payeer.ACTION_BUY {
			// Getting cached balance for the quote
			quoteBalance, ok := s.balance.Get(pair.Quote())
			if !ok {
				slog.Info("[PayerrMarketTrader] Zero quote balance for", "quote", pair.Quote())
				continue
			}

			// Resolving orderAmount based on the quote balance
			quoteBalanceAvailable := decimal.NewFromFloat(quoteBalance.Available)
			orderAmount = decimal.Zero
			for _, order := range satisfyingOrders {
				value := decimal.RequireFromString(order.Value)
				price := decimal.RequireFromString(order.Price)
				orderQuote := decimal.Min(quoteBalanceAvailable, value)
				// The amount from the order is adjusted by multiplying the order price by QuoteMult
				orderAmount = orderAmount.Add(orderQuote.Div(price.Mul(s.options.QuoteMult)))
				quoteBalanceAvailable = quoteBalanceAvailable.Sub(orderQuote)
				if quoteBalanceAvailable.IsZero() {
					break
				}
			}

			// Fixing the precision
			precision := int32(s.info.Pairs[pair].AmountPrecision)
			orderAmount = orderAmount.RoundFloor(precision)

			slog.Info("[PayeerMarketTrader] Order amount calculated", "pair", pair, "action", action, "orderAmount", orderAmount.String(), "quoteBalance", quoteBalance.Available)

			// Restricting the order amount if MaxBuyAmount is positivie
			if s.options.MaxBuyAmount.IsPositive() {
				orderAmount = decimal.Min(orderAmount, s.options.MaxBuyAmount)
				slog.Info("[PayeerMarketTrader] Order amount restricted", "orderAmount", orderAmount, "maxBuyAmount", s.options.MaxBuyAmount.String())
			}
		} else {
			// Getting cached balance for the base
			baseBalance, ok := s.balance.Get(pair.Base())
			if !ok {
				slog.Info("[PayerrMarketTrader] Zero base balance for", "base", pair.Base())
				continue
			}

			// Resolving orderAmount based on the base balance
			baseBalanceAvailable := decimal.NewFromFloat(baseBalance.Available)
			orderAmount = decimal.Min(totalAmount, baseBalanceAvailable)
		}

		// Checking against the minimum amount for the pair
		minAmount := decimal.NewFromFloat(s.info.Pairs[pair].MinAmount)
		if orderAmount.LessThan(minAmount) {
			slog.Info("[PayeerMarketTrader] Order amount less than minAmount", "minAmount", minAmount.String(), "orderAmount", orderAmount.String())
			continue
		}

		// Placing the order
		slog.Info("[PayeerMarketTrader] Market order should be placed", "pair", pair, "action", action, "amount", orderAmount.String(), "satisfying orders", satisfyingOrders)
		// rsp := s.placeMarketOrder(payeer.ACTION_BUY, pair, totalAmount.String())
		// slog.Info("[PayeerMarketTrader] Market order placed", "orderId", rsp.OrderId, "details", rsp.Params)
		// s.fetchAndUpdateBalance()
	}
}

func (s *PayeerMarketTrader) fetchOrdersLoop() {
	pairs := s.getPairs()
	for {
		orders := s.fetchOrders(pairs)
		for p, o := range orders.Pairs {
			s.orders.Set(p, o)
		}
	}
}

func (s *PayeerMarketTrader) fetchAndUpdateBalance() {
	for asset, balance := range s.fetchBalance() {
		if balance.Available > 0 {
			s.balance.Set(asset, balance)
			slog.Info("[PayeerMarketTrader] Balance update:", "asset", asset, "balance", balance)
		}
	}
}

func (s *PayeerMarketTrader) resetInfo() {
	info, err := s.payeerClient.Info()
	if err != nil {
		panic(err)
	}
	if !info.Success {
		slog.Error("[PayeerMarketTrader] Info response error", "error", info.Error)
		os.Exit(1)
	}
	s.info = info
}

func (s *PayeerMarketTrader) placeMarketOrder(action payeer.Action, pair payeer.Pair, amount string) *payeer.PostOrderResponse {
	for {
		rsp, err := s.payeerClient.PlaceOrder(&payeer.PostOrderRequest{
			Pair:   pair,
			Type:   payeer.ORDER_TYPE_MARKET,
			Action: action,
			Amount: amount,
		})
		if err != nil {
			slog.Error("[PayeerMarketTrader] HTTP error occured. Retrying...", "error", err)
			continue
		}
		s.updateWeights(10)
		if !rsp.Success {
			slog.Error("[PayeerMarketTrader] Place order response error", "response", rsp)
			os.Exit(1)
		}
		slog.Info("[PayeerMarketTrader] Order placed:", "order", rsp)
		return rsp
	}
}

func (s *PayeerMarketTrader) fetchOrders(pairs []payeer.Pair) *payeer.OrdersResponse {
	for {
		orders, err := s.payeerClient.Orders(pairs)
		if err != nil {
			slog.Error("[PayeerMarketTrader] HTTP error occured. Retrying...", "error", err)
			continue
		}
		s.updateWeights(len(pairs))
		if !orders.Success {
			slog.Error("[PayeerMarketTrader] Orders response error", "error", orders.Error)
			os.Exit(1)
		}
		return orders
	}
}

func (s *PayeerMarketTrader) fetchBalance() map[string]payeer.Balance {
	for {
		balance, err := s.payeerClient.Balance()
		if err != nil {
			slog.Error("[PayeerMarketTrader] HTTP error occured. Retrying...", "error", err)
			continue
		}
		s.updateWeights(10)
		if !balance.Success {
			slog.Error("[PayeerMarketTrader] Balance response error", "error", balance.Error)
			os.Exit(1)
		}
		slog.Debug("[PayeerMarketTrader] Payeer balance fetched", "balance", balance.Balances)
		return balance.Balances
	}

}

func (s *PayeerMarketTrader) updateWeights(count int) {
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

func (s *PayeerMarketTrader) getPairs() []payeer.Pair {
	pairs := make([]payeer.Pair, 0, len(s.options.Pairs))
	for pair := range s.options.Pairs {
		pairs = append(pairs, pair)
	}
	return pairs
}
