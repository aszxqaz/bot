package payeer

import (
	"automata/signer"
	"errors"
	"strings"

	fastshot "github.com/opus-domini/fast-shot"
	"github.com/opus-domini/fast-shot/constant/mime"
)

const (
	baseUrl = "https://payeer.com/api/trade"
)

type Config struct {
	ApiId  string
	Secret string
}

type Client struct {
	config     *Config
	httpClient fastshot.ClientHttpMethods
}

func NewClient(config *Config) *Client {
	httpClient := setupHttpClient(config.ApiId)
	return &Client{
		config:     config,
		httpClient: httpClient,
	}
}

func (p *Client) Info() (*InfoResponse, error) {
	fastResp, err := p.httpClient.
		GET("/info").
		Send()
	if err != nil {
		return nil, err
	}
	if fastResp.Status().IsError() {
		body, _ := fastResp.Body().AsString()
		return nil, errors.New(body)
	}
	var data InfoResponse
	err = fastResp.Body().AsJSON(&data)
	if err != nil {
		return nil, err
	}
	return &data, err
}

// Request Weight: 5 (10 for a market order)
func (p *Client) PlaceOrder(req *PostOrderRequest) (*PostOrderResponse, error) {
	req.Timestamp = getTimestamp()
	body := mustMarshalJson(req)
	sign := p.signBody("order_create", body)
	fastResp, err := p.httpClient.
		POST("/order_create").
		Header().Add("API-SIGN", string(sign)).
		Body().AsString(string(body)).
		Send()
	if err != nil {
		return nil, err
	}
	if fastResp.Status().IsError() {
		body, _ := fastResp.Body().AsString()
		return nil, errors.New(body)
	}
	var data PostOrderResponse
	err = fastResp.Body().AsJSON(&data)
	if err != nil {
		return nil, err
	}
	return &data, err
}

// Request Weight: 5
func (p *Client) OrderStatus(req *OrderStatusRequest) (*OrderStatusResponse, error) {
	req.Timestamp = getTimestamp()
	body := mustMarshalJson(req)
	sign := p.signBody("order_status", body)
	fastResp, err := p.httpClient.POST("/order_status").
		Header().Add("API-SIGN", string(sign)).
		Body().AsString(string(body)).
		Send()
	if err != nil {
		return nil, err
	}
	if fastResp.Status().IsError() {
		body, _ := fastResp.Body().AsString()
		return nil, errors.New(body)
	}
	var data OrderStatusResponse
	err = fastResp.Body().AsJSON(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// Request Weight: 10
func (p *Client) Balance() (*BalanceResponse, error) {
	req := BalanceRequest{}
	req.Timestamp = getTimestamp()
	body := mustMarshalJson(req)
	sign := p.signBody("account", body)
	fastResp, err := p.httpClient.GET("/account").
		Header().Add("API-SIGN", string(sign)).
		Body().AsString(string(body)).
		Send()
	if err != nil {
		return nil, err
	}
	// slog.Info("[PAYEER CLIENT] Balance response", "json", text)
	if fastResp.Status().IsError() {
		text, _ := fastResp.Body().AsString()
		return nil, errors.New(text)
	}
	var data BalanceResponse
	err = fastResp.Body().AsJSON(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// Request Weight: 10
func (p *Client) CancelOrder(req *CancelOrderRequest) (*CancelOrderResponse, error) {
	req.Timestamp = getTimestamp()
	body := mustMarshalJson(req)
	sign := p.signBody("order_cancel", body)
	fastResp, err := p.httpClient.POST("/order_cancel").
		Header().Add("API-SIGN", string(sign)).
		Body().AsString(string(body)).
		Send()
	if err != nil {
		return nil, err
	}
	if fastResp.Status().IsError() {
		body, _ := fastResp.Body().AsString()
		return nil, errors.New(body)
	}
	var data CancelOrderResponse
	err = fastResp.Body().AsJSON(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// Request Weight: 1 * count of pairs
func (p *Client) Orders(pairs []Pair) (*OrdersResponse, error) {
	req := &OrdersRequest{Pairs: joinPairs(pairs)}
	fastResp, err := p.httpClient.GET("/orders").Body().AsJSON(req).Send()
	if err != nil {
		return nil, err
	}
	if fastResp.Status().IsError() {
		body, _ := fastResp.Body().AsString()
		return nil, errors.New(body)
	}
	var data OrdersResponse
	err = fastResp.Body().AsJSON(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// Request Weight: 1
func (p *Client) Tickers(pairs []Pair) (*TickersResponse, error) {
	req := &TickersRequest{Pairs: joinPairs(pairs)}
	fastResp, err := p.httpClient.GET("/ticker").Body().AsJSON(req).Send()
	if err != nil {
		return nil, err
	}
	if fastResp.Status().IsError() {
		body, _ := fastResp.Body().AsString()
		return nil, errors.New(body)
	}
	var data TickersResponse
	err = fastResp.Body().AsJSON(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (p *Client) signBody(method string, body []byte) string {
	payload := append([]byte(method), body...)
	return signer.Sign(payload, []byte(p.config.Secret))
}

func setupHttpClient(apiId string) fastshot.ClientHttpMethods {
	return fastshot.NewClient(baseUrl).
		Header().Add("API-ID", apiId).
		Header().AddAccept(mime.JSON).
		Build()
}

func joinPairs(pairs []Pair) string {
	joined := strings.Builder{}
	for i, pair := range pairs {
		joined.WriteString(string(pair))
		if i != len(pairs)-1 {
			joined.WriteString(",")
		}
	}
	return joined.String()
}
