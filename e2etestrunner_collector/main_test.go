//go:build e2ecollector

package e2etestrunner_collector

import (
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
)

const labelName = "otelcol_google_e2e"

var (
	args           e2etesting.Args
	resourceType   string
	resourceFilter string
)

func TestMain(m *testing.M) {
	logger, ctx := e2etesting.InitTestMain(&args, setuptf.ApplyPersistentCollector)
	resourceFilter = fmt.Sprintf("otelcol_google_e2e:%s", args.TestRunID)
	var setupFunc e2etesting.SetupCollectorFunc
	switch {
	case args.GceCollector != nil:
		setupFunc = SetupGceCollector
		resourceType = "gce_instance"
	case args.GceCollectorArm != nil:
		setupFunc = SetupGceCollectorArm
		resourceType = "gce_instance"
	case args.GkeCollector != nil:
		setupFunc = SetupGkeCollector
		resourceType = "k8s_container"
	case args.GkeOperatorCollector != nil:
		setupFunc = SetupGkeOperatorCollector
		resourceType = "k8s_container"
	case args.CloudRunCollector != nil:
		setupFunc = SetupCloudRunCollector
		resourceType = "generic_task"
	}
	cleanup, err := setupFunc(ctx, &args, logger)

	defer cleanup()
	if err != nil {
		logger.Panic(err)
	}

	// wait for instrumented test server to be healthy
	logger.Printf("Waiting for health check on (will timeout after %v)\n", args.HealthCheckTimeout)
	time.Sleep(args.HealthCheckTimeout)
	// Run tests
	logger.Print(e2etesting.BeginOutputArt)
	m.Run()
	logger.Print(e2etesting.EndOutputArt)
}
