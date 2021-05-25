# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

// This GKE cluster is used across GKE test runs for running pods, because GKE
// cluster creation/destruction take ~3 minutes each
resource "google_container_cluster" "default" {
  # The terraform workspace will be given a random name (test run id) which we
  # can use to get unique resource names.
  name     = local.gke_cluster_name
  location = local.gke_cluster_location

  initial_node_count = 1
  node_config {
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}
