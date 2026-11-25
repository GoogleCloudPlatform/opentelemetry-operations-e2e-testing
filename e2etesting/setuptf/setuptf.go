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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

const (
	tfPersistentDir          = "tf/persistent"
	tfPersistentCollectorDir = "tf/persistent-collector"
	// Must stay in sync with backend "gcs" prefix in tf/common/main-common.tf.
	tfStatePrefix                            = "terraform/state"
	gcsCheckTimeout                          = 30 * time.Second
	Push                    SubscriptionMode = "push"
	Pull                    SubscriptionMode = "pull"
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

// ErrStockout means terraform apply failed because GCE had no capacity left
// for the requested resources in the requested zone. Another zone may work.
var ErrStockout = errors.New("GCE has no capacity for this request in this zone")

// stockoutRe matches the errors GCE returns when a zone is out of capacity.
// GCE words these errors in several ways, see
// https://cloud.google.com/compute/docs/troubleshooting/troubleshooting-resource-availability
var stockoutRe = regexp.MustCompile(`ZONE_RESOURCE_POOL_EXHAUSTED|STOCKOUT|does not have enough resources available|VM instance is currently unavailable`)

// runApply runs a terraform apply command like runWithOutput, but also keeps a
// copy of stderr, where terraform prints errors. If the apply failed because
// of a GCE stockout, the returned error wraps ErrStockout.
func runApply(cmd *exec.Cmd, logger *log.Logger) error {
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	logger.Printf("Running command: %v\n", cmd)
	if err := cmd.Run(); err != nil {
		logger.Println(err)
		if stockoutRe.Match(stderr.Bytes()) {
			return fmt.Errorf("%w: %w", ErrStockout, err)
		}
		return err
	}
	return nil
}

func tfStateBucket(projectID string) string {
	return fmt.Sprintf("%v-e2e-tfstate", projectID)
}

func initCommand(ctx context.Context, projectID string) *exec.Cmd {
	return exec.CommandContext(
		ctx,
		"terraform",
		"init",
		"-input=false",
		fmt.Sprintf("-backend-config=bucket=%v", tfStateBucket(projectID)),
	)
}

// Runs the sequence of terraform commands most environments need, returns the
// parsed PubsubInfo from `terraform output -json`.
//
// 1. Run terraform init
// 2. Create a new terraform workspace for the test run ID
// 3. Run terraform apply
// 4. Get output results from terraform output
//
// If terraform apply fails because of a GCE stockout, the returned error wraps
// ErrStockout, so callers can retry in another zone.
//
// Cleanup method runs terraform destroy and then deletes the workspace.
func SetupTf(
	ctx context.Context,
	projectID string,
	testRunID string,
	tfDir string, // the Dir to set when running terraform commands in e.g. tf/gke
	tfVars map[string]string, // key-values for terraform input vars to send to terraform
	logger *log.Logger,
) (*PubsubInfo, error) {
	tfVarArgs := tfVarMapToArgs(projectID, tfVars)
	cmd := initCommand(ctx, projectID)
	cmd.Args = append(cmd.Args, tfVarArgs...)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		return nil, err
	}

	logger.Printf("Running %s with image: %s\n", tfDir, tfVars["image"])

	// Create new terraform workspace
	cmd = exec.CommandContext(ctx, "terraform", "workspace", "new", testRunID)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		// try to switch to workspace if it already exists
		cmd = exec.CommandContext(ctx, "terraform", "workspace", "select", testRunID)
		cmd.Dir = tfDir

		if err := runWithOutput(cmd, logger); err != nil {
			return nil, err
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
	if err := runApply(cmd, logger); err != nil {
		return nil, err
	}

	// Run terraform output
	cmd = exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = tfDir
	out, err := cmd.Output()
	if err != nil {
		logger.Println(err)
		return nil, err
	}

	tfOutput := &tfOutput{}
	if err := json.Unmarshal(out, tfOutput); err != nil {
		return nil, err
	}
	return &tfOutput.PubsubInfoWrapper.Value, nil
}

func ApplyPersistent(
	ctx context.Context,
	projectID string,
	autoApprove bool,
	logger *log.Logger,
) error {
	return applyPersistent(ctx, projectID, autoApprove, logger, tfPersistentDir)
}

func ApplyPersistentCollector(
	ctx context.Context,
	projectID string,
	autoApprove bool,
	logger *log.Logger,
) error {
	return applyPersistent(ctx, projectID, autoApprove, logger, tfPersistentCollectorDir)
}

// Create persistent resources (in tf/persistent) that are used across tests. No
// cleanup is required
func applyPersistent(
	ctx context.Context,
	projectID string,
	autoApprove bool,
	logger *log.Logger,
	persistentDir string,
) error {
	logger.Println("Applying any changes to persistent resources")
	// Run terraform init
	cmd := initCommand(ctx, projectID)
	cmd.Dir = persistentDir
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

// gcsWorkspaceExists checks whether the Terraform workspace still exists by
// reading the GCS backend state object. Note that GCS maps any HTTP 404
// (including a nonexistent bucket) to storage.ErrObjectNotExist.
func gcsWorkspaceExists(ctx context.Context, projectID, testRunID string, opts ...option.ClientOption) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, gcsCheckTimeout)
	defer cancel()

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return false, fmt.Errorf("creating storage client: %w", err)
	}
	defer client.Close()

	bucket := tfStateBucket(projectID)
	object := fmt.Sprintf("%s/%s.tfstate", tfStatePrefix, testRunID)
	_, err = client.Bucket(bucket).Object(object).Attrs(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking terraform state object gs://%s/%s: %w", bucket, object, err)
	}
	return true, nil
}

// CleanupTf destroys resources and deletes the Terraform workspace. If the
// workspace does not exist, it is a no-op and returns nil.
func CleanupTf(
	ctx context.Context,
	projectID string,
	testRunID string,
	logger *log.Logger,
) error {
	if strings.TrimSpace(testRunID) == "" {
		return errors.New("testRunID is required for cleanup")
	}
	if testRunID == "default" {
		return errors.New("refusing to clean up default terraform workspace")
	}

	exists, err := gcsWorkspaceExists(ctx, projectID, testRunID)
	if err != nil {
		logger.Printf("error checking if terraform workspace %s exists: %v", testRunID, err)
		return err
	}
	if !exists {
		logger.Printf("Terraform workspace %s does not exist; skipping cleanup", testRunID)
		return nil
	}

	const tfDir = "tf/destroy"
	tfVarArgs := []string{fmt.Sprintf("-var=project_id=%s", projectID)}
	cmd := initCommand(ctx, projectID)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		logger.Printf("error cleaning up terraform (init) in %s: %v", tfDir, err)
		return err
	}

	// Switch to target workspace
	cmd = exec.CommandContext(ctx, "terraform", "workspace", "select", testRunID)
	cmd.Dir = tfDir
	if err := runWithOutput(cmd, logger); err != nil {
		logger.Printf("error cleaning up terraform (workspace select %s) in %s: %v", testRunID, tfDir, err)
		return err
	}

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
		logger.Printf("error cleaning up terraform (destroy) in %s: %v", tfDir, err)
		return err
	}

	deleteWorkspace(ctx, testRunID, tfDir, logger)

	return nil
}
