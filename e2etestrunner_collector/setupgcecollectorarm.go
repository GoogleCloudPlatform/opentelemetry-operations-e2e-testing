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
	"errors"
	"fmt"
	"log"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
)

const gceCollectorArmTfDir string = "tf/gce-collector-arm"

// gceCollectorArmZones are the zones to try for the C4A VM, in order. A
// stockout can cover a whole region, so each zone is in a different region.
var gceCollectorArmZones = []string{"us-east1-b", "us-west1-a", "us-east4-a", "us-central1-a"}

// SetupGceCollectorArm Set up the collector to run in GCE arm container. Creates a new
// GCE VM resources, and runs the specified container image. If a zone has no
// capacity for the VM, it tries the next one in gceCollectorArmZones. The
// returned cleanup function tears down the VM.
func SetupGceCollectorArm(
	ctx context.Context,
	args *e2etesting.Args,
	logger *log.Logger,
) (e2etesting.Cleanup, error) {
	cleanup := func() {
		setuptf.CleanupTf(ctx, args.ProjectID, args.TestRunID, logger)
	}
	for _, zone := range gceCollectorArmZones {
		_, err := setuptf.SetupTf(
			ctx,
			args.ProjectID,
			args.TestRunID,
			gceCollectorArmTfDir,
			map[string]string{
				"image": args.GceCollectorArm.Image,
				"zone":  zone,
			},
			logger,
		)
		if !errors.Is(err, setuptf.ErrStockout) {
			// Either it worked, or it failed in a way another zone won't fix.
			return cleanup, err
		}
		logger.Printf("No capacity for the VM in %s\n", zone)
	}
	return cleanup, fmt.Errorf("no capacity for the VM in any of %v: %w", gceCollectorArmZones, setuptf.ErrStockout)
}
