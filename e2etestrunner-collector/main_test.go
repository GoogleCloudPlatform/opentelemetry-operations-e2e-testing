//go:build e2ecollector

package e2etestrunner_collector

import (
	"context"
	"flag"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/util"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/util/setuptf"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"
)

var (
	args         util.Args
	resourceType string
)

func TestMain(m *testing.M) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	p := arg.MustParse(&args)
	if p.Subcommand() == nil {
		p.Fail("missing command")
	}
	// Need a logger just for TestMain() before testing.T is available
	logger := log.New(os.Stdout, "TestMain: ", log.LstdFlags|log.Lshortfile)
	ctx := context.Background()

	// Handle special case of just creating persistent resources
	if args.ApplyPersistent != nil {
		err := setuptf.ApplyPersistentCollector(ctx, args.ProjectID, args.ApplyPersistent.AutoApprove, logger)
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
		hex, err := util.RandomHex(6)
		if err != nil {
			logger.Fatalf("error generating random hex string: %v\n", err)
		}
		args.TestRunID = hex
	}

	var setupFunc util.SetupCollectorFunc
	switch {
	case args.GceCollector != nil:
		setupFunc = SetupGceCollector
		resourceType = "gce_instance"
	case args.GkeCollector != nil:
		setupFunc = SetupGkeCollector
		resourceType = "k8s_container"
	case args.GkeOperatorCollector != nil:
		setupFunc = SetupGkeOperatorCollector
		resourceType = "k8s_container"
	case args.CloudRunCollector != nil:
		setupFunc = SetupCloudRunCollector
		resourceType = "generic_task"
	}
	cleanup, err := setupFunc(ctx, &args, logger)

	defer cleanup()
	if err != nil {
		logger.Panic(err)
	}

	// wait for instrumented test server to be healthy
	logger.Printf("Waiting for health check on (will timeout after %v)\n", args.HealthCheckTimeout)
	time.Sleep(args.HealthCheckTimeout)
	// Run tests
	logger.Print(util.BeginOutputArt)
	m.Run()
	logger.Print(util.EndOutputArt)
}
