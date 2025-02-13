//go:build e2ecollector

package e2etestrunner_collector

import (
	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/cloudtrace/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
	"testing"
	"time"
)

const (
	representativeMetric = "workload.googleapis.com/otelcol_exporter_sent_metric_points"
	queryMaxAttempts     = 3
	labelName            = "otelcol_google_e2e"
)

func TestMetrics(t *testing.T) {
	ctx := context.Background()
	c, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	tsList := make([]*monitoringpb.TimeSeries, 0)
	for attempt := 1; attempt <= queryMaxAttempts; attempt++ {
		window := 10 * time.Minute
		now := time.Now()
		start := timestamppb.New(now.Add(-window))
		end := timestamppb.New(now)
		fmt.Println(fmt.Sprintf("resource.type = %q", resourceType))
		filters := []string{
			fmt.Sprintf("metric.type = %q", representativeMetric),
			fmt.Sprintf("resource.type = %q", resourceType),
			fmt.Sprintf("metric.labels.otelcol_google_e2e = %s", args.TestRunID),
		}

		req := &monitoringpb.ListTimeSeriesRequest{
			Name:   "projects/" + args.ProjectID,
			Filter: strings.Join(filters, " AND "),
			Interval: &monitoringpb.TimeInterval{
				EndTime:   end,
				StartTime: start,
			},
			View: monitoringpb.ListTimeSeriesRequest_FULL,
		}

		it := c.ListTimeSeries(ctx, req)
		for {
			series, err := it.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				t.Fatal("Timeseries query failed with error:", err)
			}
			if len(series.Points) == 0 {
				continue
			}
			tsList = append(tsList, series)
		}
		if len(tsList) == 0 {
			continue
		}
		break
	}

	if len(tsList) == 0 {
		t.Fatal(fmt.Errorf("Could not find representative metric: %q", representativeMetric))
	}

	fmt.Printf("Found representative metric: %v\n", tsList[0].Metric)
}

func TestLogging(t *testing.T) {
	ctx := context.Background()
	client, err := logadmin.NewClient(ctx, args.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now().Add(-time.Hour)

	window := start.Format(time.RFC3339)
	filter := fmt.Sprintf(`resource.type="%s" AND labels.%s="%s" AND timestamp > "%s"`, resourceType, labelName, args.TestRunID, window)

	var entry *logging.Entry
	for attempt := 1; attempt <= queryMaxAttempts; attempt++ {
		entries := client.Entries(ctx,
			logadmin.Filter(filter),
		)

		entry, err = entries.Next()
		if err == nil {
			// log found
			break
		}
		time.Sleep(10 * time.Second)
	}

	fmt.Println(fmt.Sprintf("Entry found: %v", entry))
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not any logs matching filter: %s", filter))
	}
}

func TestTraces(t *testing.T) {
	ctx := context.Background()
	cloudService, err := cloudtrace.NewService(ctx)
	if err != nil {
		t.Fatal(err)
	}

	req := cloudService.Projects.Traces.List(args.ProjectID)
	req.Filter(fmt.Sprintf("otelcol_google_e2e:%s", args.TestRunID))
	resp, err := req.Do()
	if err != nil {
		t.Fatal(err)
	}
	tracesFound := len(resp.Traces)
	if tracesFound == 0 {
		t.Errorf(fmt.Sprintf("Could not find traces with resource attribute otelcol_google_e2e:%s", args.TestRunID))
	}
	fmt.Println(fmt.Sprintf("Found traces with otelcol_google_e2e:%s", args.TestRunID))
}
