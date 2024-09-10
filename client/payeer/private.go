package payeer

import (
	"automata/signer"
	"errors"
)

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

// Request Weight: 60
func (p *Client) MyOrders(req *MyOrdersRequest) (*MyOrdersResponse, error) {
	req.Timestamp = getTimestamp()
	body := mustMarshalJson(req)
	sign := p.signBody("my_orders", body)
	fastResp, err := p.httpClient.POST("/my_orders").
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
	var empty MyOrdersEmptyResponse
	err = fastResp.Body().AsJSON(&empty)
	if err != nil {
		var data MyOrdersResponse
		err = fastResp.Body().AsJSON(&data)
		if err != nil {
			return nil, err
		}
		return &data, nil
	}
	return &MyOrdersResponse{
		BaseResponse: BaseResponse{Success: true},
		Orders:       make(map[string]MyOrdersOrder),
	}, nil
}

func (p *Client) signBody(method string, body []byte) string {
	payload := append([]byte(method), body...)
	return signer.Sign(payload, []byte(p.config.Secret))
}
