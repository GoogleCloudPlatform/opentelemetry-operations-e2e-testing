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

package e2e_testing

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
	"github.com/sethvargo/go-retry"
	cloudtrace "google.golang.org/api/cloudtrace/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
)

const (
	basicTraceSpanName      string = "basicTrace"
	basicPropagatorSpanName string = "basicPropagator"
	rpcClient               string = "RPC_CLIENT"
	rpcServer               string = "RPC_SERVER"
	xCloudTraceContextName  string = "X-Cloud-Trace-Context"
)

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
	require.NoErrorf(t, err, "test server failed for scenario %v: %v", scenario, err)

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
	require.NoErrorf(t, err, "Couldn't marshal time to text: %v", err)

	var gctRes *cloudtrace.ListTracesResponse
	backoff, _ := retry.NewConstant(1 * time.Second)
	backoff = retry.WithMaxDuration(time.Second*10, backoff)
	err = retry.Do(ctx, backoff, func(ctx context.Context) error {
		gctRes, err = cloudtraceService.Projects.Traces.List(args.ProjectID).
			StartTime(string(startTimeBytes)).
			Filter(fmt.Sprintf("+%v:%v", testclient.TestID, testID)).
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

	require.NoErrorf(t, err, "Failed to list traces: %v", err)
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
	res, err := testServerClient.Request(
		reqCtx,
		testclient.Request{Scenario: scenario, TestID: testID},
	)
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
	require.Equalf(
		t,
		span.Name,
		basicTraceSpanName,
		`Expected span name %v, got "%v"`,
		basicTraceSpanName,
		span.Name,
	)

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
			assert.Truef(t, ok, `Missing label "%v"`, tc.expectKey)
			assert.Regexpf(
				t,
				tc.expectRe,
				val,
				`For label key %v, value "%v" did not match regex "%v"`,
				tc.expectKey,
				val,
				tc.expectRe,
			)
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
	res, err := testServerClient.Request(
		reqCtx,
		testclient.Request{Scenario: scenario, TestID: testID},
	)
	checkTestScenarioResponse(t, scenario, res, err)

	// Assert response
	gctRes := listTracesWithRetry(ctx, t, cloudtraceService, startTime, testID)
	if numTraces := len(gctRes.Traces); numTraces != 1 {
		t.Fatalf("Got %v traces, expected 1", numTraces)
	}
	trace := gctRes.Traces[0]
	if numSpans := len(trace.Spans); numSpans != 4 {
		t.Fatalf("Got %v spans in trace %v, but expected 4", numSpans, trace.TraceId)
	}

	spanByName := make(map[string]*cloudtrace.TraceSpan)
	for _, span := range trace.Spans {
		spanByName[span.Name] = span
	}

	cases := []struct {
		name             string
		expectKind       string
		expectParentName string
	}{
		{
			name: "complexTrace/root",
		},
		{
			name:             "complexTrace/child1",
			expectKind:       rpcServer,
			expectParentName: "complexTrace/root",
		},
		{
			name:             "complexTrace/child2",
			expectKind:       rpcClient,
			expectParentName: "complexTrace/child1",
		},
		{
			name:             "complexTrace/child3",
			expectKind:       "",
			expectParentName: "complexTrace/root",
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("assert structure of span %v", tc.name), func(t *testing.T) {
			span := spanByName[tc.name]
			assert.NotNilf(t, span, "Missing span named %v", tc.name)

			if tc.expectParentName == "" {
				assert.EqualValuesf(
					t,
					0,
					span.ParentSpanId,
					"Expected no parent, but got %v",
					span.ParentSpanId,
				)
			} else {
				parentSpan := spanByName[tc.expectParentName]
				if parentSpan == nil {
					t.Errorf("The parent span %v does not exist in the trace", tc.expectParentName)
				} else if parentSpan.SpanId != span.ParentSpanId {
					t.Errorf("Expected parent span ID %v, but got %v", parentSpan.SpanId, span.ParentSpanId)
				}
			}
			if tc.expectKind != "" && span.Kind != tc.expectKind {
				t.Errorf("Expected span kind %v, but got %v", tc.expectKind, span.Kind)
			}
		})
	}
}

func TestBasicPropagator(t *testing.T) {
	ctx := context.Background()
	scenario := "/basicPropagator"
	startTime := time.Now().Add(time.Second * -5)
	cloudtraceService := newTraceService(t, ctx)
	testID := fmt.Sprint(rand.Uint64())

	// Generate random trace and span IDs
	traceIdHex, err := randomHex(16)
	require.NoErrorf(t, err, "test server failed for scenario %v: %v", scenario, err)
	parentSpanIdDec := rand.Uint64()
	xCloudTraceContext := fmt.Sprintf("%v/%v;o=1", traceIdHex, parentSpanIdDec)

	// Call test server
	reqCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	res, err := testServerClient.Request(
		reqCtx,
		testclient.Request{
			Scenario: scenario,
			TestID:   testID,
			Headers:  map[string]string{xCloudTraceContextName: xCloudTraceContext},
		},
	)
	checkTestScenarioResponse(t, scenario, res, err)

	// Assert response
	gctRes := listTracesWithRetry(ctx, t, cloudtraceService, startTime, testID)

	if numTraces := len(gctRes.Traces); numTraces != 1 {
		t.Fatalf("Got %v traces, expected 1", numTraces)
	}
	if len(gctRes.Traces[0].Spans) == 0 {
		t.Fatalf("Got zero spans in trace %v", gctRes.Traces[0].TraceId)
	}

	trace := gctRes.Traces[0]
	span := trace.Spans[0]
	require.Equalf(
		t,
		span.Name,
		basicPropagatorSpanName,
		`Expected span name %v, got "%v"`,
		basicPropagatorSpanName,
		span.Name,
	)
	require.Equalf(
		t,
		traceIdHex,
		trace.TraceId,
		`Expected trace ID %v, got "%v"`,
		traceIdHex,
		trace.TraceId,
	)
	require.Equalf(
		t,
		parentSpanIdDec,
		span.ParentSpanId,
		`Expected parent span ID %v, got "%v"`,
		parentSpanIdDec,
		span.ParentSpanId,
	)
}
