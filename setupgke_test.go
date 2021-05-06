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
	"os/exec"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
)

const gkeTfDir string = "tf/gce"

func runWithOutput(cmd *exec.Cmd, logger *log.Logger) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logger.Printf("Running command: %v\n", cmd)
	if err := cmd.Run(); err != nil {
		logger.Println(err)
		return err
	}
	return nil
}

// Set up the instrumented test server to run in GKE. Creates a new GKE cluster
// and runs the specified container image in a pod. The returned cleanup
// function tears down the whole cluster.
func setupGke(
	ctx context.Context,
	args *Args,
	logger *log.Logger,
) (*testclient.Client, Cleanup, error) {
	// Run terraform init just in case
	cmd := exec.CommandContext(
		ctx,
		"terraform",
		"init",
		fmt.Sprintf("-backend-config=bucket=%v-e2e-tfstate", args.ProjectID),
		fmt.Sprintf("-var=project_id=%v", args.ProjectID),
		"-input=false",
	)
	cmd.Dir = gkeTfDir
	if err := runWithOutput(cmd, logger); err != nil {
		return nil, noopCleanup, err
	}

	delete_workspace := func() {
		// first, switch to default terraform workspace
		cmd = exec.CommandContext(ctx, "terraform", "workspace", "select", "default")
		cmd.Dir = gkeTfDir
		if err := runWithOutput(cmd, logger); err != nil {
			panic(err)
		}

		// issue delete
		cmd = exec.CommandContext(ctx, "terraform", "workspace", "delete", args.TestRunID)
		cmd.Dir = gkeTfDir
		if err := runWithOutput(cmd, logger); err != nil {
			panic(err)
		}
	}

	// Create new terraform workspace
	cmd = exec.CommandContext(ctx, "terraform", "workspace", "new", args.TestRunID)
	cmd.Dir = gkeTfDir
	if err := runWithOutput(cmd, logger); err != nil {
		return nil, delete_workspace, err
	}

	// Run terraform apply
	cmd = exec.CommandContext(
		ctx,
		"terraform",
		"apply",
		fmt.Sprintf("-var=project_id=%v", args.ProjectID),
		"-input=false",
		"-auto-approve",
	)
	cmd.Dir = gkeTfDir
	if err := runWithOutput(cmd, logger); err != nil {
		return nil, delete_workspace, err
	}

	cleanup := func() {
		defer delete_workspace()

		// Run terraform destroy
		cmd = exec.CommandContext(
			ctx,
			"terraform",
			"destroy",
			fmt.Sprintf("-var=project_id=%v", args.ProjectID),
			"-input=false",
			"-auto-approve",
		)
		cmd.Dir = gkeTfDir
		if err := runWithOutput(cmd, logger); err != nil {
			panic(err)
		}
	}

	return testclient.New("foobar"), cleanup, nil
}
