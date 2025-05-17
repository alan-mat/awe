package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	gohttp "net/http"
	"net/url"
	"time"
)

// Common HTTP method, as defined in net/http package
const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH" // RFC 5789
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

var retryStatusCodes = map[int]bool{
	429: true,
	500: true,
	502: true,
	503: true,
	504: true,
}

type Client struct {
	httpClient *gohttp.Client
	maxRetries int

	endpoint string
	apiKey   string
}

type ClientOption func(*Client)

func NewClient(endpoint string, opts ...ClientOption) Client {
	c := Client{
		endpoint: endpoint,
		httpClient: &gohttp.Client{
			Timeout: 60 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(&c)
	}

	return c
}

func WithApiKey(key string) ClientOption {
	return func(c *Client) {
		c.apiKey = key
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

func (c *Client) Request(method string, path string, paylaod map[string]any) (map[string]any, error) {
	resp, err := c.do(method, path, paylaod)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) RequestStream(method string, path string, paylaod map[string]any) (io.ReadCloser, error) {
	resp, err := c.do(method, path, paylaod)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c *Client) do(method string, path string, paylaod map[string]any) (*gohttp.Response, error) {
	uri, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, err
	}
	uri.Path = path

	jsonData, _ := json.Marshal(paylaod)
	req, err := gohttp.NewRequest(method, uri.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer"+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	var resp *gohttp.Response
	for i := range c.maxRetries {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if i == c.maxRetries-1 {
				return nil, err
			}
			continue
		}

		if _, ok := retryStatusCodes[resp.StatusCode]; ok {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}
		break
	}

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("(HTTP Error %d) %s", resp.StatusCode, string(respBytes))
	}

	return resp, nil
}
