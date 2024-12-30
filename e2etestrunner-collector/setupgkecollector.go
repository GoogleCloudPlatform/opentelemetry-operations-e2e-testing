package e2etestrunner_collector

import (
	"context"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/util"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/util/setuptf"
	"log"
)

const gkeCollectorTfDir string = "tf/gke-collector"

// SetupGkeCollector Set up the collector to run in GKE.
// Creates a new pod and runs the specified container image in a pod.
// The returned cleanup function tears down the whole cluster.
func SetupGkeCollector(
	ctx context.Context,
	args *util.Args,
	logger *log.Logger,
) (util.Cleanup, error) {
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
	if err != nil {
		return cleanupTf, err
	}

	return cleanupTf, err
}
