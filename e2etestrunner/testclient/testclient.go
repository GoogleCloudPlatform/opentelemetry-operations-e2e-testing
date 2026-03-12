// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testclient

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
	"log"
	"strconv"
	"sync"

	"cloud.google.com/go/pubsub"
	"google.golang.org/genproto/googleapis/rpc/code"
)

const (
	TestID     string = "test_id"
	Scenario   string = "scenario"
	StatusCode string = "status_code"
	Health     string = "/health"
)

type Request struct {
	// name of the scenario to run
	Scenario string
	TestID   string
	Headers  map[string]string
}

type Response struct {
	StatusCode code.Code
	Headers    map[string]string
}

type Client struct {
	pubsubClient         *pubsub.Client
	requestTopic         *pubsub.Topic
	responseSubscription *pubsub.Subscription

	mu              sync.Mutex
	pendingRequests map[string]chan asyncResponse
}

type asyncResponse struct {
	res *Response
	err error
}

func New(ctx context.Context, projectID string, pubsubInfo *setuptf.PubsubInfo) (*Client, error) {
	pubsub, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	client := &Client{
		pubsubClient:         pubsub,
		requestTopic:         pubsub.Topic(pubsubInfo.RequestTopic.TopicName),
		responseSubscription: pubsub.Subscription(pubsubInfo.ResponseTopic.SubscriptionName),
		pendingRequests:      make(map[string]chan asyncResponse),
	}
	// Disable buffering
	client.requestTopic.PublishSettings.CountThreshold = 1

	go client.startReceiver(ctx)

	return client, nil
}

func (c *Client) startReceiver(ctx context.Context) {
	err := c.responseSubscription.Receive(ctx, func(ctx context.Context, message *pubsub.Message) {
		testID := message.Attributes[TestID]

		c.mu.Lock()
		ch, ok := c.pendingRequests[testID]
		c.mu.Unlock()

		if ok {
			message.Ack()
			var resErr error
			var res *Response
			codeInt, err := strconv.Atoi(message.Attributes[StatusCode])
			if err != nil {
				resErr = fmt.Errorf(`response pub/sub message invalid attribute %q: %v, message: %v`, StatusCode, err, message)
			} else {
				res = &Response{StatusCode: code.Code(codeInt), Headers: message.Attributes}
			}
			ch <- asyncResponse{res: res, err: resErr}
		} else {
			message.Nack()
		}
	})
	if err != nil {
		log.Printf("Background subscriber error: %v", err)
	}
}

func (c *Client) Request(
	ctx context.Context,
	request Request,
) (*Response, error) {
	attributes := map[string]string{TestID: request.TestID, "scenario": request.Scenario}
	for k, v := range request.Headers {
		attributes[k] = v
	}

	resCh := make(chan asyncResponse, 1)
	c.mu.Lock()
	c.pendingRequests[request.TestID] = resCh
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pendingRequests, request.TestID)
		c.mu.Unlock()
	}()

	pubResult := c.requestTopic.Publish(ctx, &pubsub.Message{
		Attributes: attributes,
	})
	messageID, err := pubResult.Get(ctx)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf(
			"sent message ID %v, but never received a response on subscription %v: %w",
			messageID,
			c.responseSubscription.String(),
			ctx.Err(),
		)
	case asyncRes := <-resCh:
		return asyncRes.res, asyncRes.err
	}
}

// Call in TestMain() to block until the test server is ready for requests. Uses
// a *log.Logger because this runs before testing.T is available
func (c *Client) WaitForHealth(ctx context.Context, logger *log.Logger) error {
	_, err := c.Request(ctx, Request{Scenario: Health})
	return err
}
