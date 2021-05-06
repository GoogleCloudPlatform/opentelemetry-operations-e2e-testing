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
	"flag"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
	"github.com/alexflint/go-arg"
)

type LocalCmd struct {
	Image string `arg:"required" help:"docker container image to deploy and test"`
	Port  string `default:"8000"`

	// Needed when running without a metadata server for credentials
	GoogleApplicationCredentials string `arg:"--google-application-credentials,env:GOOGLE_APPLICATION_CREDENTIALS" help:"Path to google credentials key file to mount into test server container"`

	// May be needed when running this binary in a container
	Network string `help:"Docker network to use when starting the container, optional"`
}

type GkeCmd struct {
	Image string `arg:"required" help:"docker container image to deploy and test"`
}

type Args struct {
	Local *LocalCmd `arg:"subcommand:local"`
	Gke   *GkeCmd   `arg:"subcommand:gke"`

	GoTestFlags string `help:"go test flags to pass through, e.g. --gotestflags='-test.v'"`
	ProjectID   string `arg:"required,--project-id,env:PROJECT_ID" help:"GCP project id/name"`

	// This is used in a new terraform workspace's name and in the GCP resources
	// we create. Pass the GCB build ID in CI to get the build id formatted into
	// resources created for debugging. If not provided, we generate a hex
	// string.
	TestRunID string `arg:"--test-run-id,env:TEST_RUN_ID" help:"Optional test run id to use to partition terraform resources"`
}

type Cleanup func()
type SetupFunc func(
	context.Context,
	*Args,
	*log.Logger,
) (*testclient.Client, Cleanup, error)

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

	// hacky but works
	os.Args = append([]string{os.Args[0]}, strings.Fields(args.GoTestFlags)...)
	flag.Parse()

	// need a logger just for TestMain() before testing.T is available
	logger := log.New(os.Stdout, "TestMain: ", log.LstdFlags|log.Lshortfile)

	// handle any complex defaults
	if args.TestRunID == "" {
		hex, err := randomHex(6)
		if err != nil {
			logger.Fatalf("error generating random hex string: %v\n", err)
		}
		args.TestRunID = hex
	}

	ctx := context.Background()

	var setupFunc SetupFunc
	switch {
	case args.Local != nil:
		setupFunc = setupLocal
	case args.Gke != nil:
		setupFunc = setupGke
	}
	client, cleanup, err := setupFunc(ctx, &args, logger)

	defer cleanup()
	if err != nil {
		panic(err)
	}

	// set global client
	testServerClient = client

	// wait for instrumented test server to be healthy
	err = testServerClient.WaitForHealth(context.Background(), logger)
	if err != nil {
		panic(err)
	}

	// Run tests
	m.Run()
}

func noopCleanup() {}
