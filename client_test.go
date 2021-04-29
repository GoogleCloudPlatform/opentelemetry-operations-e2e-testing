package e2e_testing

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

const TestID string = "test-id"

var testServerClient *Client

type Client struct {
	// The base address for making request to the instrumented test server
	Address string

	// Option to apply to all requests, for auth purposes. E.g. adding a header
	AuthOption Option
	Headers    http.Header

	client http.Client
}

type Option func(*http.Request) *http.Request

/**
 * Apply passed options + AuthOption from the struct
 */
func (c *Client) addOptions(req *http.Request, options ...Option) *http.Request {
	if c.AuthOption != nil {
		req = c.AuthOption(req)
	}
	for _, option := range options {
		req = option(req)
	}
	return req
}

func (c *Client) getUrl(path string) url.URL {
	return url.URL{
		Scheme: "http",
		Host:   c.Address,
		Path:   path,
	}
}

func (c *Client) request(
	ctx context.Context,
	method string,
	path string,
	body io.Reader,
	options ...Option,
) (*http.Response, error) {
	url := c.getUrl(path)
	req, _ := http.NewRequestWithContext(ctx, method, url.String(), body)
	req = c.addOptions(req, options...)
	return c.client.Do(req)
}

func (c *Client) Get(ctx context.Context, path string, options ...Option) (*http.Response, error) {
	return c.request(ctx, http.MethodGet, path, nil, options...)
}

func (c *Client) Post(
	ctx context.Context,
	path string,
	body io.Reader,
	options ...Option,
) (*http.Response, error) {
	return c.request(ctx, http.MethodPost, path, body, options...)
}

func WithTestID(testID string) Option {
	return func(req *http.Request) *http.Request {
		req.Header.Add(TestID, testID)
		return req
	}
}
