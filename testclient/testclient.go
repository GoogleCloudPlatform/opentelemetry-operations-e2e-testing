package testclient

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/sethvargo/go-retry"
)

const TestID string = "test-id"

type Client struct {
	// The base address for making request to the instrumented test server
	address string

	// Underlying http client for making requests
	httpClient http.Client

	// Option to apply to all requests, for auth purposes. E.g. adding a header
	requestOptions []Option
}

type Option func(*http.Request) *http.Request

func New(address string, requestOptions ...Option) *Client {
	return &Client{address: address, requestOptions: requestOptions}
}

/**
 * Apply passed options + requestOptions from the struct
 */
func (c *Client) addOptions(req *http.Request, options ...Option) *http.Request {
	for _, option := range c.requestOptions {
		req = option(req)
	}
	for _, option := range options {
		req = option(req)
	}
	return req
}

func (c *Client) getUrl(path string) url.URL {
	return url.URL{
		Scheme: "http",
		Host:   c.address,
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
	return c.httpClient.Do(req)
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

// Call in TestMain() to block until the test server is ready for requests. Uses
// a *log.Logger because this runs before testing.T is available
func (c *Client) WaitForHealth(ctx context.Context, logger *log.Logger) error {
	backoff, _ := retry.NewConstant(time.Millisecond * 500)
	backoff = retry.WithMaxDuration(time.Second*10, backoff)
	err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, time.Millisecond*500)
		defer cancel()

		res, err := c.Get(ctx, "/health")
		if err != nil {
			logger.Printf("waiting for instrumented test server /health: %v", err)
			return retry.RetryableError(err)
		}
		if res.StatusCode != 200 {
			err = fmt.Errorf(
				"expected status code 200 from /health, got %v", res.StatusCode,
			)
			logger.Printf("Waiting for instrumented test server /health: %v", err)
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("test server did not respond to health checks: %v", err)
	}
	return nil
}

func WithTestID(testID string) Option {
	return func(req *http.Request) *http.Request {
		req.Header.Add(TestID, testID)
		return req
	}
}
