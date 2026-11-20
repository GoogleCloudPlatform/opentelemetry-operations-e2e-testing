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
	"log"
	"strconv"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etestrunner/testclient"
)

const gceMigTfDir string = "tf/gce-mig"

// SetupGceMig sets up the instrumented test server to run in a GCE Managed
// Instance Group (zonal by default, or regional if --regional is set). Creates
// a new instance template, MIG, and pubsub resources, and runs the specified
// container image. The returned cleanup function tears down all resources.
func SetupGceMig(
	ctx context.Context,
	args *e2etesting.Args,
	logger *log.Logger,
) (*testclient.Client, e2etesting.Cleanup, error) {
	cleanup := func() {
		setuptf.CleanupTf(ctx, args.ProjectID, args.TestRunID, logger)
	}
	pubsubInfo, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		gceMigTfDir,
		map[string]string{
			"image":    args.GceMig.Image,
			"regional": strconv.FormatBool(args.GceMig.Regional),
		},
		logger,
	)
	if err != nil {
		return nil, cleanup, err
	}

	client, err := testclient.New(ctx, args.ProjectID, pubsubInfo)
	return client, cleanup, err
}
