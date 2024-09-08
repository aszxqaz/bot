package binance

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	baseWsUrl     = "wss://ws-api.binance.com:443/ws-api/v3"
	baseStreamUrl = "wss://data-stream.binance.vision"
)

// {
//   "method": "SUBSCRIBE",
//   "params": [
//     "btcusdt@aggTrade",
//     "btcusdt@depth"
//   ],
//   "id": 1
// }

type Client struct {
	wsConn *websocket.Conn
}

func NewClient() *Client {
	return &Client{}
}

func (b *Client) SubscribeTicker(symbol Symbol, interval time.Duration) chan OrderBookTickerStreamResult {
	url := baseStreamUrl + "/ws/" + strings.ToLower(string(symbol)) + "@bookTicker"
	slog.Debug("[BinanceClient] SubscribeTicker called", "url", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		slog.Error("[BinanceClient] Failed to dial stream", "error", err)
		os.Exit(1)
	}
	orders := make(chan OrderBookTickerStreamResult, 1024)
	go func() {
		now := time.Now()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				slog.Error("[BinanceClient] Failed to read ws message:", "error", err)
				os.Exit(1)
			}
			slog.Debug("[BinanceClient] Received ws message", "symbol", string(symbol), "message", string(msg))
			var result OrderBookTickerStreamResult
			err = json.Unmarshal(msg, &result)
			if err != nil {
				slog.Warn("[BinanceClient] Failed to unmarshal ws message as OrderBookTickerStreamResult:", "error", err)
				continue
			}
			if time.Since(now) >= interval {
				now = time.Now()
				orders <- result
			}
		}
	}()
	return orders
}

// func (b *Client) Start() {
// 	b.wsConnect()
// 	go b.listen()
// }

func (b *Client) listen() {
	for {
		_, msg, err := b.wsConn.ReadMessage()
		if err != nil {
			slog.Error("[BinanceClient] Failed to read ws message:", "error", err)
			os.Exit(1)
		}
		slog.Debug("[BinanceClient] Received ws message", "message", string(msg))
	}
	// for {
	// 	_, msg, err := b.wsConn.ReadMessage()
	// 	if err != nil {
	// 		slog.Error("[BinanceClient] Failed to read ws message:", "error", err)
	// 		os.Exit(1)
	// 	}
	// 	slog.Debug("[BinanceClient] Received ws message", "message", string(msg))
	// 	var wsResponse wsResponse
	// 	err = json.Unmarshal(msg, &wsResponse)
	// 	if err != nil {
	// 		slog.Warn("[BinanceClient] Failed to unmarshal ws message as wsResponse:", "error", err)
	// 		continue
	// 	}
	// 	method := strings.SplitAfter(wsResponse.Id, ":")[0]
	// 	switch method {
	// 	case string(METHOD_TICKER_BOOK):
	// 		// b.handleTickerBookResponse(msg)
	// 	default:
	// 		slog.Warn("[BinanceClient] Unknown ws response method:", "method", method)
	// 	}
	// }
}

// func (b *Client) GetOrderBookTickers(symbols []Symbol) (*OrderBookTickerWsResponse, error) {
// 	request := NewOrderBookTickerWsRequest(symbols)
// 	bytes, _ := json.Marshal(request)
// 	slog.Debug("[BinanceClient] OrderBookTickerWsRequest", "json", string(bytes))
// 	err := b.wsConn.WriteJSON(request)
// 	if err != nil {
// 		slog.Error("[BinanceClient] Failed to send ws request:", "error", err)
// 		return nil, err
// 	}
// 	_, message, err := b.wsConn.ReadMessage()
// 	if err != nil {
// 		slog.Error("[BinanceClient] Failed to read ws response:", "error", err)
// 		return nil, err
// 	}
// 	var wsResponse wsResponse
// 	err = json.Unmarshal(message, &wsResponse)
// 	if err != nil {
// 		slog.Warn("[BinanceClient] Failed to unmarshal ws response as wsResponse:", "error", err)
// 		return nil, err
// 	}
// 	if wsResponse.Status >= 400 {
// 		slog.Error("[BinanceClient] GetOrderBookTickers error:", "code", wsResponse.Error.Code, "message", wsResponse.Error.Message)
// 		return nil, err
// 	}
// 	var resp OrderBookTickerWsResponse
// 	err = json.Unmarshal(message, &resp)
// 	if err != nil {
// 		slog.Error("[BinanceClient] Failed to unmarshal ws response as OrderBookTickerWsResponse:", "error", err)
// 		return nil, err
// 	}
// 	return &resp, nil
// }

func (b *Client) wsConnect() {
	wsConn, _, err := websocket.DefaultDialer.Dial(baseWsUrl, nil)
	if err != nil {
		slog.Error("[BinanceClient] Failed to dial ws", "error", err)
		os.Exit(1)
	}
	slog.Debug("[BinanceClient] Ws dialed successfully")
	b.wsConn = wsConn
	// ticker := time.NewTicker(time.Second * 29)
	// go func() {
	// 	for range ticker.C {
	// 		err := b.wsConn.WriteJSON(map[string]string{"method": "PING"})
	// 		if err != nil {
	// 			slog.Error("[BinanceClient] Failed to ping ws:", "error", err)
	// 			return
	// 		}
	// 	}
	// }()
}
