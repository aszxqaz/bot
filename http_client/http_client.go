package httpclient

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
)

type HttpClient struct {
	client  *http.Client
	baseUrl string
	headers http.Header
}

func NewHttpClient(baseUrl string) *HttpClient {
	return &HttpClient{client: &http.Client{}, baseUrl: baseUrl}
}

func (c *HttpClient) SetHeaders(headers http.Header) {
	c.headers = headers
}

func (c *HttpClient) Post(url string, data any) error {
	return c.do("POST", url, data)
}

func (c *HttpClient) Put(url string, data any) error {
	return c.do("PUT", url, data)
}

func (c *HttpClient) Get(url string, data any) error {
	return c.do("GET", url, data)
}

func (c *HttpClient) Delete(url string, data any) error {
	return c.do("DELETE", url, data)
}

func (c *HttpClient) do(method string, path string, data any) error {
	curl, err := url.Parse(c.baseUrl + path)
	if err != nil {
		log.Println("[HttpClient] Failed to parse url", err)
		return err
	}
	req := &http.Request{
		Method: method,
		URL:    curl,
		Header: c.headers,
	}
	// log.Printf("[HttpClient] Making %s request to %s", method, curl.String())
	resp, err := c.client.Do(req)
	if err != nil {
		log.Println("[HttpClient] Failed to send request", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 429 || resp.StatusCode == 403 {
			log.Fatal("[HttpClient] FATAL: ", string(body))
		}
		return errors.New(string(body))
	}

	if data == nil {
		return nil
	}
	return c.readJson(resp.Body, data)
}

func (c *HttpClient) readJson(r io.Reader, data any) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	// log.Println("[HttpClient] Received response: " + string(body))
	// decoder := json.NewDecoder(r)
	// err := decoder.Decode(data)
	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}

	return err
}
