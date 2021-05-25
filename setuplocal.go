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
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/setuptf"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

const localTfDir = "tf/local"

// Set up the instrumented test server for a local run by running in a docker
// container on the local host
func SetupLocal(
	ctx context.Context,
	args *Args,
	logger *log.Logger,
) (*testclient.Client, Cleanup, error) {
	pubsubInfo, cleanupTf, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		localTfDir,
		map[string]string{},
		logger,
	)
	if err != nil {
		return nil, cleanupTf, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, cleanupTf, err
	}
	cli.NegotiateAPIVersion(ctx)

	createdRes, err := createContainer(ctx, cli, args, pubsubInfo, logger)
	if err != nil {
		if errdefs.IsNotFound(err) {
			err = fmt.Errorf(
				`docker image not found, try running "docker pull %v": %w`,
				args.Local.Image,
				err,
			)
		}
		return nil, cleanupTf, err
	}

	if len(createdRes.Warnings) != 0 {
		logger.Printf("Started with warnings: %v", createdRes.Warnings)
	}
	containerID := createdRes.ID
	removeContainer := func() {
		defer cleanupTf()
		err = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			logger.Panic(err)
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
			logger.Panic(err)
		}
	}

	err = startForwardingContainerLogs(ctx, cli, containerID, logger)
	if err != nil {
		return nil, cleanup, err
	}

	client, err := testclient.New(ctx, args.ProjectID, pubsubInfo)
	if err != nil {
		return nil, cleanup, err
	}
	return client, cleanup, err
}

func createContainer(
	ctx context.Context,
	cli *client.Client,
	args *Args,
	pubsubInfo *setuptf.PubsubInfo,
	logger *log.Logger,
) (container.ContainerCreateCreatedBody, error) {
	env := []string{
		"PORT=" + args.Local.Port,
		"PROJECT_ID=" + args.ProjectID,
		"REQUEST_SUBSCRIPTION_NAME=" + pubsubInfo.RequestTopic.SubscriptionName,
		"RESPONSE_TOPIC_NAME=" + pubsubInfo.ResponseTopic.TopicName,
		"SUBSCRIPTION_MODE=" + string(setuptf.Pull),
	}
	mounts := []mount.Mount{}
	if args.Local.GoogleApplicationCredentials != "" {
		env = append(env, "GOOGLE_APPLICATION_CREDENTIALS="+args.Local.GoogleApplicationCredentials)
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   args.Local.GoogleApplicationCredentials,
			Target:   args.Local.GoogleApplicationCredentials,
			ReadOnly: true,
		})

	}
	return cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: args.Local.Image,
			Env:   env,
			ExposedPorts: nat.PortSet{
				nat.Port(args.Local.Port): struct{}{},
			},
			User: args.Local.ContainerUser,
		},
		&container.HostConfig{
			Mounts:      mounts,
			NetworkMode: container.NetworkMode(args.Local.Network),
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
