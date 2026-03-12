package testclient

import (
	"context"
	"os"
	"strconv"
	"testing"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestClientRequest(t *testing.T) {
	ctx := context.Background()

	srv := pstest.NewServer()
	defer srv.Close()

	os.Setenv("PUBSUB_EMULATOR_HOST", srv.Addr)
	defer os.Unsetenv("PUBSUB_EMULATOR_HOST")

	conn, err := grpc.Dial(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client, err := pubsub.NewClient(ctx, "project", option.WithGRPCConn(conn))
	require.NoError(t, err)
	defer client.Close()

	pubsubInfo := &setuptf.PubsubInfo{
		RequestTopic: setuptf.TopicInfo{
			TopicName: "request-topic",
		},
		ResponseTopic: setuptf.TopicInfo{
			TopicName:        "response-topic",
			SubscriptionName: "response-sub",
		},
	}

	reqTopic, err := client.CreateTopic(ctx, "request-topic")
	require.NoError(t, err)
	reqSub, err := client.CreateSubscription(ctx, "request-sub", pubsub.SubscriptionConfig{
		Topic: reqTopic,
	})
	require.NoError(t, err)

	respTopic, err := client.CreateTopic(ctx, "response-topic")
	require.NoError(t, err)

	_, err = client.CreateSubscription(ctx, "response-sub", pubsub.SubscriptionConfig{
		Topic: respTopic,
	})
	require.NoError(t, err)

	// Mock server answering requests
	go func() {
		err := reqSub.Receive(ctx, func(c context.Context, msg *pubsub.Message) {
			msg.Ack()
			
			// Send response
			res := &pubsub.Message{
				Attributes: map[string]string{
					TestID:     msg.Attributes[TestID],
					StatusCode: strconv.Itoa(int(code.Code_OK)),
					"custom":   "header",
				},
			}
			respTopic.Publish(c, res)
		})
		if err != nil {
			t.Logf("reqSub receive error: %v", err)
		}
	}()

	sut, err := New(ctx, "project", pubsubInfo)
	require.NoError(t, err)

	req := Request{
		TestID:   "test-123",
		Scenario: "my-scenario",
		Headers:  map[string]string{"foo": "bar"},
	}

	// We may need a small timeout for the test client receiving the response, but it blocks
	resp, err := sut.Request(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, code.Code_OK, resp.StatusCode)
	assert.Equal(t, "header", resp.Headers["custom"])
	assert.Equal(t, "test-123", resp.Headers[TestID])
}
