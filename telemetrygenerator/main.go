package main

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/telemetrygenerator/provider"
	"github.com/cenkalti/backoff/v4"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"
)

const (
	OTEL_OTLP_ENDPOINT_ENV    = "OTEL_OTLP_ENDPOINT"
	HEALTH_CHECK_ENDPOINT_ENV = "HEALTH_CHECK_ENDPOINT"
	HEALTH_CHECK_PATH_ENV     = "HEALTH_CHECK_PATH"
	MAX_TIME_CONTEXT          = 60 * time.Minute
)

func main() {
	otlpEndpoint := os.Getenv(OTEL_OTLP_ENDPOINT_ENV)
	healthEndpoint := os.Getenv(HEALTH_CHECK_ENDPOINT_ENV)
	healthCheckPath := os.Getenv(HEALTH_CHECK_PATH_ENV)
	if otlpEndpoint == "" {
		log.Fatal(fmt.Sprintf("%s needs to be set", OTEL_OTLP_ENDPOINT_ENV))
	}
	if healthEndpoint == "" {
		log.Fatal(fmt.Sprintf("%s needs to be set", HEALTH_CHECK_ENDPOINT_ENV))
	}
	if healthCheckPath == "" {
		log.Fatal(fmt.Sprintf("%s needs to be set", HEALTH_CHECK_PATH_ENV))
	}

	runCtx, cancelRunning := context.WithTimeout(context.Background(), MAX_TIME_CONTEXT)

	// Gracefully close on SIGINT
	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt)
	go func() {
		<-interrupted
		log.Println("Received interrupt signal, shutting down.")
		cancelRunning()
	}()

	healthCheck := func() error {
		resp, err := http.Get(fmt.Sprintf("%s%s", healthEndpoint, healthCheckPath))
		if err != nil {
			return err
		}
		match, _ := regexp.MatchString("\\b(?:4[0-9]{2}|5[0-4][0-9]|550)\\b", resp.Status)
		if match {
			return fmt.Errorf("health check response: %s", resp.Status)
		}
		return nil
	}

	if err := backoff.Retry(healthCheck, backoff.WithContext(backoff.NewConstantBackOff(MAX_TIME_CONTEXT), runCtx)); err != nil {
		log.Fatal(fmt.Errorf("could not establish health check, %v", err))
	}

	provider.GenerateMetrics(runCtx, otlpEndpoint)
}
