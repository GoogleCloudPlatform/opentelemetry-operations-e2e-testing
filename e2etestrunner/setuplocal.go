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

package e2etestrunner

import (
	"context"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etesting/setuptf"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/e2etestrunner/testclient"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

const localTfDir = "tf/local"

// Set up the instrumented test server for a local run by running in a docker
// container on the local host
func SetupLocal(
	ctx context.Context,
	args *e2etesting.Args,
	logger *log.Logger,
) (*testclient.Client, e2etesting.Cleanup, error) {
	// 1. Define basic cleanup that only does Terraform
	cleanupTf := func() {
		setuptf.CleanupTf(ctx, args.ProjectID, args.TestRunID, "tf/destroy", logger)
	}

	pubsubInfo, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		localTfDir,
		map[string]string{},
		logger,
	)
	if err != nil {
		// If SetupTf fails, we still want to try destroying workspace
		return nil, cleanupTf, err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, cleanupTf, err
	}
	cli.NegotiateAPIVersion(ctx)

	createdRes, err := createContainer(ctx, cli, args, pubsubInfo, logger)
	if err != nil {
		// Container creation failed, so no container to remove, but must cleanup TF
		return nil, cleanupTf, err
	}

	containerID := createdRes.ID

	// 2. Now we have a container, update cleanup to do both!
	cleanupAll := func() {
		logger.Printf("Stopping and removing container ID %v\n", containerID)
		timeout := 15
		err := cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			logger.Printf("Error stopping container: %v", err)
		}

		// Defer ensures they run even if something panics
		defer cleanupTf()

		err = cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
		if err != nil {
			logger.Printf("Error removing container: %v", err)
		}
	}

	err = cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return nil, cleanupAll, err
	}

	err = startForwardingContainerLogs(ctx, cli, containerID, logger)
	if err != nil {
		return nil, cleanupAll, err
	}

	client, err := testclient.New(ctx, args.ProjectID, pubsubInfo)
	if err != nil {
		return nil, cleanupAll, err
	}
	return client, cleanupAll, nil
}

func createContainer(
	ctx context.Context,
	cli *client.Client,
	args *e2etesting.Args,
	pubsubInfo *setuptf.PubsubInfo,
	logger *log.Logger,
) (container.CreateResponse, error) {
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
		container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true},
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
