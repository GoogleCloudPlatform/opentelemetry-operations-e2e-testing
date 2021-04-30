package e2e_testing

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
	"github.com/alexflint/go-arg"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

var testServerClient *testclient.Client

type Cleanup func()

type LocalCmd struct {
	Image string `arg:"required" help:"docker container image to deploy and test"`
	Port  string `default:"8000"`

	// Needed when running without a metadata server for credentials
	GoogleApplicationCredentials string `arg:"--google-application-credentials,env:GOOGLE_APPLICATION_CREDENTIALS" help:"Path to google credentials key file to mount into test server container"`

	// May be needed when running this binary in a container
	Network string `help:"Docker network to use when starting the container, optional"`
}

var args struct {
	Local *LocalCmd `arg:"subcommand:local"`

	GoTestFlags string `help:"go test flags to pass through, e.g. --gotestflags='-test.v'"`
	ProjectID   string `arg:"required,--project-id,env:PROJECT_ID" help:"GCP project id/name"`
}

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

	var (
		client  *testclient.Client
		cleanup Cleanup
		err     error
	)
	switch {
	case args.Local != nil:
		client, cleanup, err = setupLocal(args.Local, logger)
	}

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

/**
 * Set up the instrumented test server for a local run by running in a docker
 * container on the local host
 */
func setupLocal(local *LocalCmd, logger *log.Logger) (*testclient.Client, Cleanup, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, noopCleanup, err
	}
	cli.NegotiateAPIVersion(ctx)

	createdRes, err := createContainer(ctx, cli, local, logger)
	if err != nil {
		return nil, noopCleanup, err
	}
	if len(createdRes.Warnings) != 0 {
		logger.Printf("Started with warnings: %v", createdRes.Warnings)
	}
	containerID := createdRes.ID
	removeContainer := func() {
		err = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			panic(err)
		}
	}

	err = cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return nil, removeContainer, err
	}

	cleanup := func() {
		logger.Printf("Stopping and removing container ID %v\n", containerID)
		timeout := (time.Second * 15)
		err = cli.ContainerStop(ctx, containerID, &timeout)
		defer removeContainer()
		if err != nil {
			panic(err)
		}
	}

	err = startForwardingContainerLogs(ctx, cli, containerID, logger)
	if err != nil {
		return nil, cleanup, err
	}

	address, err := getContainerAddress(ctx, cli, containerID, logger)
	if err != nil {
		return nil, cleanup, err
	}

	return testclient.New(fmt.Sprintf("%v:%v", address, local.Port)), cleanup, err
}

func createContainer(
	ctx context.Context,
	cli *client.Client,
	local *LocalCmd,
	logger *log.Logger,
) (container.ContainerCreateCreatedBody, error) {
	env := []string{"PORT=" + local.Port, "PROJECT_ID=" + args.ProjectID}
	mounts := []mount.Mount{}
	if local.GoogleApplicationCredentials != "" {
		env = append(env, "GOOGLE_APPLICATION_CREDENTIALS="+local.GoogleApplicationCredentials)
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   local.GoogleApplicationCredentials,
			Target:   local.GoogleApplicationCredentials,
			ReadOnly: true,
		})

	}
	return cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: local.Image,
			Env:   env,
			ExposedPorts: nat.PortSet{
				nat.Port(local.Port): struct{}{},
			},
		},
		&container.HostConfig{
			Mounts:      mounts,
			NetworkMode: container.NetworkMode(local.Network),
		},
		nil,
		nil,
		"",
	)
}

// forward container logs to stdout/stderr
func startForwardingContainerLogs(
	ctx context.Context,
	cli *client.Client,
	containerID string,
	logger *log.Logger,
) error {
	reader, err := cli.ContainerLogs(
		ctx,
		containerID,
		types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true},
	)
	if err != nil {
		return err
	}
	go func() {
		defer reader.Close()
		if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, reader); err != nil {
			logger.Printf("Error while reading logs, %v\n", err)
		}
	}()
	return nil
}

func getContainerAddress(
	ctx context.Context,
	cli *client.Client,
	containerID string,
	logger *log.Logger,
) (string, error) {
	inspectRes, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	networks := inspectRes.NetworkSettings.Networks
	if len(networks) != 1 {
		return "", fmt.Errorf("Expected only one network, instead got: %v", networks)
	}
	var address string
	for _, v := range networks {
		address = v.IPAddress
	}
	return address, nil
}

func noopCleanup() {}
