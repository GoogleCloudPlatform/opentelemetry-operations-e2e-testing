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
	"fmt"
	"math/rand"
	"testing"
	"time"

	cloudtrace "google.golang.org/api/cloudtrace/v1"
)

const basicTraceSpanName string = "basicTrace"

func newTraceService(t *testing.T, ctx context.Context) *cloudtrace.Service {
	cloudtraceService, err := cloudtrace.NewService(ctx)
	if err != nil {
		t.Fatalf("Failed to get cloud trace service: %v", err)
	}
	return cloudtraceService
}

/**
 * Just makes a request to ListTraces and asserts there was no error
 */
func TestQueryRecentTraces(t *testing.T) {
	ctx := context.Background()
	startTime := time.Now().Add(time.Second * -5)
	cloudtraceService := newTraceService(t, ctx)
	testID := fmt.Sprint(rand.Uint64())

	// Call test server
	res, err := testServerClient.Post("/basicTrace", nil, WithTestID(testID))
	if err != nil {
		t.Fatalf("Couldn't call test server: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected response code 200, got %v", res.StatusCode)
	}

	// Assert response
	startTimeBytes, err := startTime.MarshalText()
	if err != nil {
		t.Fatalf("Couldn't marshal time to text: %v", err)
	}
	fmt.Printf("startTime: %v\n", string(startTimeBytes))
	gctRes, err := cloudtraceService.Projects.Traces.List(args.ProjectID).
		StartTime(string(startTimeBytes)).
		Filter(fmt.Sprintf("+test-id:%v", testID)).
		View("COMPLETE").
		PageSize(10).
		Do()

	if err != nil {
		t.Fatalf("Failed to list traces: %v", err)
	}
	if len(gctRes.Traces) == 0 {
		t.Fatal("Got zero traces, expected 1")
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
		expectVal string
	}{
		{
			expectKey: "g.co/agent",
			expectVal: "opentelemetry-python 0.170; google-cloud-trace-exporter 0.170",
		},
		{
			expectKey: TestID,
			expectVal: testID,
		},
	}
	for _, tc := range labelCases {
		t.Run(fmt.Sprintf("Span has label %v", tc.expectKey), func(t *testing.T) {
			val, ok := span.Labels[tc.expectKey]
			if !ok {
				t.Fatalf(`Missing label "%v"`, tc.expectKey)
			}
			if val != tc.expectVal {
				t.Fatalf(`For label key %v, expected "%v" but got "%v"`, tc.expectKey, tc.expectVal, val)
			}
		})
	}
}
