// Copyright 2021 Google LLC
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

package e2etestrunner_collector

import (
	"context"
	"log"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
)

const cloudRunCollectorTfDir string = "tf/cloud-run-collector"

// SetupCloudRunCollector sets up the collector to run in Cloud Run.
// Creates a new service and runs the specified container image as a revision.
// The returned cleanup function tears down everything.
func SetupCloudRunCollector(
	ctx context.Context,
	args *e2etesting.Args,
	logger *log.Logger,
) (e2etesting.Cleanup, error) {
	_, cleanupTf, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		cloudRunCollectorTfDir,
		map[string]string{
			"image": args.CloudRunCollector.Image,
		},
		logger,
	)

	return cleanupTf, err
}
