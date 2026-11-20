// Copyright 2026 Google LLC
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

package e2etestrunner

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etestrunner/testclient"
)

const cloudRunInstanceTfDir string = "tf/cloud-run-instance"

// cloudRunInstanceRegions are the regions to try for a Cloud Run instance, in
// order, falling back if a region hits REGION_CAPACITY_EXHAUSTED.
var cloudRunInstanceRegions = []string{"us-central1", "us-west1", "us-east4", "us-east5"}

// SetupCloudRunInstance sets up the instrumented test server to run in a
// Cloud Run instance. Creates a new instance and pubsub resources, and runs
// the specified container image in pull mode. If a region has no capacity, it
// retries in the next region in cloudRunInstanceRegions. The returned cleanup
// function tears down everything.
func SetupCloudRunInstance(
	ctx context.Context,
	args *e2etesting.Args,
	logger *log.Logger,
) (*testclient.Client, e2etesting.Cleanup, error) {
	cleanup := func() {
		setuptf.CleanupTf(ctx, args.ProjectID, args.TestRunID, logger)
	}
	for _, location := range cloudRunInstanceRegions {
		pubsubInfo, err := setuptf.SetupTf(
			ctx,
			args.ProjectID,
			args.TestRunID,
			cloudRunInstanceTfDir,
			map[string]string{
				"image":    args.CloudRunInstance.Image,
				"location": location,
			},
			logger,
		)
		if err == nil {
			client, err := testclient.New(ctx, args.ProjectID, pubsubInfo)
			return client, cleanup, err
		}
		if !errors.Is(err, setuptf.ErrStockout) {
			return nil, cleanup, err
		}
		logger.Printf("No capacity for Cloud Run instance in %s\n", location)
	}
	return nil, cleanup, fmt.Errorf("no capacity for Cloud Run instance in any of %v: %w", cloudRunInstanceRegions, setuptf.ErrStockout)
}
