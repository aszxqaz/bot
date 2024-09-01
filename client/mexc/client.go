package mexc

import (
	"encoding/json"
	"log/slog"
	"mexc-bot/client"
	httpclient "mexc-bot/http_client"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const (
	baseHttpUrl                = "https://api.mexc.com"
	baseWsUrl                  = "wss://wbs.mexc.com/ws"
	wsDealsEndpoint            = "spot@private.deals.v3.api"
	wsBalanceEndpoint          = "spot@private.account.v3.api"
	wsOrdersEndpoint           = "spot@private.orders.v3.api"
	wsETHUSDCTickersEndpoint   = "spot@public.bookTicker.v3.api@ETHUSDC"
	wsSTETHUSDCTickersEndpoint = "spot@public.bookTicker.v3.api@STETHUSDC"
)

type MexcClient struct {
	wsConn             *websocket.Conn
	apiKey             string
	httpClient         *httpclient.HttpClient
	lkm                *listenKeyManager
	qm                 *queryMaker
	DealStream         chan *client.Deal
	BalanceStream      chan *client.Balance
	TickersStream      chan *client.OrderBookTicker
	OrderUpdateStream  chan *client.OrderUpdate
	PartialDepthStream chan *client.PartialDepth
}

func NewMexcClient(apiKey string, secret string) *MexcClient {
	headers := make(http.Header)
	headers.Set("X-MEXC-APIKEY", apiKey)
	headers.Set("Content-Type", "application/json")
	httpClient := httpclient.NewHttpClient("https://api.mexc.com/api/v3")
	httpClient.SetHeaders(headers)
	qm := newQueryMaker(secret)
	lkm := &listenKeyManager{
		httpClient: httpClient,
		qm:         qm,
	}
	return &MexcClient{
		apiKey:             apiKey,
		qm:                 qm,
		httpClient:         httpClient,
		lkm:                lkm,
		DealStream:         make(chan *client.Deal, 1024),
		BalanceStream:      make(chan *client.Balance, 1024),
		TickersStream:      make(chan *client.OrderBookTicker, 1024),
		OrderUpdateStream:  make(chan *client.OrderUpdate, 1024),
		PartialDepthStream: make(chan *client.PartialDepth, 1024),
	}
}

func (m *MexcClient) Start() {
	m.lkm.Start()
	m.wsConnect()
	params := []string{
		wsDealsEndpoint,
		wsBalanceEndpoint,
		wsOrdersEndpoint,
		wsETHUSDCTickersEndpoint,
		wsSTETHUSDCTickersEndpoint,
		getPartialBookDepthStreamEndpoint(client.STETHUSDC, 5),
	}
	subscriptionMsg := map[string]any{"method": "SUBSCRIPTION", "params": params}
	if err := m.wsConn.WriteJSON(subscriptionMsg); err != nil {
		slog.Error("[MexcClient] Failed to write to ws conn:", "error", err)
		os.Exit(1)
	}
	go func() {
		for {
			_, message, err := m.wsConn.ReadMessage()
			if err != nil {
				slog.Error("[MexcClient] Failed to read ws message:", "error", err)
				os.Exit(1)
			}
			// slog.Debug("[MexcClient] Received ws message", "message", string(message))
			var wsResponse wsResponse
			err = json.Unmarshal(message, &wsResponse)
			if err != nil {
				slog.Warn("[MexcClient] Failed to unmarshal ws message as wsResponse:", "error", err)
				continue
			}
			switch wsResponse.Endpoint {
			case "":
				continue
			case wsDealsEndpoint:
				err = m.handleWsDealResponse(message)
				if err != nil {
					continue
				}
			case wsBalanceEndpoint:
				err = m.handleWsAccountUpdateMessage(message)
				if err != nil {
					continue
				}
			case wsOrdersEndpoint:
				err = m.handleWsOrderUpdateMessage(message)
				if err != nil {
					continue
				}
			case wsETHUSDCTickersEndpoint:
				err = m.handleWsTickerResponse(message)
				if err != nil {
					continue
				}
			case wsSTETHUSDCTickersEndpoint:
				err = m.handleWsTickerResponse(message)
				if err != nil {
					continue
				}
			case getPartialBookDepthStreamEndpoint(client.STETHUSDC, 5):
				err = m.handleWsPartialBookDepthResponse(message)
				if err != nil {
					continue
				}
			}
		}
	}()
}

func (m *MexcClient) handleWsPartialBookDepthResponse(message []byte) error {
	var partialDepthMsg wsPartialDepthMessage
	err := json.Unmarshal(message, &partialDepthMsg)
	if err != nil || partialDepthMsg.Timestamp == 0 {
		slog.Warn("[MexcClient] Failed to unmarshal wsPartialDepthMessage:", "error", err)
		return err
	}
	depth, err := partialDepthMsg.toPartialDepth()
	if err != nil {
		slog.Warn("[MexcClient] Failed to convert json to OrderUpdate:", "error", err)
		return err
	}
	m.PartialDepthStream <- depth
	slog.Debug("[MexcClient] Partial depth update", "depth", depth)
	return nil
}

func (m *MexcClient) handleWsOrderUpdateMessage(message []byte) error {
	var accountOrderMsg wsAccountOrdersMessage
	err := json.Unmarshal(message, &accountOrderMsg)
	if err != nil || accountOrderMsg.Timestamp == 0 {
		slog.Warn("[MexcClient] Failed to unmarshal wsAccountOrdersMessage:", "error", err)
		return err
	}
	update, err := accountOrderMsg.toOrderUpdate()
	if err != nil {
		slog.Warn("[MexcClient] Failed to convert json to OrderUpdate:", "error", err)
		return err
	}
	m.OrderUpdateStream <- update
	slog.Debug("[MexcClient] Order update", "order", update)
	return nil
}

func (m *MexcClient) handleWsAccountUpdateMessage(message []byte) error {
	var accountResponse wsAccountUpdateMessage
	err := json.Unmarshal(message, &accountResponse)
	if err != nil {
		slog.Warn("[MexcClient] Failed to unmarshal wsAccountResponse:", "error", err)
		return err
	}
	update, err := accountResponse.Update.toAccountUpdate()
	if err != nil {
		slog.Warn("[MexcClient] Failed to convert json to AccountUpdate:", "error", err)
		return err
	}
	m.BalanceStream <- update
	slog.Debug("[MexcClient] Balance update", "balance", update)
	return nil
}

func (m *MexcClient) handleWsTickerResponse(message []byte) error {
	var tickerResponse wsTickerResponse
	err := json.Unmarshal(message, &tickerResponse)
	if err != nil {
		slog.Warn("[MexcClient] Failed to unmarshal wsTickerResponse:", "error", err)
		return err
	}
	ticker, err := tickerResponse.toTicker()
	if err != nil {
		slog.Warn("[MexcClient] Failed to convert deal to client.OrderBookTicker:", "error", err)
		return err
	}
	switch tickerResponse.Symbol {
	case client.ETHUSDC:
		m.TickersStream <- ticker
	case client.STETHUSDC:
		m.TickersStream <- ticker
	default:
		slog.Warn("[MexcClient] Unknown ticker symbol. Ignoring.\n", "symbol", ticker.Symbol)
		return nil
	}
	slog.Debug("[MexcClient] Ticker update", "ticker", ticker)
	return nil
}

func (m *MexcClient) handleWsDealResponse(message []byte) error {
	var dealResponse wsDealResponse
	err := json.Unmarshal(message, &dealResponse)
	if err != nil {
		slog.Error("[MexcClient] Failed to unmarshal wsDealResponse:", "error", err)
		return err
	}
	deal, err := dealResponse.Deal.toDeal(dealResponse.Symbol)
	if err != nil {
		slog.Error("[MexcClient] Failed to convert deal to client.Deal:", "error", err)
		return err
	}
	m.DealStream <- deal
	slog.Debug("[MexcClient] Deal update", "deal", deal)
	return nil
}

func (m *MexcClient) wsConnect() {
	endpoint := baseWsUrl + "?listenKey=" + m.lkm.ListenKey()
	c, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		slog.Error("[MexcClient] Failed to dial ws", "error", err)
		os.Exit(1)
	}
	m.wsConn = c
	ticker := time.NewTicker(time.Second * 29)
	go func() {
		for range ticker.C {
			err := m.wsConn.WriteJSON(map[string]string{"method": "PING"})
			if err != nil {
				slog.Error("[MexcClient] Failed to ping ws:", "error", err)
				return
			}
		}
	}()
}

