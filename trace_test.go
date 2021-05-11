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

package e2e_testing

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
	"github.com/sethvargo/go-retry"
	cloudtrace "google.golang.org/api/cloudtrace/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
)

const basicTraceSpanName string = "basicTrace"

func newTraceService(t *testing.T, ctx context.Context) *cloudtrace.Service {
	cloudtraceService, err := cloudtrace.NewService(ctx)
	if err != nil {
		t.Fatalf("Failed to get cloud trace service: %v", err)
	}
	return cloudtraceService
}

// Checks response code for the test server response and fatals or skips the
// test if necessary.
func checkTestScenarioResponse(t *testing.T, scenario string, res *testclient.Response, err error) {
	if err != nil {
		t.Fatalf("test server failed for scenario %v: %v", scenario, err)
	}
	switch res.StatusCode {
	case code.Code_OK:
	case code.Code_UNIMPLEMENTED:
		t.Skipf("test server does not support this scenario, skipping")
	default:
		t.Fatalf(`got unexpected response code "%v" from test server`, res.StatusCode)
	}
}

func listTracesWithRetry(
	ctx context.Context,
	t *testing.T,
	cloudtraceService *cloudtrace.Service,
	startTime time.Time,
	testID string,
) *cloudtrace.ListTracesResponse {
	// Assert response
	startTimeBytes, err := startTime.MarshalText()
	if err != nil {
		t.Fatalf("Couldn't marshal time to text: %v", err)
	}

	var gctRes *cloudtrace.ListTracesResponse
	backoff, _ := retry.NewConstant(1 * time.Second)
	backoff = retry.WithMaxDuration(time.Second*10, backoff)
	err = retry.Do(ctx, backoff, func(ctx context.Context) error {
		gctRes, err = cloudtraceService.Projects.Traces.List(args.ProjectID).
			StartTime(string(startTimeBytes)).
			Filter(fmt.Sprintf("+test-id:%v", testID)).
			View("COMPLETE").
			PageSize(10).
			Do()
		if err != nil {
			return retry.RetryableError(err)
		}
		if len(gctRes.Traces) == 0 {
			err = errors.New("Got zero spans in trace")
			t.Logf("Retrying ListSpans: %v", err)
			return retry.RetryableError(err)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to list traces: %v", err)
	}
	return gctRes
}

func TestBasicTrace(t *testing.T) {
	ctx := context.Background()
	scenario := "/basicTrace"
	startTime := time.Now().Add(time.Second * -5)
	cloudtraceService := newTraceService(t, ctx)
	testID := fmt.Sprint(rand.Uint64())

	// Call test server
	reqCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	res, err := testServerClient.Request(reqCtx, testclient.Request{Scenario: scenario, TestID: testID})
	checkTestScenarioResponse(t, scenario, res, err)

	// Assert response
	gctRes := listTracesWithRetry(ctx, t, cloudtraceService, startTime, testID)
	if numTraces := len(gctRes.Traces); numTraces != 1 {
		t.Fatalf("Got %v traces, expected 1", numTraces)
	}
	if len(gctRes.Traces[0].Spans) == 0 {
		t.Fatalf("Got zero spans in trace %v", gctRes.Traces[0].TraceId)
	}

	span := gctRes.Traces[0].Spans[0]

	if span.Name != basicTraceSpanName {
		t.Fatalf(`Expected span name %v, got "%v"`, basicTraceSpanName, span.Name)
	}

	if numLabels := len(span.Labels); numLabels != 2 {
		t.Fatalf("Expected exactly 2 labels, got %v", numLabels)
	}

	labelCases := []struct {
		expectKey string
		expectRe  string
	}{
		{
			expectKey: "g.co/agent",
			// TODO button this re down more
			expectRe: `opentelemetry-[\w-]+ [\d\.]+; [\w-]+ [\d\.]+`,
		},
		{
			expectKey: testclient.TestID,
			expectRe:  regexp.QuoteMeta(testID),
		},
	}
	for _, tc := range labelCases {
		t.Run(fmt.Sprintf("Span has label %v", tc.expectKey), func(t *testing.T) {
			val, ok := span.Labels[tc.expectKey]
			if !ok {
				t.Fatalf(`Missing label "%v"`, tc.expectKey)
			}
			match, _ := regexp.MatchString(tc.expectRe, val)
			if !match {
				t.Fatalf(
					`For label key %v, value "%v" did not match regex "%v"`,
					tc.expectKey,
					val,
					tc.expectRe,
				)
			}
		})
	}
}

func TestComplexTrace(t *testing.T) {
	ctx := context.Background()
	scenario := "/complexTrace"
	startTime := time.Now().Add(time.Second * -5)
	cloudtraceService := newTraceService(t, ctx)
	testID := fmt.Sprint(rand.Uint64())

	// Call test server
	reqCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	res, err := testServerClient.Request(reqCtx, testclient.Request{Scenario: scenario, TestID: testID})
	checkTestScenarioResponse(t, scenario, res, err)

	// Assert response
	gctRes := listTracesWithRetry(ctx, t, cloudtraceService, startTime, testID)
	if numTraces := len(gctRes.Traces); numTraces != 1 {
		t.Fatalf("Got %v traces, expected 1", numTraces)
	}
	if len(gctRes.Traces[0].Spans) == 4 {
		t.Fatalf("Got zero spans in trace %v, expected 4", gctRes.Traces[0].TraceId)
	}

	// ... finish implementing
}
