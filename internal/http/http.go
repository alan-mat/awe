// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

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
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	var resp *gohttp.Response
	for i := range c.maxRetries {
		resp, err = c.httpClient.Do(req)
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

		// truncate error responses
		if len(respBytes) > 512 {
			respBytes = respBytes[:512]
		}

		return nil, fmt.Errorf("(HTTP Error %d) %s", resp.StatusCode, string(respBytes))
	}

	return resp, nil
}
