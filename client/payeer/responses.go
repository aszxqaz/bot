package payeer

type ResponseErrorCode string

const (
	ERR_INVALID_SIGNATURE         ResponseErrorCode = "INVALID_SIGNATURE"
	ERR_INVALID_IP_ADDRESS        ResponseErrorCode = "INVALID_IP_ADDRESS"
	ERR_LIMIT_EXCEEDED            ResponseErrorCode = "LIMIT_EXCEEDED"
	ERR_INVALID_TIMESTAMP         ResponseErrorCode = "INVALID_TIMESTAMP"
	ERR_ACCESS_DENIED             ResponseErrorCode = "ACCESS_DENIED"
	ERR_INVALID_PARAMETER         ResponseErrorCode = "INVALID_PARAMETER"
	ERR_PARAMETER_EMPTY           ResponseErrorCode = "PARAMETER_EMPTY"
	ERR_INVALID_STATUS_FOR_REFUND ResponseErrorCode = "INVALID_STATUS_FOR_REFUND"
	ERR_REFUND_LIMIT              ResponseErrorCode = "REFUND_LIMIT"
	ERR_UNKNOWN_ERROR             ResponseErrorCode = "UNKNOWN_ERROR"
	ERR_INVALID_DATE_RANGE        ResponseErrorCode = "INVALID_DATE_RANGE"
	ERR_INSUFFICIENT_FUNDS        ResponseErrorCode = "INSUFFICIENT_FUNDS"
	ERR_INSUFFICIENT_VOLUME       ResponseErrorCode = "INSUFFICIENT_VOLUME"
	ERR_INCORRECT_PRICE           ResponseErrorCode = "INCORRECT_PRICE"
	ERR_MIN_AMOUNT                ResponseErrorCode = "MIN_AMOUNT"
	ERR_MIN_VALUE                 ResponseErrorCode = "MIN_VALUE"
)

type ResponseError struct {
	Code ResponseErrorCode `json:"code"`
}

type BaseResponse struct {
	Success bool          `json:"success"`
	Error   ResponseError `json:"error"`
}

// Info [/info]
type Limit struct {
	Interval string `json:"interval"`
	Num      int    `json:"interval_num"`
	Limit    int    `json:"limit"`
}

type Limits struct {
	Requests []Limit `json:"requests"`
	Weights  []Limit `json:"weights"`
	Orders   []Limit `json:"orders"`
}

type PairInfo struct {
	PricePrecision  int     `json:"price_prec"`
	AmountPrecision int     `json:"amount_prec"`
	ValuePrecition  int     `json:"value_prec"`
	MinPrice        string  `json:"min_price"`
	MaxPrice        string  `json:"max_price"`
	MinAmount       float64 `json:"min_amount"`
	MinValue        float64 `json:"min_value"`
	FeeMakerPercent float64 `json:"fee_maker_percent"`
	FeeTakerPercent float64 `json:"fee_taker_percent"`
}

type InfoResponse struct {
	Success bool   `json:"success"`
	Limits  Limits `json:"limits"`
	Pairs   map[Pair]PairInfo
}

// Balance [/account]
type Balance struct {
	Total     float64 `json:"total"`
	Available float64 `json:"available"`
	Hold      float64 `json:"hold"`
}

type BalanceRequest struct {
	Timestamp int64 `json:"ts"`
}

type BalanceResponse struct {
	BaseResponse
	Balances map[string]Balance `json:"balances"`
}

// New order [/order_create]
type PostOrderRequest struct {
	Pair      Pair      `json:"pair"`
	Type      OrderType `json:"type"`
	Action    Action    `json:"action"`
	Amount    string    `json:"amount"`
	Price     string    `json:"price"`
	Timestamp int64     `json:"ts"`
}

type OrderParams struct {
	Pair      Pair      `json:"pair"`
	Type      OrderType `json:"type"`
	Action    Action    `json:"action"`
	Amount    string    `json:"amount"`
	Price     string    `json:"price"`
	Value     string    `json:"value"`
	StopPrice string    `json:"stop_price"`
}

type PostOrderResponse struct {
	BaseResponse
	OrderId int         `json:"order_id"`
	Params  OrderParams `json:"params"`
}

// Order status [/order_status]
type OrderStatusRequest struct {
	OrderId   int   `json:"order_id"`
	Timestamp int64 `json:"ts"`
}

type OrderStatusTrade struct {
	Id                 int         `json:"id"`
	Date               int64       `json:"date"`
	Status             TradeStatus `json:"status"`
	Price              string      `json:"price"`
	Amount             string      `json:"amount"`
	Value              string      `json:"value"`
	IsMaker            bool        `json:"is_maker"`
	IsTaker            bool        `json:"is_taker"`
	MakerTransactionId int         `json:"m_transaction_id"`
	MakerCommission    string      `json:"m_fee"`
	TakerTransactionId int         `json:"t_transaction_id"`
	TakerCommission    string      `json:"t_fee"`
}

type OrderDetails struct {
	Id              int                `json:"id"`
	Date            int64              `json:"date"`
	Pair            Pair               `json:"pair"`
	Action          Action             `json:"action"`
	Type            OrderType          `json:"type"`
	Status          OrderStatus        `json:"status"`
	Amount          string             `json:"amount"`
	Price           string             `json:"price"`
	StopPrice       string             `json:"stop_price"`
	Value           string             `json:"value"`
	AmountProcessed string             `json:"amount_processed"`
	AmountRemaining string             `json:"amount_remaining"`
	ValueProcessed  string             `json:"value_processed"`
	ValueRemaining  string             `json:"value_remaining"`
	AveragePrice    string             `json:"avg_price"`
	Trades          []OrderStatusTrade `json:"trades"`
}

type OrderStatusResponse struct {
	BaseResponse
	Order OrderDetails `json:"order"`
}

// Cancel order [/order_cancel]
type CancelOrderRequest struct {
	OrderId   int   `json:"order_id"`
	Timestamp int64 `json:"ts"`
}

type CancelOrderResponse struct {
	BaseResponse
}

// Orders [/orders]
type OrdersRequest struct {
	Pairs string `json:"pair"`
}

type OrdersOrder struct {
	Price  string `json:"price"`
	Amount string `json:"amount"`
	Value  string `json:"value"`
}

type PairsOrderInfo struct {
	Ask  string        `json:"ask"`
	Bid  string        `json:"bid"`
	Asks []OrdersOrder `json:"asks"`
	Bids []OrdersOrder `json:"bids"`
}

type OrdersResponse struct {
	BaseResponse
	Pairs map[Pair]PairsOrderInfo `json:"pairs"`
}

// Price statistics [/ticker]
type TickersRequest struct {
	Pairs string `json:"pair"`
}

type Ticker struct {
	Ask        string `json:"ask"`
	Bid        string `json:"bid"`
	Last       string `json:"last"`
	Min24h     string `json:"min24"`
	Max24h     string `json:"max24"`
	Delta      string `json:"delta"`
	DeltaPrice string `json:"delta_price"`
}

type TickersResponse struct {
	BaseResponse
	Pairs map[Pair]Ticker `json:"pairs"`
}
