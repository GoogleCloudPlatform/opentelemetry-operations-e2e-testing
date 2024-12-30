package e2etestrunner_collector

import (
	"context"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/util"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/util/setuptf"
	"log"
)

const gkeOperatorCollectorTfDir string = "tf/gke-operator-collector"

// SetupGkeOperatorCollector Set up the collector to run in GKE.
// Creates a new pod and runs the specified container image in a pod.
// The returned cleanup function tears down the whole cluster.
func SetupGkeOperatorCollector(
	ctx context.Context,
	args *util.Args,
	logger *log.Logger,
) (util.Cleanup, error) {
	_, cleanupTf, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		gkeOperatorCollectorTfDir,
		map[string]string{
			"image": args.GkeOperatorCollector.Image,
		},
		logger,
	)
	if err != nil {
		return cleanupTf, err
	}

	return cleanupTf, err
}
