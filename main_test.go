package e2e_testing

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Cleanup func()

type LocalCmd struct {
	Image string `arg:"required" help:"docker container image to deploy and test"`
}

var args struct {
	Local *LocalCmd `arg:"subcommand:local"`

	GoTestFlags string `help:"go test flags to pass through, e.g. --gotestflags='-test.v'"`
	ProjectID   string `arg:"required,--project-id,env:PROJECT_ID" help:"GCP project id/name"`
}

func TestMain(m *testing.M) {
	p := arg.MustParse(&args)
	if p.Subcommand() == nil {
		p.Fail("missing command")
	}

	// hacky but works
	os.Args = append([]string{os.Args[0]}, strings.Fields(args.GoTestFlags)...)
	flag.Parse()

	var err error
	var cleanup Cleanup
	switch {
	case args.Local != nil:
		cleanup, err = setupLocal(args.Local)
	}

	defer cleanup()
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 5)

	// Run tests
	m.Run()
}

/**
 * Set up the instrumented test server for a local run by running in a docker
 * container on the local host
 */
func setupLocal(local *LocalCmd) (Cleanup, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return noopCleanup, err
	}

	ctx := context.Background()
	createdRes, err := cli.ContainerCreate(ctx, &container.Config{
		Image: local.Image,
		// AttachStdout: true, AttachStderr: true,
	}, nil, nil, nil, "")
	if err != nil {
		return noopCleanup, err
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
		return removeContainer, err
	}

	cleanup := func() {
		fmt.Printf("Cleanup called, killing container ID %v\n", containerID)
		defer removeContainer()
		timeout := (time.Second * 15)
		err = cli.ContainerStop(ctx, containerID, &timeout)
		if err != nil {
			panic(err)
		}
	}

	// read logs
	reader, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	if err != nil {
		return cleanup, err
	}
	go func() {
		defer reader.Close()
		written, err := stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
		fmt.Printf("Wrote %v, err = %v\n", written, err)
	}()

	return cleanup, err
}

func noopCleanup() {}
