package binance

type WsMethod string

const (
	METHOD_TICKER_BOOK WsMethod = "ticker.book"
)

type Symbol string

const (
	SYMBOL_BNBBTC  Symbol = "BNBBTC"
	SYMBOL_BTCUSDT Symbol = "BTCUSDT"
)

type Params struct {
	Symbols []Symbol `json:"symbols"`
}

type wsError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

type wsResponse struct {
	Id     string  `json:"id"`
	Status int     `json:"status"`
	Error  wsError `json:"error"`
}

type OrderBookTickerWsRequest struct {
	Id     string   `json:"id"`
	Method WsMethod `json:"method"`
	Params Params   `json:"params"`
}

func NewOrderBookTickerWsRequest(symbols []Symbol) *OrderBookTickerWsRequest {
	// id := fmt.Sprintf("%s:%s", METHOD_TICKER_BOOK, uuid.NewString())
	return &OrderBookTickerWsRequest{
		Id:     "1",
		Method: METHOD_TICKER_BOOK,
		Params: Params{
			Symbols: symbols,
		},
	}
}

type OrderBookTickerResult struct {
	Symbol      Symbol `json:"symbol"`
	AskPrice    string `json:"askPrice"`
	AskQuantity string `json:"askQuantity"`
	BidPrice    string `json:"bidPrice"`
	BidQuantity string `json:"bidQty"`
}

type OrderBookTickerWsResponse struct {
	Id     string `json:"id"`
	Status int    `json:"status"`
	Result []OrderBookTickerResult
}
