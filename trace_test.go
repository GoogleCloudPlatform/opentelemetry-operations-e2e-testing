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
	"testing"
	"time"

	cloudtrace "google.golang.org/api/cloudtrace/v1"
)

const (
	projectID        = "otel-starter-project"
	expectedSpanName = "integration_test_span"
)

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
	cloudtraceService := newTraceService(t, ctx)
	startTime, _ := time.Now().Add(time.Second * -30).MarshalText()

	_, err := cloudtraceService.Projects.Traces.List(projectID).
		StartTime(string(startTime)).
		View("COMPLETE").
		PageSize(10).
		Do()

	if err != nil {
		t.Fatalf("Failed to make request, %v", err)
	}
}
