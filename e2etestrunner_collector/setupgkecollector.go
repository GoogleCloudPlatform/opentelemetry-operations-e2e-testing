package e2etestrunner_collector

import (
	"context"
	"log"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
)

const gkeCollectorTfDir string = "tf/gke-collector"

// SetupGkeCollector Set up the collector to run in GKE.
// Creates a new pod and runs the specified container image in a pod.
// The returned cleanup function tears down the whole cluster.
func SetupGkeCollector(
	ctx context.Context,
	args *e2etesting.Args,
	logger *log.Logger,
) (e2etesting.Cleanup, error) {
	_, cleanupTf, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		gkeCollectorTfDir,
		map[string]string{
			"image": args.GkeCollector.Image,
		},
		logger,
	)

	return cleanupTf, err
}
