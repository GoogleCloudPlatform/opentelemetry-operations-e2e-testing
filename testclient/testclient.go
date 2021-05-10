// Copyright 2021 Google
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
	"log"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/setuptf"
)

const (
	TestID   string = "test-id"
	Scenario string = "scenario"
	Status   string = "status"
	Health   string = "/health"
)

type Request struct {
	// name of the scenario to run
	Scenario string
	TestID   string
}

type Response struct {
	Status string `json:"status"`
}

type Option func(*http.Request) *http.Request

type Client struct {
	pubsubClient         *pubsub.Client
	requestTopic         *pubsub.Topic
	responseSubscription *pubsub.Subscription
}

func New(ctx context.Context, projectID string, pubsubInfo *setuptf.PubsubInfo) (*Client, error) {
	pubsub, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &Client{
		pubsubClient:         pubsub,
		requestTopic:         pubsub.Topic(pubsubInfo.RequestTopic.TopicName),
		responseSubscription: pubsub.Subscription(pubsubInfo.ResponseTopic.SubscriptionName),
	}, nil
}

func (c *Client) Request(
	ctx context.Context,
	request Request,
) (*Response, error) {
	pubResult := c.requestTopic.Publish(ctx, &pubsub.Message{
		Attributes: map[string]string{TestID: request.TestID, "scenario": request.Scenario},
	})
	messageID, err := pubResult.Get(ctx)
	if err != nil {
		return nil, err
	}

	var res *Response
	cctx, cancel := context.WithCancel(ctx)
	err = c.responseSubscription.Receive(cctx, func(ctx context.Context, message *pubsub.Message) {
		if testID := message.Attributes[TestID]; testID == request.TestID {
			message.Ack()
			res = &Response{Status: message.Attributes[Status]}
			cancel()
		}
		message.Nack()
	})
	if res == nil {
		err = fmt.Errorf(
			"sent message ID %v, but never received a response on subscription %v",
			messageID,
			c.responseSubscription.String(),
		)
	}
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Call in TestMain() to block until the test server is ready for requests. Uses
// a *log.Logger because this runs before testing.T is available
func (c *Client) WaitForHealth(ctx context.Context, logger *log.Logger) error {
	_, err := c.Request(ctx, Request{Scenario: Health})
	return err
}
