package payeer

import (
	"errors"
)

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

// Request Weight: 1 * count of pairs
func (p *Client) Trades(pairs []Pair) (*TradesResponse, error) {
	req := &TradesRequest{Pairs: joinPairs(pairs)}
	fastResp, err := p.httpClient.GET("/trades").Body().AsJSON(req).Send()
	if err != nil {
		return nil, err
	}
	if fastResp.Status().IsError() {
		body, _ := fastResp.Body().AsString()
		return nil, errors.New(body)
	}
	var data TradesResponse
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
