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

module "python" {
  source     = "../modules/repo-ci-triggers"
  project_id = var.project_id
  repository = "opentelemetry-operations-python"
  run_on     = ["local", "gce", "gke", "gae", "gae-standard", "cloud-run", "cloud-functions-gen2"]
}

module "java" {
  source     = "../modules/repo-ci-triggers"
  project_id = var.project_id
  repository = "opentelemetry-operations-java"
  run_on     = ["local", "gce", "gke", "gae", "cloud-run", "cloud-functions-gen2"]
}

module "js" {
  source     = "../modules/repo-ci-triggers"
  project_id = var.project_id
  repository = "opentelemetry-operations-js"
  run_on     = ["local", "gce", "gke", "gae", "gae-standard", "cloud-run", "cloud-functions-gen2"]
}

module "go" {
  source     = "../modules/repo-ci-triggers"
  project_id = var.project_id
  repository = "opentelemetry-operations-go"
  run_on     = ["local", "gce", "gke", "gae", "gae-standard", "cloud-run", "cloud-functions-gen2"]
}

resource "google_cloudbuild_trigger" "global_cleanup" {
  name        = "global-e2e-cleanup"
  description = "Global cleanup for E2E tests triggered by Pub/Sub"

  pubsub_config {
    topic = "projects/${var.project_id}/topics/cloud-builds"
  }

  filter = "\"terraform-resources\" in body.message.data.tags && (body.message.data.status == \"SUCCESS\" || body.message.data.status == \"FAILURE\")"

  git_file_source {
    path       = "cloudbuild-cleanup.yaml"
    uri        = "https://github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing"
    revision   = "refs/heads/main"
    repo_type  = "GITHUB"
  }

  substitutions = {
    _TEST_RUN_ID     = "$(body.message.data.id)"
    _E2E_ENVIRONMENT = "$(body.message.data.substitutions._E2E_ENVIRONMENT)"
  }
}
