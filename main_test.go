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

package e2e_testing

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/setuptf"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
	"github.com/alexflint/go-arg"
)

var (
	args             Args
	testServerClient *testclient.Client
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	p := arg.MustParse(&args)
	if p.Subcommand() == nil {
		p.Fail("missing command")
	}
	// Need a logger just for TestMain() before testing.T is available
	logger := log.New(os.Stdout, "TestMain: ", log.LstdFlags|log.Lshortfile)
	ctx := context.Background()

	// Handle special case of just creating persistent resources
	if args.ApplyPersistent != nil {
		err := setuptf.ApplyPersistent(ctx, args.ProjectID, args.ApplyPersistent.AutoApprove, logger)
		if err != nil {
			logger.Panic(err)
		}
		return
	}

	// hacky but works
	os.Args = append([]string{os.Args[0]}, strings.Fields(args.GoTestFlags)...)
	flag.Parse()

	// handle any complex defaults
	if args.TestRunID == "" {
		hex, err := randomHex(6)
		if err != nil {
			logger.Fatalf("error generating random hex string: %v\n", err)
		}
		args.TestRunID = hex
	}

	var setupFunc SetupFunc
	switch {
	case args.Local != nil:
		setupFunc = SetupLocal
	case args.Gce != nil:
		setupFunc = SetupGce
	case args.Gke != nil:
		setupFunc = SetupGke
	case args.CloudRun != nil:
		setupFunc = SetupCloudRun
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
	logger.Print(BeginOutputArt)
	m.Run()
	logger.Print(EndOutputArt)
}