func (m *MexcClient) Balances() (map[string]client.Balance, error) {
	var account accountResponse
	err := m.httpClient.Get("/account?"+m.qm.defaultSignature(), &account)
	if err != nil {
		slog.Error("[MexcClient] Failed to get account data", "error", err)
		return nil, err
	}
	balances := make(map[string]client.Balance)
	for _, b := range account.Balances {
		balances[b.Asset] = b
	}
	return balances, nil
}

func (m *MexcClient) PlaceOrder(order *client.Order) error {
	query := m.qm.getOrderQuery(order)
	err := m.httpClient.Post("/order?"+query, &order)
	if err != nil {
		slog.Error("[MexcClient] Failed to place order", "error", err)
		return err
	}
	slog.Debug("[MexcClient] Order placed", "order", order)
	return nil
}

func (m *MexcClient) OrderBookTicker(symbol string) (*client.OrderBookTicker, error) {
	var tickerJson orderBookTicker
	err := m.httpClient.Get("/ticker/bookTicker?"+m.qm.getSymbolQuery(symbol), &tickerJson)
	if err != nil {
		slog.Error("[MexcClient] Failed to get order book ticker", "error", err)
		return nil, err
	}
	ticker, err := tickerJson.toOrderBookTicker()
	if err != nil {
		slog.Error("[MexcClient] Failed to convert order book ticker json to struct", "error", err)
		return nil, err
	}
	slog.Debug("[MexcClient] Got order book ticker", "ticker", ticker)
	return ticker, nil
}

func (m *MexcClient) CancelOrder(symbol string, orderId string) error {
	var order *client.Order
	err := m.httpClient.Delete("/order?"+m.qm.getCancelOrderQuery(symbol, orderId), &order)
	if err != nil {
		slog.Error("[MexcClient] Failed to cancel order", "error", err)
		return err
	}
	slog.Debug("[MexcClient] Order canceled.", "order", order)
	return nil
}
