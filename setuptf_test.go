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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
)

type tfVar struct {
	Sensitive bool   `json:"sensitive"`
	Type      string `json:"type"`
	Value     string `json:"value"`
}

type tfOutput struct {
	Address tfVar `json:"address"`
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

// Runs the sequence of terraform commands most environemnts need, returns the
// output string and a cleanup function to teardown the created resources.
//
// 1. Run terraform init
// 2. Create a new terraform workspace for the test run ID
// 3. Run terraform apply
// 4. Get output results from terraform output
//
// Cleanup method runs terraform destroy and then deletes the workspace.
func setupTf(
	ctx context.Context,
	tfDir string, // the Dir to set when running terraform commands in e.g. tf/gke
	args *Args,
	logger *log.Logger,
) (string, Cleanup, error) {
	// Run terraform init just in case
	cmd := exec.CommandContext(
		ctx,
		"terraform",
		"init",
		fmt.Sprintf("-backend-config=bucket=%v-e2e-tfstate", args.ProjectID),
		fmt.Sprintf("-var=project_id=%v", args.ProjectID),
		"-input=false",
	)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		return "", noopCleanup, err
	}

	cleanup := func() {
		defer deleteWorkspace(ctx, tfDir, args, logger)

		// Run terraform destroy
		cmd = exec.CommandContext(
			ctx,
			"terraform",
			"destroy",
			fmt.Sprintf("-var=project_id=%v", args.ProjectID),
			"-input=false",
			"-auto-approve",
		)
		cmd.Dir = tfDir
		if err := runWithOutput(cmd, logger); err != nil {
			panic(err)
		}
	}

	// Create new terraform workspace
	cmd = exec.CommandContext(ctx, "terraform", "workspace", "new", args.TestRunID)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		// try to switch to workspace if it already exists
		cmd = exec.CommandContext(ctx, "terraform", "workspace", "select", args.TestRunID)
		cmd.Dir = tfDir

		if err := runWithOutput(cmd, logger); err != nil {
			return "", cleanup, err
		}
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
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		return "", cleanup, err
	}

	// Run terraform output
	cmd = exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = tfDir
	out, err := cmd.Output()
	if err != nil {
		logger.Println(err)
		return "", cleanup, err
	}
	var output tfOutput
	if err := json.Unmarshal(out, &output); err != nil {
		logger.Printf("Error unmarshaling terraform output json: %v\n", err)
		return "", cleanup, err
	}
	logger.Printf("Got address: %v\n", output.Address.Value)
	return output.Address.Value, cleanup, nil
}

func deleteWorkspace(
	ctx context.Context,
	tfDir string, // the Dir to set when running terraform commands in e.g. tf/gke
	args *Args,
	logger *log.Logger,
) {
	// first, switch to default terraform workspace
	cmd := exec.CommandContext(ctx, "terraform", "workspace", "select", "default")
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		panic(err)
	}

	// issue delete
	cmd = exec.CommandContext(ctx, "terraform", "workspace", "delete", args.TestRunID)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		panic(err)
	}
}
