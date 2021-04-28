package e2e_testing

import (
	"net/http"
	"net/url"
)

var testServerClient *Client

type Client struct {
	// The base address for making request to the instrumented test server
	Address string

	// Headers to add to all requests, for auth purposes
	Headers http.Header

	client http.Client
}

func (c *Client) Get(path string) (*http.Response, error) {
	url := url.URL{
		Scheme: "http",
		Host:   c.Address,
		Path:   path,
	}
	req, _ := http.NewRequest("GET", url.String(), nil)
	req.Header = c.Headers
	return c.client.Do(req)
}
