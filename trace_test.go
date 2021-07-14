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
	basicPropagatorSpanName string = "basicPropagator"
	basicTraceSpanName      string = "basicTrace"
	rpcClient               string = "RPC_CLIENT"
	rpcServer               string = "RPC_SERVER"
	traceIdKey              string = "trace_id"
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

func getTraceWithRetry(
	ctx context.Context,
	t *testing.T,
	cloudtraceService *cloudtrace.Service,
	traceId string,
) *cloudtrace.Trace {
	var trace *cloudtrace.Trace
	backoff, _ := retry.NewConstant(1 * time.Second)
	backoff = retry.WithMaxDuration(time.Second*10, backoff)
	err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		var err error
		trace, err = cloudtraceService.Projects.Traces.Get(args.ProjectID, traceId).Context(ctx).Do()
		if err != nil {
			t.Logf("Retrying GetTrace: %v", err)
			return retry.RetryableError(err)
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, trace)
	return trace
}

func TestBasicTrace(t *testing.T) {
	ctx := context.Background()
	scenario := "/basicTrace"
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
	traceId := res.Headers[traceIdKey]
	require.NotEmptyf(t, traceId, "Expected header %q but it was missing", traceIdKey)
	trace := getTraceWithRetry(ctx, t, cloudtraceService, traceId)

	// Assert response
	if len(trace.Spans) == 0 {
		t.Fatalf("Got zero spans in trace %v", trace.TraceId)
	}

	span := trace.Spans[0]
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
			expectRe: `opentelemetry-\S+ \S+; google-cloud-trace-exporter \S+`,
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
	traceId := res.Headers[traceIdKey]
	require.NotEmptyf(t, traceId, "Expected header %q but it was missing", traceIdKey)
	trace := getTraceWithRetry(ctx, t, cloudtraceService, traceId)
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
	trace := getTraceWithRetry(ctx, t, cloudtraceService, traceIdHex)

	if len(trace.Spans) == 0 {
		t.Fatalf("Got zero spans in trace %v", trace.TraceId)
	}

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
