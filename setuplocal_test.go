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
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// Set up the instrumented test server for a local run by running in a docker
// container on the local host
func setupLocal(
	ctx context.Context,
	args *Args,
	logger *log.Logger,
) (*testclient.Client, Cleanup, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, noopCleanup, err
	}
	cli.NegotiateAPIVersion(ctx)

	createdRes, err := createContainer(ctx, cli, args.Local, logger)
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

	return testclient.New(fmt.Sprintf("%v:%v", address, args.Local.Port)), cleanup, err
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
