// Copyright 2024 Google LLC
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

package quickstarttest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
)

type MockT struct {
	Failed  bool
	Message string
}

func (t *MockT) Errorf(format string, args ...any) {
	t.Failed = true
	t.Message = fmt.Sprintf(format, args...)
}

func TestVerifyPromMetric(t *testing.T) {
	tcs := []struct {
		name       string
		textFormat string
		testCase   testCase
		expectFail bool
	}{
		{
			name: "metric is above threshold pass",
			textFormat: `
			# HELP otelcol_exporter_sent_log_records Number of log record successfully sent to destination.
			# TYPE otelcol_exporter_sent_log_records counter
			otelcol_exporter_sent_log_records{exporter="googlecloud"} 631
			`,
			testCase: testCase{
				exporter:   "googlecloud",
				metricName: "otelcol_exporter_sent_log_records",
				threshold:  100,
			},
		},
		{
			name: "metric is below threshold fail",
			textFormat: `
			# HELP otelcol_exporter_sent_log_records Number of log record successfully sent to destination.
			# TYPE otelcol_exporter_sent_log_records counter
			otelcol_exporter_sent_log_records{exporter="googlecloud"} 1
			`,
			expectFail: true,
			testCase: testCase{
				exporter:   "googlecloud",
				metricName: "otelcol_exporter_sent_log_records",
				threshold:  100,
			},
		},
		{
			name:       "metric is not present fail",
			textFormat: ``,
			testCase: testCase{
				exporter:   "googlecloud",
				metricName: "otelcol_exporter_sent_log_records",
				threshold:  100,
			},
			expectFail: true,
		},
		{
			name: "exporter is not present fail",
			textFormat: `
			# HELP otelcol_exporter_sent_log_records Number of log record successfully sent to destination.
			# TYPE otelcol_exporter_sent_log_records counter
			otelcol_exporter_sent_log_records{exporter="googlecloud"} 631
			`,
			expectFail: true,
			testCase: testCase{
				exporter:   "fooexporter",
				metricName: "otelcol_exporter_sent_log_records",
				threshold:  100,
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			mockT := &MockT{}
			var parser expfmt.TextParser
			actual, err := parser.TextToMetricFamilies(strings.NewReader(tc.textFormat))
			require.NoError(t, err)
			verifyPromMetric(mockT, actual, tc.testCase)

			if tc.expectFail {
				require.True(t, mockT.Failed, "Expected test case to fail but passed")
			} else {
				require.Falsef(t, mockT.Failed, "Expected test case to pass but failed with: %v", mockT.Message)
			}
		})
	}
}

func TestVerifyPromRealTestCasesSuccess(t *testing.T) {
	var parser expfmt.TextParser
	// Taken from a real run of the quickstart
	actual, err := parser.TextToMetricFamilies(strings.NewReader(`
	# HELP otelcol_exporter_sent_log_records Number of log record successfully sent to destination.
	# TYPE otelcol_exporter_sent_log_records counter
	otelcol_exporter_sent_log_records{exporter="googlecloud",service_instance_id="cc2396b4-e313-4c5a-8c35-0cb221a02fa8",service_name="otelcol-contrib",service_version="0.107.0"} 437
	# HELP otelcol_exporter_sent_metric_points Number of metric points successfully sent to destination.
	# TYPE otelcol_exporter_sent_metric_points counter
	otelcol_exporter_sent_metric_points{exporter="googlemanagedprometheus",service_instance_id="cc2396b4-e313-4c5a-8c35-0cb221a02fa8",service_name="otelcol-contrib",service_version="0.107.0"} 333
	# HELP otelcol_exporter_sent_spans Number of spans successfully sent to destination.
	# TYPE otelcol_exporter_sent_spans counter
	otelcol_exporter_sent_spans{exporter="googlecloud",service_instance_id="cc2396b4-e313-4c5a-8c35-0cb221a02fa8",service_name="otelcol-contrib",service_version="0.107.0"} 499
	`))
	require.NoError(t, err)

	for _, tc := range testCases {
		verifyPromMetric(t, actual, tc)
	}
}

func TestVerifyPromRealTestCasesFails(t *testing.T) {
	var parser expfmt.TextParser
	// Taken from a real run of the quickstart
	actual, err := parser.TextToMetricFamilies(strings.NewReader(`
	# HELP otelcol_exporter_sent_log_records Number of log record successfully sent to destination.
	# TYPE otelcol_exporter_sent_log_records counter
	otelcol_exporter_sent_log_records{exporter="googlecloud",service_instance_id="cc2396b4-e313-4c5a-8c35-0cb221a02fa8",service_name="otelcol-contrib",service_version="0.107.0"} 0
	# HELP otelcol_exporter_sent_metric_points Number of metric points successfully sent to destination.
	# TYPE otelcol_exporter_sent_metric_points counter
	otelcol_exporter_sent_metric_points{exporter="googlemanagedprometheus",service_instance_id="cc2396b4-e313-4c5a-8c35-0cb221a02fa8",service_name="otelcol-contrib",service_version="0.107.0"} 0
	# HELP otelcol_exporter_sent_spans Number of spans successfully sent to destination.
	# TYPE otelcol_exporter_sent_spans counter
	otelcol_exporter_sent_spans{exporter="googlecloud",service_instance_id="cc2396b4-e313-4c5a-8c35-0cb221a02fa8",service_name="otelcol-contrib",service_version="0.107.0"} 0
	`))
	require.NoError(t, err)

	mockT := &MockT{}
	for _, tc := range testCases {
		verifyPromMetric(mockT, actual, tc)
	}
	require.True(t, mockT.Failed, "Expected test case to fail but passed")
}
