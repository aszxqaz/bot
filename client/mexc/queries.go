package mexc

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"mexc-bot/client"
	"time"
)

type queryMaker struct {
	secret string
}

func newQueryMaker(secret string) *queryMaker {
	return &queryMaker{secret: secret}
}

func (q *queryMaker) getOrderQuery(order *client.Order) string {
	qb := client.NewQueryBuilder()
	qb.Add("symbol", order.Symbol)
	qb.Add("side", order.Side)
	qb.Add("type", order.Type)
	qb.Add("quantity", order.OrigQty)
	if order.Price > 0 {
		qb.Add("price", order.Price)
	}
	return q.signQuery(qb)
}

func (q *queryMaker) getListenKeyQuery(listenKey string) string {
	qb := client.NewQueryBuilder()
	qb.Add("listenKey", listenKey)
	return q.signQuery(qb)
}

func (q *queryMaker) getSymbolQuery(symbol string) string {
	qb := client.NewQueryBuilder()
	qb.Add("symbol", symbol)
	return q.signQuery(qb)
}

func (q *queryMaker) getCancelOrderQuery(symbol string, orderId string) string {
	qb := client.NewQueryBuilder()
	qb.Add("symbol", symbol)
	qb.Add("orderId", orderId)
	return q.signQuery(qb)
}

func (q *queryMaker) signQuery(qb *client.QueryBuilder) string {
	timestamp := time.Now().UTC().UnixMilli()
	// slog.Info("[QUERY MAKER] signing query", "timestamp", timestamp)
	qb.Add("timestamp", timestamp)
	signature := q.sign(qb.String())
	qb.Add("signature", signature)
	return qb.String()
}

func (q *queryMaker) defaultSignature() string {
	qb := client.NewQueryBuilder()
	return q.signQuery(qb)
}

func (q *queryMaker) sign(req string) string {
	hmac := hmac.New(sha256.New, []byte(q.secret))
	hmac.Write([]byte(req))
	dataHmac := hmac.Sum(nil)
	hmacHex := hex.EncodeToString(dataHmac)
	return hmacHex
}
