// Copyright 2024 Google LLC
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
	"log"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/setuptf"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing/testclient"
)

const gaeStandardTfDir string = "tf/gae-standard"

func SetupGaeStandard(
	ctx context.Context,
	args *Args,
	logger *log.Logger,
) (*testclient.Client, Cleanup, error) {
	pubsubInfo, cleanupTf, err := setuptf.SetupTf(
		ctx,
		args.ProjectID,
		args.TestRunID,
		gaeStandardTfDir,
		map[string]string{
			"runtime":   args.GaeStandard.Runtime,
			"appsource": args.GaeStandard.AppSource,
		},
		logger,
	)
	if err != nil {
		return nil, cleanupTf, err
	}

	client, err := testclient.New(ctx, args.ProjectID, pubsubInfo)
	return client, cleanupTf, err
}
