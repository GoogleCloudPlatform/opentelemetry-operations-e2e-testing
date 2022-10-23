package e2e_testing

import (
	"context"
	"log"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/setuptf"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
)

const cloudFunctionTfDir string = "tf/cloud-function"

func SetupCloudFunction(
	ctx context.Context,
	args *Args,
	logger *log.Logger,
) (*testclient.Client, Cleanup, error) {
	pubsubInfo, cleanupTf, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		cloudFunctionTfDir,
		map[string]string{
			"image":      args.CloudFunction.Image,
			"runtime":    args.CloudFunction.Runtime,
			"source":     args.CloudFunction.SourceZip,
			"entrypoint": args.CloudFunction.EntryPoint,
		},
		logger,
	)
	if err != nil {
		return nil, cleanupTf, err
	}

	client, err := testclient.New(ctx, args.ProjectID, pubsubInfo)
	return client, cleanupTf, err
}
