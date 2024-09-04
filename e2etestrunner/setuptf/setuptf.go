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

package setuptf

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
)

const (
	tfPersistentDir                  = "tf/persistent"
	Push            SubscriptionMode = "push"
	Pull            SubscriptionMode = "pull"
)

type SubscriptionMode string

type tfOutput struct {
	PubsubInfoWrapper struct {
		Value PubsubInfo `json:"value"`
	} `json:"pubsub_info"`
}

type TopicInfo struct {
	TopicName        string `json:"topic_name"`
	SubscriptionName string `json:"subscription_name"`
}

type PubsubInfo struct {
	RequestTopic  TopicInfo `json:"request_topic"`
	ResponseTopic TopicInfo `json:"response_topic"`
}

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

func initCommand(ctx context.Context, projectID string) *exec.Cmd {
	return exec.CommandContext(
		ctx,
		"terraform",
		"init",
		"-input=false",
		fmt.Sprintf("-backend-config=bucket=%v-e2e-tfstate", projectID),
	)
}

// Runs the sequence of terraform commands most environemnts need, returns the
// output bytes of `terraform output -json` and a cleanup function to teardown
// the created resources.
//
// 1. Run terraform init
// 2. Create a new terraform workspace for the test run ID
// 3. Run terraform apply
// 4. Get output results from terraform output
//
// Cleanup method runs terraform destroy and then deletes the workspace.
func SetupTf(
	ctx context.Context,
	projectID string,
	testRunID string,
	tfDir string, // the Dir to set when running terraform commands in e.g. tf/gke
	tfVars map[string]string, // key-values for terraform input vars to send to terraform
	logger *log.Logger,
) (*PubsubInfo, func(), error) {
	tfVarArgs := tfVarMapToArgs(projectID, tfVars)
	cmd := initCommand(ctx, projectID)
	cmd.Args = append(cmd.Args, tfVarArgs...)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		return nil, func() {}, err
	}

	cleanup := func() {
		defer deleteWorkspace(ctx, testRunID, tfDir, logger)

		// Run terraform destroy
		cmd = exec.CommandContext(
			ctx,
			"terraform",
			"destroy",
			"-input=false",
			"-auto-approve",
		)
		cmd.Args = append(cmd.Args, tfVarArgs...)
		cmd.Dir = tfDir
		if err := runWithOutput(cmd, logger); err != nil {
			logger.Panic(err)
		}
	}

	// Create new terraform workspace
	cmd = exec.CommandContext(ctx, "terraform", "workspace", "new", testRunID)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		// try to switch to workspace if it already exists
		cmd = exec.CommandContext(ctx, "terraform", "workspace", "select", testRunID)
		cmd.Dir = tfDir

		if err := runWithOutput(cmd, logger); err != nil {
			return nil, cleanup, err
		}
	}

	// Run terraform apply
	cmd = exec.CommandContext(
		ctx,
		"terraform",
		"apply",
		"-input=false",
		"-auto-approve",
	)
	cmd.Args = append(cmd.Args, tfVarArgs...)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		return nil, cleanup, err
	}

	// Run terraform output
	cmd = exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = tfDir
	out, err := cmd.Output()
	if err != nil {
		logger.Println(err)
		return nil, cleanup, err
	}

	tfOutput := &tfOutput{}
	if err := json.Unmarshal(out, tfOutput); err != nil {
		return nil, cleanup, err
	}
	return &tfOutput.PubsubInfoWrapper.Value, cleanup, nil
}

// Create persistent resources (in tf/persistent) that are used across tests. No
// cleanup is required
func ApplyPersistent(
	ctx context.Context,
	projectID string,
	autoApprove bool,
	logger *log.Logger,
) error {
	logger.Println("Applying any changes to persistent resources")
	// Run terraform init
	cmd := initCommand(ctx, projectID)
	cmd.Dir = tfPersistentDir
	if err := runWithOutput(cmd, logger); err != nil {
		return err
	}

	// Select default terraform workspace
	cmd = exec.CommandContext(ctx, "terraform", "workspace", "select", "default")
	cmd.Dir = tfPersistentDir
	if err := runWithOutput(cmd, logger); err != nil {
		return err
	}

	// Run terraform apply
	cmd = exec.CommandContext(
		ctx,
		"terraform",
		"apply",
		"-input=false",
		// lock may not be acquired immediately in CI if there are multiple
		// jobs, but should only be a short wait
		"-lock-timeout=10m",
		fmt.Sprintf("-var=project_id=%v", projectID),
	)
	if autoApprove {
		cmd.Args = append(cmd.Args, "-auto-approve")
	} else {
		cmd.Stdin = os.Stdin
	}
	cmd.Dir = tfPersistentDir
	if err := runWithOutput(cmd, logger); err != nil {
		return err
	}

	return nil
}

func deleteWorkspace(
	ctx context.Context,
	testRunID string,
	tfDir string, // the Dir to set when running terraform commands in e.g. tf/gke
	logger *log.Logger,
) {
	// first, switch to default terraform workspace
	cmd := exec.CommandContext(ctx, "terraform", "workspace", "select", "default")
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		logger.Panic(err)
	}

	// issue delete
	cmd = exec.CommandContext(ctx, "terraform", "workspace", "delete", testRunID)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		logger.Panic(err)
	}
}

func tfVarMapToArgs(
	projectID string,
	tfVars map[string]string,
) []string {
	tfVarArgs := []string{fmt.Sprintf("-var=project_id=%v", projectID)}
	for k, v := range tfVars {
		tfVarArgs = append(tfVarArgs, fmt.Sprintf("-var=%v=%v", k, v))
	}
	return tfVarArgs
}
