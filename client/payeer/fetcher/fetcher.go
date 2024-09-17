package fetcher

import (
	"automata/client/payeer"
	"automata/msync"
	"log/slog"
	"os"
	"time"
)

const POINTS_LOG_OFFSET = 100

type Fetcher struct {
	payeerClient   *payeer.Client
	curWPoints     *msync.Mu[int]
	lastWTimestamp *msync.Mu[time.Time]
	lastWPoints    *msync.Mu[int]
}

func NewFetcher(
	payeerClient *payeer.Client,
) *Fetcher {
	return &Fetcher{
		payeerClient:   payeerClient,
		curWPoints:     msync.NewMu(600),
		lastWPoints:    msync.NewMu(600),
		lastWTimestamp: msync.NewMu(time.Now()),
	}
}

func (s *Fetcher) Info() *payeer.InfoResponse {
	info, err := s.payeerClient.Info()
	if err != nil {
		panic(err)
	}
	if !info.Success {
		slog.Error("[PayeerFetcher] Info response error", "error", info.Error)
		os.Exit(1)
	}
	return info
}

func (s *Fetcher) MyOrders() map[string]payeer.MyOrdersOrder {
	ordersRsp, err := s.payeerClient.MyOrders(&payeer.MyOrdersRequest{})
	if err != nil {
		panic(err)
	}

	s.updateWeights(60)
	if !ordersRsp.Success {
		slog.Error("[PayeerFetcher] MyOrders response error", "error", ordersRsp.Error)
		os.Exit(1)
	}
	return ordersRsp.Orders
}

func (s *Fetcher) OrderDetails(orderId int) *payeer.OrderDetails {
	orderStatusRsp, err := s.payeerClient.OrderStatus(&payeer.OrderStatusRequest{OrderId: orderId})
	if err != nil {
		panic(err)
	}
	s.updateWeights(5)
	if !orderStatusRsp.Success {
		slog.Error("[PayeerFetcher] Order status response error", "error", orderStatusRsp.Error)
		os.Exit(1)
	}
	return &orderStatusRsp.Order
}

func (s *Fetcher) PlaceOrder(action payeer.Action, pair payeer.Pair, amount string, price string) *payeer.PostOrderResponse {
	for {
		rsp, err := s.payeerClient.PlaceOrder(&payeer.PostOrderRequest{
			Pair:   pair,
			Type:   payeer.ORDER_TYPE_LIMIT,
			Action: action,
			Amount: amount,
			Price:  price,
		})
		if err != nil {
			slog.Error("[PayeerFetcher] Place order HTTP error. Retrying...", "error", err)
			continue
		}
		s.updateWeights(5)
		if !rsp.Success {
			slog.Error("[PayeerFetcher] Place order response - no success", "response", rsp)
		}
		slog.Info("[PayeerFetcher] Order placed:", "order", rsp)
		return rsp
	}
}

func (s *Fetcher) CancelOrder(orderId int) *payeer.CancelOrderResponse {
	for {
		rsp, err := s.payeerClient.CancelOrder(&payeer.CancelOrderRequest{
			OrderId: orderId,
		})
		if err != nil {
			slog.Error("[PayeerFetcher] Cancel order HTTP error. Retrying...", "error", err)
			continue
		}
		s.updateWeights(10)
		if !rsp.Success {
			slog.Error("[PayeerFetcher] Cancel order response - no success", "error", rsp.Error)
			return rsp
		}
		slog.Info("[PayeerFetcher] Order canceled", "orderId", orderId)
		return rsp
	}
}

func (s *Fetcher) Balance() map[string]payeer.Balance {
	for {
		balance, err := s.payeerClient.Balance()
		if err != nil {
			slog.Error("[PayeerFetcher] Balance response HTTP error. Retrying...", "error", err)
			continue
		}
		s.updateWeights(10)
		if !balance.Success {
			slog.Error("[PayeerFetcher] Balance response error", "error", balance.Error)
			os.Exit(1)
		}
		slog.Debug("[PayeerFetcher] Payeer balance:", "balance", balance.Balances)
		return balance.Balances
	}
}

func (s *Fetcher) OrdersByPairs(pairs []payeer.Pair) map[payeer.Pair]payeer.PairsOrderInfo {
	for {
		orders, err := s.payeerClient.Orders(pairs)
		if err != nil {
			slog.Error("[PayeerFetcher] Orders response HTTP error. Retrying...", "error", err)
			continue
		}
		s.updateWeights(len(pairs))
		if !orders.Success {
			slog.Error("[PayeerFetcher] Orders response error", "error", orders.Error)
			os.Exit(1)
		}
		return orders.Pairs
	}
}

func (s *Fetcher) Orders(pair payeer.Pair) payeer.PairsOrderInfo {
	for {
		orders, err := s.payeerClient.Orders([]payeer.Pair{pair})
		if err != nil {
			slog.Error("[PayeerFetcher] Orders response HTTP error. Retrying...", "error", err)
			continue
		}
		s.updateWeights(1)
		if !orders.Success {
			slog.Error("[PayeerFetcher] Orders response error", "error", orders.Error)
			os.Exit(1)
		}
		return orders.Pairs[pair]
	}
}

func (s *Fetcher) updateWeights(count int) {
	now := time.Now()
	if now.Sub(s.lastWTimestamp.Get()).Minutes() > 1 {
		s.lastWTimestamp.Set(now)
		s.curWPoints.Set(600 - count)
		return
	} else {
		s.curWPoints.Update(func(value int) int {
			return value - count
		})
	}
	if s.curWPoints.Get()/POINTS_LOG_OFFSET != s.lastWPoints.Get()/POINTS_LOG_OFFSET || s.curWPoints.Get() < 100 {
		slog.Info("[PayeerFetcher] Weights info", "remaining/min", s.curWPoints.Get())
	}
}
