# Copyright 2024 Google LLC
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

// The e2e test runner in this repo is published to this repo as are the instrumented test
// servers built during CI in each tested repo
resource "google_artifact_registry_repository" "e2e_testing" {
  location      = "us-central1"
  repository_id = "e2e-testing"
  description   = "Docker repo containing images for e2e testing"
  format        = "DOCKER"
}
