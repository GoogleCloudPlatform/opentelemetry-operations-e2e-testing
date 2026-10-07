// Copyright 2026 Google LLC
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
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"testing"
)

// fakeApply returns a command that stands in for terraform apply: it prints
// stderr to stderr and exits with exitCode.
func fakeApply(stderr string, exitCode int) *exec.Cmd {
	cmd := exec.Command("sh", "-c", `printf '%s\n' "$FAKE_STDERR" >&2; exit "$FAKE_EXIT_CODE"`)
	cmd.Env = append(os.Environ(), "FAKE_STDERR="+stderr, fmt.Sprintf("FAKE_EXIT_CODE=%d", exitCode))
	return cmd
}

func TestRunApply(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	for _, tc := range []struct {
		name         string
		stderr       string
		exitCode     int
		wantErr      bool
		wantStockout bool
	}{
		{
			name: "success",
		},
		{
			name:         "c4a stockout",
			stderr:       "Error: Error waiting for instance to create: The zone 'projects/my-project/zones/us-central1-a' does not have enough resources available to fulfill the request. 'NULL:0/NULL:0/NULL:0 (state:STOCKOUT, sub-state:STOCKOUT, resource type:compute)'. A c4a-standard-1 VM instance is currently unavailable in the us-central1-a zone.",
			exitCode:     1,
			wantErr:      true,
			wantStockout: true,
		},
		{
			name:         "zone resource pool exhausted",
			stderr:       "Error: Error creating instance: googleapi: Error 503: The zone 'projects/my-project/zones/us-east4-a' does not have enough resources available to fulfill the request. Try a different zone, or try again later., ZONE_RESOURCE_POOL_EXHAUSTED",
			exitCode:     1,
			wantErr:      true,
			wantStockout: true,
		},
		{
			name:         "vm type currently unavailable",
			stderr:       "Error: Error waiting for instance to create: A c4a-standard-1 VM instance is currently unavailable in the us-east1-b zone. Capacity changes frequently, so try your request in a different zone, with a different VM hardware configuration, or at a later time.",
			exitCode:     1,
			wantErr:      true,
			wantStockout: true,
		},
		{
			name:         "error code only",
			stderr:       "Error: Error creating instance: googleapi: Error 503: ZONE_RESOURCE_POOL_EXHAUSTED_WITH_DETAILS",
			exitCode:     1,
			wantErr:      true,
			wantStockout: true,
		},
		{
			name:         "stockout state only",
			stderr:       "Error: Error waiting for instance to create: 'NULL:0/NULL:0/NULL:0 (state:STOCKOUT, sub-state:STOCKOUT, resource type:compute)'",
			exitCode:     1,
			wantErr:      true,
			wantStockout: true,
		},
		{
			name:     "other failure",
			stderr:   "Error: Error creating instance: googleapi: Error 403: Required 'compute.instances.create' permission for 'projects/my-project/zones/us-central1-a/instances/e2etest-123', forbidden",
			exitCode: 1,
			wantErr:  true,
		},
		{
			name:     "quota exceeded is not a stockout",
			stderr:   "Error: Error creating instance: googleapi: Error 403: Quota 'C4A_CPUS' exceeded.  Limit: 8.0 in region us-east1., quotaExceeded",
			exitCode: 1,
			wantErr:  true,
		},
		{
			name:     "service unavailable is not a stockout",
			stderr:   "Error: Error creating instance: googleapi: Error 503: The service is currently unavailable., backendError",
			exitCode: 1,
			wantErr:  true,
		},
		{
			name:   "stockout text on success is ignored",
			stderr: "Warning: an earlier attempt said ZONE_RESOURCE_POOL_EXHAUSTED",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := runApply(fakeApply(tc.stderr, tc.exitCode), logger)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("runApply() error = %v, want error: %v", err, tc.wantErr)
			}
			if gotStockout := errors.Is(err, ErrStockout); gotStockout != tc.wantStockout {
				t.Errorf("errors.Is(err, ErrStockout) = %v, want %v (err: %v)", gotStockout, tc.wantStockout, err)
			}
		})
	}
}
