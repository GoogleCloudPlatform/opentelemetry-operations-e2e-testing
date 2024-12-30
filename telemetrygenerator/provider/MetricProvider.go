package provider

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

var renameMap = map[string]string{
	"otlp.test.prefix1": "workload.googleapis.com/otlp.test.prefix1",
	"otlp.test.prefix2": ".invalid.googleapis.com/otlp.test.prefix2",
	"abc":               "otlp.test.prefix3/workload.googleapis.com/abc",
	"otlp.test.prefix4": "WORKLOAD.GOOGLEAPIS.COM/otlp.test.prefix4",
	"otlp.test.prefix5": "WORKLOAD.googleapis.com/otlp.test.prefix5",
}

func installMetricExportPipeline(ctx context.Context, otlpEndpoint string) (func(context.Context) error, error) {
	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(otlpEndpoint))
	if err != nil {
		log.Fatal(err)
	}
	metricProvider := metricsdk.NewMeterProvider(
		metricsdk.WithReader(metricsdk.NewPeriodicReader(exporter)),
		metricsdk.WithResource(resource.Default()),
		metricsdk.WithView(func(i metricsdk.Instrument) (metricsdk.Stream, bool) {
			s := metricsdk.Stream{Name: i.Name, Description: i.Description, Unit: i.Unit}
			newName, ok := renameMap[i.Name]
			if !ok {
				return s, false
			}
			s.Name = newName
			return s, true
		}),
	)
	otel.SetMeterProvider(metricProvider)
	return metricProvider.Shutdown, nil
}

func GenerateMetrics(ctx context.Context, otlpEndpoint string) {
	shutdown, err := installMetricExportPipeline(ctx, otlpEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	meter := otel.GetMeterProvider().Meter("foo")

	// Test cumulative metrics
	counter, err := meter.Float64Counter("otlp.test.cumulative")
	if err != nil {
		log.Fatal(err)
	}
	// Counters need two samples with a short delay in between
	counter.Add(ctx, 5)
	time.Sleep(1 * time.Second)
	counter.Add(ctx, 10)
}
