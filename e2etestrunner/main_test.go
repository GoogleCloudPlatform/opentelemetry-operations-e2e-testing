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

// Only build as part of e2e tests, not regular go test invocations
//go:build e2e

package e2etestrunner

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etestrunner/testclient"
)

var (
	args             e2etesting.Args
	testServerClient *testclient.Client
)

func TestMain(m *testing.M) {
	logger, ctx := e2etesting.InitTestMain(&args, setuptf.ApplyPersistent)

	var setupFunc e2etesting.SetupFunc
	switch {
	case args.Local != nil:
		setupFunc = SetupLocal
	case args.Gce != nil:
		setupFunc = SetupGce
	case args.Gke != nil:
		setupFunc = SetupGke
	case args.CloudRun != nil:
		setupFunc = SetupCloudRun
	case args.CloudFunctionsGen2 != nil:
		setupFunc = SetupCloudFunctionsGen2
	case args.Gae != nil:
		setupFunc = SetupGae
	case args.GaeStandard != nil:
		setupFunc = SetupGaeStandard
	}
	client, cleanup, err := setupFunc(ctx, &args, logger)

	defer cleanup()
	if err != nil {
		logger.Panic(err)
	}

	// set global client
	testServerClient = client

	// wait for instrumented test server to be healthy
	logger.Printf("Waiting for health check on pub/sub channel (will timeout after %v)\n", args.HealthCheckTimeout)
	cctx, cancel := context.WithTimeout(ctx, args.HealthCheckTimeout)
	defer cancel()
	err = testServerClient.WaitForHealth(cctx, logger)
	if err != nil {
		logger.Panic(err)
	}

	// Run tests
	logger.Print(e2etesting.BeginOutputArt)
	m.Run()
	logger.Print(e2etesting.EndOutputArt)
}
