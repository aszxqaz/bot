package payeer

import fastshot "github.com/opus-domini/fast-shot"

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
