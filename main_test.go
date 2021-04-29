package e2e_testing

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

type Cleanup func()

type LocalCmd struct {
	Image string `arg:"required" help:"docker container image to deploy and test"`
	Port  string `default:"8000"`

	// Needed when running without a metadata server for credentials
	BindMountGcloud string `arg:"--bind-mount-gcloud" help:"Optional path to gcloud directory to bind mount into the container"`

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

	var (
		client  *Client
		cleanup Cleanup
		err     error
	)
	switch {
	case args.Local != nil:
		client, cleanup, err = setupLocal(args.Local)
	}

	defer cleanup()
	if err != nil {
		panic(err)
	}

	// set global client
	testServerClient = client

	time.Sleep(time.Second * 2)

	// Run tests
	m.Run()
}

/**
 * Set up the instrumented test server for a local run by running in a docker
 * container on the local host
 */
func setupLocal(local *LocalCmd) (*Client, Cleanup, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, noopCleanup, err
	}
	cli.NegotiateAPIVersion(ctx)

	createdRes, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: local.Image,
			Env:   []string{"PORT=" + local.Port, "PROJECT_ID=" + args.ProjectID},
			ExposedPorts: nat.PortSet{
				nat.Port(local.Port): struct{}{},
			},
		},
		&container.HostConfig{
			Mounts: func() []mount.Mount {
				if local.BindMountGcloud != "" {
					return []mount.Mount{
						{
							Type:     mount.TypeBind,
							Source:   local.BindMountGcloud,
							Target:   "/root/.config/gcloud",
							ReadOnly: true,
						},
					}
				} else {
					return nil
				}
			}(),
			NetworkMode: container.NetworkMode(local.Network),
		},
		nil,
		nil,
		"",
	)
	if err != nil {
		return nil, noopCleanup, err
	}
	if len(createdRes.Warnings) != 0 {
		fmt.Printf("Started with warnings: %v", createdRes.Warnings)
	}
	containerID := createdRes.ID
	removeContainer := func() {
		err = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
		if err != nil {
			panic(err)
		}
	}

	err = cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return nil, removeContainer, err
	}

	cleanup := func() {
		fmt.Printf("Stopping and removing container ID %v\n", containerID)
		timeout := (time.Second * 15)
		err = cli.ContainerStop(ctx, containerID, &timeout)
		if err != nil {
			panic(err)
		}
		removeContainer()
	}

	// forward container logs to stdout/stderr
	reader, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	if err != nil {
		return nil, cleanup, err
	}
	go func() {
		defer reader.Close()
		if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, reader); err != nil {
			fmt.Fprintf(os.Stderr, "Error while reading logs, %v\n", err)
		}
	}()

	// Get IP address of the test server
	inspectRes, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, cleanup, err
	}
	networks := inspectRes.NetworkSettings.Networks
	if len(networks) != 1 {
		return nil, cleanup, fmt.Errorf("Expected only one network, instead got: %v", networks)
	}
	var address string
	for _, v := range networks {
		address = v.IPAddress
	}

	return &Client{Address: fmt.Sprintf("%v:%v", address, local.Port)}, cleanup, err
}

func noopCleanup() {}
