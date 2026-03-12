package testclient

import (
	"context"
	"os"
	"strconv"
	"sync"
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

			custom := "header"
			if val, ok := msg.Attributes["foo"]; ok {
				custom = val
			}

			// Send response
			res := &pubsub.Message{
				Attributes: map[string]string{
					TestID:     msg.Attributes[TestID],
					StatusCode: strconv.Itoa(int(code.Code_OK)),
					"custom":   custom,
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

	t.Run("single request", func(t *testing.T) {
		req := Request{
			TestID:   "test-123",
			Scenario: "my-scenario",
			Headers:  map[string]string{"foo": "bar"},
		}

		resp, err := sut.Request(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, code.Code_OK, resp.StatusCode)
		assert.Equal(t, "bar", resp.Headers["custom"])
		assert.Equal(t, "test-123", resp.Headers[TestID])
	})

	t.Run("multiplexing", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				req := Request{
					TestID:   "test-" + strconv.Itoa(idx),
					Scenario: "my-scenario-" + strconv.Itoa(idx),
					Headers:  map[string]string{"foo": "bar-" + strconv.Itoa(idx)},
				}

				resp, err := sut.Request(ctx, req)
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, code.Code_OK, resp.StatusCode)
				assert.Equal(t, "bar-"+strconv.Itoa(idx), resp.Headers["custom"])
				assert.Equal(t, "test-"+strconv.Itoa(idx), resp.Headers[TestID])
			}(i)
		}
		wg.Wait()
	})
}
