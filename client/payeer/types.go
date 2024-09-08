package payeer

import "strings"

// Trade sides
type Action string

const (
	ACTION_BUY  Action = "buy"
	ACTION_SELL Action = "sell"
)

// Symbols
type Pair string

const (
	PAIR_BTCUSD  Pair = "BTC_USD"
	PAIR_BTCRUB  Pair = "BTC_RUB"
	PAIR_BTCEUR  Pair = "BTC_EUR"
	PAIR_BTCUSDT Pair = "BTC_USDT"
	PAIR_ETHUSD  Pair = "ETH_USD"
	PAIR_ETHRUB  Pair = "ETH_RUB"
	PAIR_ETHEUR  Pair = "ETH_EUR"
	PAIR_ETHBTC  Pair = "ETH_BTC"
	// "ETH_USDT"
	// "BCH_USD"
	// "BCH_RUB"
	// "BCH_EUR"
	// "BCH_BTC"
	// "BCH_ETH"
	// "BCH_DASH"
	// "BCH_USDT"
	// "BCH_XRP"
	// "DASH_USD"
	// "DASH_RUB"
	// "DASH_EUR"
	// "DASH_BTC"
	// "DASH_ETH"
	// "DASH_LTC"
	// "DASH_USDT"
	// "DASH_XRP"
	// "LTC_USD"
	// "LTC_RUB"
	// "LTC_EUR"
	// "LTC_BTC"
	// "LTC_ETH"
	// "LTC_BCH"
	// "LTC_USDT"
	// "XRP_USD"
	// "XRP_RUB"
	// "XRP_EUR"
	// "XRP_BTC"
	// "XRP_USDT"
	// "XRP_ETH"
	// "XRP_LTC"
	// "EUR_USD"
	// "EUR_USDT"
	// "EUR_RUB"
	PAIR_USDRUB Pair = "USD_RUB"
	// "USDT_USD"
	PAIR_USDTRUB Pair = "USDT_RUB"
	// "DOGE_USD"
	// "DOGE_USDT"
	// "DOGE_RUB"
	// "DOGE_EUR"
	// "DOGE_BTC"
	// "TRX_USD"
	// "TRX_USDT"
	// "TRX_RUB"
	// "TRX_EUR"
	// "TRX_BTC"
	// "BNB_USD"
	// "BNB_USDT"
	// "BNB_RUB"
	// "BNB_EUR"
	// "BNB_BTC"
	// "DOT_USDT"
	// "DOT_RUB"
	// "DOT_EUR"
	// "DOT_BTC"
	// "DAI_USDT"
	// "MATIC_USDT"
	// "MATIC_RUB"
	// "MATIC_EUR"
	// "MATIC_BTC"
	// "USDC_USDT"
)

func (p Pair) String() string {
	return string(p)
}

func (p Pair) Base() string {
	return strings.Split(string(p), "_")[0]
}

func (p Pair) Quote() string {
	return strings.Split(string(p), "_")[1]
}

// Order types
type OrderType string

const (
	ORDER_TYPE_LIMIT      OrderType = "limit"
	ORDER_TYPE_MARKET     OrderType = "market"
	ORDER_TYPE_STOP_LIMIT OrderType = "stop_limit"
)

// Order statuses
type OrderStatus string

const (
	ORDER_STATUS_SUCCESS    OrderStatus = "success"
	ORDER_STATUS_PROCESSING OrderStatus = "processing"
	ORDER_STATUS_WAITING    OrderStatus = "waiting"
	ORDER_STATUS_CANCELED   OrderStatus = "canceled"
)

// Trade status
type TradeStatus string

const (
	TRADE_STATUS_SUCCESS    TradeStatus = "success"
	TRADE_STATUS_PROCESSING TradeStatus = "processing"
)
