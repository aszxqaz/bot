package client

type ApiClient interface {
	Balances() []Balance
}
