// Copyright 2021 Google
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

package e2e_testing

import (
	"context"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/setuptf"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
)

const gkeTfDir string = "tf/gce"

// Set up the instrumented test server to run in GKE. Creates a new GKE cluster
// and runs the specified container image in a pod. The returned cleanup
// function tears down the whole cluster.
func SetupGke(
	ctx context.Context,
	args *Args,
	logger *log.Logger,
) (*testclient.Client, Cleanup, error) {
	pubsubInfo, cleanup, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		gkeTfDir,
		map[string]string{},
		logger,
	)

	// TODO remove
	os.Exit(1)

	client, err := testclient.New(ctx, args.ProjectID, pubsubInfo)
	if err != nil {
		return nil, cleanup, err
	}
	return client, cleanup, err
}
